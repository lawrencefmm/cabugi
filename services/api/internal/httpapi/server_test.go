package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
)

type stubVerifier struct {
	principal auth.Principal
	err       error
}

func (verifier stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return verifier.principal, verifier.err
}

func TestHealthzHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", recorder.Code, http.StatusOK)
	}

	if got := strings.TrimSpace(recorder.Body.String()); got != `{"status":"ok"}` {
		t.Fatalf("GET /healthz body = %q, want %q", got, `{"status":"ok"}`)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("GET /healthz content type = %q, want %q", got, "application/json")
	}
}

func TestOpenAPIHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/openapi/v1.yaml", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /openapi/v1.yaml status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "openapi: 3.1.0") {
		t.Fatalf("GET /openapi/v1.yaml body did not include OpenAPI version header")
	}
	if !strings.Contains(body, "/v1/me:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/me path")
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("GET /openapi/v1.yaml content type = %q, want %q", got, "application/yaml")
	}
}

func TestProtectedRouteRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{err: auth.ErrInvalidToken}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me with invalid token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteAcceptsVerifiedToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/me with valid token status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"subject":"user_123"}` {
		t.Fatalf("GET /v1/me body = %q, want %q", got, `{"subject":"user_123"}`)
	}
}

func TestProtectedRouteReturnsServiceUnavailableWhenVerifierIsDisabled(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer configured-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{err: auth.ErrVerifierNotConfigured}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /v1/me with disabled verifier status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
