package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
)

type stubVerifier struct {
	principal auth.Principal
	err       error
}

type stubProblemStore struct {
	summaries []problems.PublishedProblemSummary
	detail    problems.PublishedProblemDetail
	listErr   error
	getErr    error
}

func (verifier stubVerifier) Verify(context.Context, string) (auth.Principal, error) {
	return verifier.principal, verifier.err
}

func (store stubProblemStore) ListPublishedProblems(context.Context) ([]problems.PublishedProblemSummary, error) {
	return store.summaries, store.listErr
}

func (store stubProblemStore) GetPublishedProblemBySlug(context.Context, string) (problems.PublishedProblemDetail, error) {
	return store.detail, store.getErr
}

func TestHealthzHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, nil).ServeHTTP(recorder, request)

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

	NewMux(nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /openapi/v1.yaml status = %d, want %d", recorder.Code, http.StatusOK)
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "openapi: 3.1.0") {
		t.Fatalf("GET /openapi/v1.yaml body did not include OpenAPI version header")
	}
	if !strings.Contains(body, "/v1/problems/{slug}:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/problems/{slug} path")
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("GET /openapi/v1.yaml content type = %q, want %q", got, "application/yaml")
	}
}

func TestProtectedRouteRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{err: auth.ErrInvalidToken}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me with invalid token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteAcceptsVerifiedToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil).ServeHTTP(recorder, request)

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

	NewMux(stubVerifier{err: auth.ErrVerifierNotConfigured}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /v1/me with disabled verifier status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestListPublishedProblemsReturnsPublishedProblems(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems", nil)
	recorder := httptest.NewRecorder()

	store := stubProblemStore{summaries: []problems.PublishedProblemSummary{
		{Slug: "a-plus-b", Title: "A + B", TimeLimitMs: 1000, MemoryLimitMB: 256},
		{Slug: "two-sum", Title: "Two Sum", TimeLimitMs: 1000, MemoryLimitMB: 256},
	}}
	NewMux(nil, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/problems status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Problems []problems.PublishedProblemSummary `json:"problems"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Problems) != 2 {
		t.Fatalf("GET /v1/problems returned %d problems, want 2", len(response.Problems))
	}
}

func TestGetPublishedProblemBySlugReturnsProblem(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems/two-sum", nil)
	recorder := httptest.NewRecorder()

	store := stubProblemStore{detail: problems.PublishedProblemDetail{
		Slug: "two-sum", Title: "Two Sum", StatementMarkdown: "Solve it", InputMarkdown: "Input", OutputMarkdown: "Output", ConstraintsMarkdown: "Constraints", NotesMarkdown: "Notes", TimeLimitMs: 1000, MemoryLimitMB: 256,
	}}
	NewMux(nil, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/problems/{slug} status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestGetPublishedProblemBySlugReturnsNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems/missing", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, stubProblemStore{getErr: problems.ErrNotFound}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /v1/problems/{slug} missing status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
