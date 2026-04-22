package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
	"github.com/lawrencefmm/cabugi/services/api/internal/submissions"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
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

type stubUserStore struct {
	user users.User
	err  error
}

type stubSubmissionStore struct {
	submissions []submissions.Summary
	submission  submissions.Summary
	createErr   error
	listErr     error
	getErr      error
	createInput submissions.CreateInput
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

func (store stubUserStore) GetOrCreateBySubject(context.Context, string) (users.User, error) {
	return store.user, store.err
}

func (store *stubSubmissionStore) CreateSubmission(_ context.Context, input submissions.CreateInput) (submissions.Summary, error) {
	store.createInput = input
	return store.submission, store.createErr
}

func (store *stubSubmissionStore) ListSubmissions(context.Context, string) ([]submissions.Summary, error) {
	return store.submissions, store.listErr
}

func (store *stubSubmissionStore) GetSubmissionByID(context.Context, string, string) (submissions.Summary, error) {
	return store.submission, store.getErr
}

func TestHealthzHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, nil, nil, nil).ServeHTTP(recorder, request)

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

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	withCORS(NewMux(nil, stubProblemStore{}, nil, nil), []string{"http://localhost:3000"}).ServeHTTP(recorder, request)

	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodGet) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want GET to be allowed", got)
	}
	if recorder.Code != http.StatusOK {
		t.Fatalf("CORS wrapped GET /v1/problems status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestCORSHandlesPreflightRequests(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/v1/submissions", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	withCORS(NewMux(nil, nil, nil, nil), []string{"http://localhost:3000"}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS preflight status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}

func TestOpenAPIHandler(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/openapi/v1.yaml", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, nil, nil, nil).ServeHTTP(recorder, request)

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

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteRejectsInvalidToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{err: auth.ErrInvalidToken}, nil, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/me with invalid token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestProtectedRouteAcceptsVerifiedToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/me with valid token status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != `{"id":"user-id","subject":"user_123","handle":"user_abcd","displayName":"User abcd"}` {
		t.Fatalf("GET /v1/me body = %q, want %q", got, `{"id":"user-id","subject":"user_123","handle":"user_abcd","displayName":"User abcd"}`)
	}
}

func TestProtectedRouteAcceptsVerifiedClerkToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKIXPublicKey() error = %v", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	verifier, err := auth.NewClerkVerifier(auth.ClerkConfig{PublicKeyPEM: string(publicKeyPEM)})
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_456",
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	})
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer "+signedToken)
	recorder := httptest.NewRecorder()

	NewMux(verifier, nil, stubUserStore{user: users.User{ID: "user-id-2", Subject: "user_456", Handle: "user_efgh", DisplayName: "User efgh"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/me with verified Clerk token status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestProtectedRouteReturnsServiceUnavailableWhenVerifierIsDisabled(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer configured-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{err: auth.ErrVerifierNotConfigured}, nil, nil, nil).ServeHTTP(recorder, request)

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
	NewMux(nil, store, nil, nil).ServeHTTP(recorder, request)

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
	NewMux(nil, store, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/problems/{slug} status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestGetPublishedProblemBySlugReturnsNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems/missing", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, stubProblemStore{getErr: problems.ErrNotFound}, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /v1/problems/{slug} missing status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestProtectedRouteReturnsServiceUnavailableWhenUserStoreIsDisabled(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer configured-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{err: users.ErrStoreNotConfigured}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /v1/me with disabled user store status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestCreateSubmissionRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions", strings.NewReader(`{"problemSlug":"two-sum","language":"cpp17","sourceCode":"int main() {}"}`))
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("POST /v1/submissions without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestListSubmissionsRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/submissions", nil)
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /v1/submissions without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestListSubmissionsReturnsOwnerHistory(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/submissions", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	store := &stubSubmissionStore{submissions: []submissions.Summary{
		{ID: "submission-2", ProblemSlug: "two-sum", Language: "python", Status: "accepted"},
		{ID: "submission-1", ProblemSlug: "a-plus-b", Language: "cpp17", Status: "wrong_answer"},
	}}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/submissions status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Submissions []submissions.Summary `json:"submissions"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Submissions) != 2 {
		t.Fatalf("GET /v1/submissions returned %d submissions, want 2", len(response.Submissions))
	}
	if response.Submissions[0].ID != "submission-2" {
		t.Fatalf("GET /v1/submissions returned unexpected first item: %#v", response.Submissions[0])
	}
}

func TestCreateSubmissionCreatesQueuedSubmission(t *testing.T) {
	store := &stubSubmissionStore{submission: submissions.Summary{ID: "submission-id", ProblemSlug: "two-sum", Language: "cpp17", Status: "queued"}}
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions", strings.NewReader(`{"problemSlug":"two-sum","language":"cpp17","sourceCode":"int main() {}"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /v1/submissions status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if store.createInput.UserID != "user-id" || store.createInput.ProblemSlug != "two-sum" {
		t.Fatalf("POST /v1/submissions stored unexpected create input: %#v", store.createInput)
	}
}

func TestCreateSubmissionRejectsMissingProblem(t *testing.T) {
	store := &stubSubmissionStore{createErr: submissions.ErrProblemNotFound}
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions", strings.NewReader(`{"problemSlug":"draft-only","language":"cpp17","sourceCode":"int main() {}"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("POST /v1/submissions missing problem status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestGetSubmissionReturnsNotFoundForUnknownID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/submissions/missing-id", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{getErr: submissions.ErrSubmissionNotFound}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("GET /v1/submissions/{id} missing status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestGetSubmissionReturnsSubmissionForOwner(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/submissions/submission-id", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{submission: submissions.Summary{ID: "submission-id", ProblemSlug: "two-sum", Language: "cpp17", Status: "queued"}}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/submissions/{id} status = %d, want %d", recorder.Code, http.StatusOK)
	}
}
