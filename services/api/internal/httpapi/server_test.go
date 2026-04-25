package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"mime/multipart"
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
	summaries           []problems.PublishedProblemSummary
	detail              problems.PublishedProblemDetail
	draft               problems.DraftProblem
	listErr             error
	getErr              error
	createErr           error
	getDraftErr         error
	updateErr           error
	submitForReviewErr  error
	listModerationErr   error
	decisionErr         error
	createDraftInput    problems.CreateDraftInput
	updateDraftInput    problems.UpdateDraftInput
	getDraftSlug        string
	getDraftActorUserID string
	getDraftAllowStaff  bool
	submitSlug          string
	submitOwnerUserID   string
	moderationItems     []problems.ModerationQueueItem
	decisionSlug        string
	decisionReviewerID  string
	decisionValue       problems.ModerationDecision
	decisionNotes       string
}

type stubUserStore struct {
	user        users.User
	err         error
	hasRole     bool
	hasRoleErr  error
	checkedUser string
}

type stubSubmissionStore struct {
	submissions []submissions.Summary
	created     submissions.Summary
	submission  submissions.Detail
	queueDepth  int
	queueErr    error
	createErr   error
	listErr     error
	getErr      error
	createInput submissions.CreateInput
}

type stubBundleValidator struct {
	err       error
	key       string
	checksum  string
	uploadErr error
	uploaded  problems.UploadedBundle
	userID    string
	fileName  string
	contents  []byte
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

func (store *stubProblemStore) CreateDraft(_ context.Context, input problems.CreateDraftInput) (problems.DraftProblem, error) {
	store.createDraftInput = input
	return store.draft, store.createErr
}

func (store *stubProblemStore) GetDraftBySlug(_ context.Context, slug string, actorUserID string, allowStaff bool) (problems.DraftProblem, error) {
	store.getDraftSlug = slug
	store.getDraftActorUserID = actorUserID
	store.getDraftAllowStaff = allowStaff
	return store.draft, store.getDraftErr
}

func (store *stubProblemStore) UpdateDraft(_ context.Context, input problems.UpdateDraftInput) (problems.DraftProblem, error) {
	store.updateDraftInput = input
	return store.draft, store.updateErr
}

func (store *stubProblemStore) SubmitDraftForReview(_ context.Context, slug string, ownerUserID string) (problems.DraftProblem, error) {
	store.submitSlug = slug
	store.submitOwnerUserID = ownerUserID
	return store.draft, store.submitForReviewErr
}

func (store stubProblemStore) ListDraftsInReview(context.Context) ([]problems.ModerationQueueItem, error) {
	return store.moderationItems, store.listModerationErr
}

func (store *stubProblemStore) ApplyModerationDecision(_ context.Context, slug string, reviewerUserID string, decision problems.ModerationDecision, moderationNotes string) (problems.DraftProblem, error) {
	store.decisionSlug = slug
	store.decisionReviewerID = reviewerUserID
	store.decisionValue = decision
	store.decisionNotes = moderationNotes
	return store.draft, store.decisionErr
}

func (store stubUserStore) GetOrCreateBySubject(context.Context, string) (users.User, error) {
	return store.user, store.err
}

func (store stubUserStore) HasAnyRole(context.Context, string, ...users.Role) (bool, error) {
	return store.hasRole, store.hasRoleErr
}

func (store *stubSubmissionStore) CreateSubmission(_ context.Context, input submissions.CreateInput) (submissions.Summary, error) {
	store.createInput = input
	return store.created, store.createErr
}

func (store *stubSubmissionStore) ListSubmissions(context.Context, string) ([]submissions.Summary, error) {
	return store.submissions, store.listErr
}

func (store *stubSubmissionStore) GetSubmissionByID(context.Context, string, string) (submissions.Detail, error) {
	return store.submission, store.getErr
}

func (store *stubSubmissionStore) QueueDepth(context.Context) (int, error) {
	if store.queueErr != nil {
		return 0, store.queueErr
	}

	return store.queueDepth, nil
}

func (validator *stubBundleValidator) ValidateBundle(_ context.Context, key string, checksum string) error {
	validator.key = key
	validator.checksum = checksum
	return validator.err
}

func (validator *stubBundleValidator) UploadBundle(_ context.Context, userID string, fileName string, contents []byte) (problems.UploadedBundle, error) {
	validator.userID = userID
	validator.fileName = fileName
	validator.contents = contents
	return validator.uploaded, validator.uploadErr
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

func TestReadyzHandlerReportsConfiguredDependencies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{}, &stubSubmissionStore{}, &stubBundleValidator{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /readyz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ready"`) {
		t.Fatalf("GET /readyz body = %q, want ready status", recorder.Body.String())
	}
}

func TestReadyzHandlerReportsMissingDependencies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	NewMux(nil, nil, nil, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"auth":false`) || !strings.Contains(body, `"database":false`) || !strings.Contains(body, `"hiddenTestBundleValidation":false`) {
		t.Fatalf("GET /readyz body = %q, want all dependency readiness flags", body)
	}
}

func TestCORSAllowsConfiguredOrigin(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/problems", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	recorder := httptest.NewRecorder()

	withCORS(NewMux(nil, &stubProblemStore{}, nil, nil), []string{"http://localhost:3000"}).ServeHTTP(recorder, request)

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
	if got := recorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPatch) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want PATCH to be allowed", got)
	}
}

func TestNewServerSetsHTTPTimeouts(t *testing.T) {
	server := NewServer("127.0.0.1:8080", slog.Default(), stubVerifier{}, &stubProblemStore{}, stubUserStore{}, &stubSubmissionStore{}, nil, &stubBundleValidator{})

	if server.ReadTimeout != defaultReadTimeout {
		t.Fatalf("ReadTimeout = %s, want %s", server.ReadTimeout, defaultReadTimeout)
	}
	if server.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", server.ReadHeaderTimeout, defaultReadHeaderTimeout)
	}
	if server.WriteTimeout != defaultWriteTimeout {
		t.Fatalf("WriteTimeout = %s, want %s", server.WriteTimeout, defaultWriteTimeout)
	}
	if server.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("IdleTimeout = %s, want %s", server.IdleTimeout, defaultIdleTimeout)
	}
	if server.MaxHeaderBytes != defaultMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", server.MaxHeaderBytes, defaultMaxHeaderBytes)
	}
}

func TestMetricszHandlerReportsRequestsAndSubmissionQueueDepth(t *testing.T) {
	store := &stubSubmissionStore{queueDepth: 2}
	mux := NewMux(nil, &stubProblemStore{}, stubUserStore{}, store, &stubBundleValidator{})

	request := httptest.NewRequest(http.MethodGet, "/metricsz", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metricsz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"submissionQueue":{"depth":2}`) || !strings.Contains(body, `"requests"`) {
		t.Fatalf("GET /metricsz body = %q, want request metrics and queue depth", body)
	}
}

func TestMetricszHandlerReportsUnavailableQueueDepth(t *testing.T) {
	store := &stubSubmissionStore{queueErr: errors.New("db unavailable")}
	mux := NewMux(nil, &stubProblemStore{}, stubUserStore{}, store, &stubBundleValidator{})

	request := httptest.NewRequest(http.MethodGet, "/metricsz", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /metricsz status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"submissionQueue":{"depth":-1}`) {
		t.Fatalf("GET /metricsz body = %q, want unavailable queue depth sentinel", body)
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
	if !strings.Contains(body, "/readyz:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /readyz path")
	}
	if !strings.Contains(body, "/metricsz:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /metricsz path")
	}
	if !strings.Contains(body, "/v1/problems/{slug}:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/problems/{slug} path")
	}
	if !strings.Contains(body, "/v1/problem-drafts/{slug}:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/problem-drafts/{slug} path")
	}
	if !strings.Contains(body, "/v1/problem-drafts/{slug}/submit-for-review:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/problem-drafts/{slug}/submit-for-review path")
	}
	if !strings.Contains(body, "/v1/problem-drafts/hidden-test-bundles:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/problem-drafts/hidden-test-bundles path")
	}
	if !strings.Contains(body, "/v1/moderation/problem-drafts:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/moderation/problem-drafts path")
	}
	if !strings.Contains(body, "/v1/moderation/problem-drafts/{slug}/decision:") {
		t.Fatalf("GET /openapi/v1.yaml body did not include /v1/moderation/problem-drafts/{slug}/decision path")
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/yaml" {
		t.Fatalf("GET /openapi/v1.yaml content type = %q, want %q", got, "application/yaml")
	}
}

func TestUploadProblemDraftBundleRejectsMissingFile(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts/hidden-test-bundles", &bytes.Buffer{})
	request.Header.Set("Authorization", "Bearer good-token")
	request.Header.Set("Content-Type", "multipart/form-data; boundary=test-boundary")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, &stubBundleValidator{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/problem-drafts/hidden-test-bundles missing file status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestUploadProblemDraftBundleRejectsInvalidBundleContents(t *testing.T) {
	request := newMultipartUploadRequest(t, []byte(`{"cases":[]}`), "bundle.json")
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{uploadErr: problems.ErrInvalidBundleContents}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/problem-drafts/hidden-test-bundles invalid upload status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if validator.userID != "user-id" || validator.fileName != "bundle.json" {
		t.Fatalf("UploadBundle() received unexpected metadata: %#v", validator)
	}
}

func TestUploadProblemDraftBundleReturnsUploadedMetadata(t *testing.T) {
	request := newMultipartUploadRequest(t, []byte(`{"cases":[{"input":"1\n","expectedOutput":"2\n"}]}`), "bundle.json")
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{uploaded: problems.UploadedBundle{Key: "problem-drafts/user-id/bundle.json", SHA256: strings.Repeat("a", 64)}}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /v1/problem-drafts/hidden-test-bundles status = %d, want %d", recorder.Code, http.StatusCreated)
	}

	var response problems.UploadedBundle
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Key != validator.uploaded.Key || response.SHA256 != validator.uploaded.SHA256 {
		t.Fatalf("POST /v1/problem-drafts/hidden-test-bundles returned unexpected response: %#v", response)
	}
}

func newMultipartUploadRequest(t *testing.T, contents []byte, fileName string) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("bundle", fileName)
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts/hidden-test-bundles", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
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
	verifier, err := auth.NewClerkVerifier(auth.ClerkConfig{
		PublicKeyPEM:     string(publicKeyPEM),
		Issuer:           "https://clerk.example.com",
		AllowedAudiences: []string{"cabugi-web"},
	})
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user_456",
		"iss": "https://clerk.example.com",
		"aud": []string{"cabugi-web"},
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

func TestProtectedRouteAcceptsSessionCookieToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.AddCookie(&http.Cookie{Name: "__session", Value: "cookie-token"})
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/me with session cookie status = %d, want %d", recorder.Code, http.StatusOK)
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

	store := &stubProblemStore{summaries: []problems.PublishedProblemSummary{
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

	store := &stubProblemStore{detail: problems.PublishedProblemDetail{
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

	NewMux(nil, &stubProblemStore{getErr: problems.ErrNotFound}, nil, nil).ServeHTTP(recorder, request)

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

func TestCreateProblemDraftRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","timeLimitMs":1000,"memoryLimitMb":256}`))
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("POST /v1/problem-drafts without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestCreateProblemDraftCreatesInitialDraft(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "draft", Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256}}
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /v1/problem-drafts status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if store.createDraftInput.UserID != "user-id" || store.createDraftInput.Slug != "two-sum-user" {
		t.Fatalf("POST /v1/problem-drafts stored unexpected create input: %#v", store.createDraftInput)
	}
}

func TestGetProblemDraftReturnsOwnedDraft(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "draft", Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256}}
	request := httptest.NewRequest(http.MethodGet, "/v1/problem-drafts/two-sum-user", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/problem-drafts/{slug} status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if store.getDraftSlug != "two-sum-user" || store.getDraftActorUserID != "user-id" || store.getDraftAllowStaff {
		t.Fatalf("GET /v1/problem-drafts/{slug} used unexpected draft lookup inputs: %#v", store)
	}
}

func TestUpdateProblemDraftReturnsNotFoundForUnownedDraft(t *testing.T) {
	request := httptest.NewRequest(http.MethodPatch, "/v1/problem-drafts/two-sum-user", strings.NewReader(`{"title":"Updated Title","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{updateErr: problems.ErrDraftNotFound}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("PATCH /v1/problem-drafts/{slug} unowned status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestUpdateProblemDraftAllowsStaffToModifyAnotherUsersDraft(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "draft", Title: "Updated Title", TimeLimitMs: 1000, MemoryLimitMB: 256}}
	request := httptest.NewRequest(http.MethodPatch, "/v1/problem-drafts/two-sum-user", strings.NewReader(`{"title":"Updated Title","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}, hasRole: true}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("PATCH /v1/problem-drafts/{slug} staff status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !store.updateDraftInput.AllowStaff || store.updateDraftInput.ActorUserID != "user-id" {
		t.Fatalf("PATCH /v1/problem-drafts/{slug} stored unexpected update input: %#v", store.updateDraftInput)
	}
}

func TestCreateProblemDraftRejectsPartialHiddenBundleMetadata(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256,"hiddenTestBundleKey":"bundles/two-sum.json"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/problem-drafts partial bundle metadata status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if validator.key != "" || validator.checksum != "" {
		t.Fatalf("validator should not run for invalid metadata, got key=%q checksum=%q", validator.key, validator.checksum)
	}
}

func TestCreateProblemDraftRejectsMissingHiddenBundleObject(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256,"hiddenTestBundleKey":"bundles/two-sum.json","hiddenTestBundleSha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{err: problems.ErrBundleNotFound}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/problem-drafts missing bundle object status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if validator.key != "bundles/two-sum.json" {
		t.Fatalf("validator received key %q, want %q", validator.key, "bundles/two-sum.json")
	}
}

func TestCreateProblemDraftStoresValidatedHiddenBundleMetadata(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "draft", Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256, HiddenTestBundleKey: "bundles/two-sum.json", HiddenTestBundleSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256,"hiddenTestBundleKey":"bundles/two-sum.json","hiddenTestBundleSha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /v1/problem-drafts with bundle metadata status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if store.createDraftInput.HiddenTestBundleKey != "bundles/two-sum.json" || store.createDraftInput.HiddenTestBundleSHA256 == "" {
		t.Fatalf("POST /v1/problem-drafts stored unexpected bundle metadata: %#v", store.createDraftInput)
	}
}

func TestCreateProblemDraftRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts", strings.NewReader(`{"slug":"two-sum-user","title":"Two Sum User","statementMarkdown":"`+strings.Repeat("a", maxProblemDraftBodyBytes)+`","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	store := &stubProblemStore{}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, &stubBundleValidator{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/problem-drafts oversized body status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if store.createDraftInput.Slug != "" {
		t.Fatalf("create draft should not run for oversized request, got %#v", store.createDraftInput)
	}
}

func TestUpdateProblemDraftPreservesExistingHiddenBundleMetadataWhenOmitted(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "draft", Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256, HiddenTestBundleKey: "bundles/two-sum.json", HiddenTestBundleSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}
	request := httptest.NewRequest(http.MethodPatch, "/v1/problem-drafts/two-sum-user", strings.NewReader(`{"title":"Updated Title","statementMarkdown":"Solve it","inputMarkdown":"Input","outputMarkdown":"Output","constraintsMarkdown":"Constraints","notesMarkdown":"Notes","timeLimitMs":1000,"memoryLimitMb":256}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	validator := &stubBundleValidator{}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil, validator).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("PATCH /v1/problem-drafts/{slug} preserve bundle metadata status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if store.updateDraftInput.HiddenTestBundleKey != "bundles/two-sum.json" || store.updateDraftInput.HiddenTestBundleSHA256 == "" {
		t.Fatalf("PATCH /v1/problem-drafts/{slug} did not preserve bundle metadata: %#v", store.updateDraftInput)
	}
	if validator.key != "" {
		t.Fatalf("validator should not run when bundle metadata is omitted, got key=%q", validator.key)
	}
}

func TestSubmitProblemDraftForReviewRejectsMissingToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts/two-sum-user/submit-for-review", nil)
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("POST /v1/problem-drafts/{slug}/submit-for-review without token status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestSubmitProblemDraftForReviewTransitionsOwnedDraft(t *testing.T) {
	store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: "in_review", Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256}}
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts/two-sum-user/submit-for-review", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /v1/problem-drafts/{slug}/submit-for-review status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if store.submitSlug != "two-sum-user" || store.submitOwnerUserID != "user-id" {
		t.Fatalf("POST /v1/problem-drafts/{slug}/submit-for-review used unexpected submit inputs: %#v", store)
	}
}

func TestSubmitProblemDraftForReviewRejectsInvalidLifecycleTransition(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/problem-drafts/two-sum-user/submit-for-review", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{submitForReviewErr: problems.ErrInvalidLifecycleTransition}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("POST /v1/problem-drafts/{slug}/submit-for-review invalid transition status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestModerationQueueRejectsNonModerators(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/moderation/problem-drafts", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}, hasRole: false}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("GET /v1/moderation/problem-drafts non-moderator status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestModerationQueueReturnsInReviewDraftsForModerators(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/moderation/problem-drafts", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	store := &stubProblemStore{moderationItems: []problems.ModerationQueueItem{{Slug: "two-sum-user", VersionNumber: 1, Title: "Two Sum User", SubmittedForReviewAt: time.Now().UTC()}}}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "moderator-id", Subject: "user_123", Handle: "moderator_abcd", DisplayName: "Moderator abcd"}, hasRole: true}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/moderation/problem-drafts status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Drafts []problems.ModerationQueueItem `json:"drafts"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.Drafts) != 1 || response.Drafts[0].Slug != "two-sum-user" {
		t.Fatalf("GET /v1/moderation/problem-drafts returned unexpected drafts: %#v", response.Drafts)
	}
}

func TestModerationDecisionAllowsApproveRejectAndRequestChanges(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision string
		status   string
	}{
		{name: "approve", decision: "approve", status: "published"},
		{name: "reject", decision: "reject", status: "archived"},
		{name: "request changes", decision: "request_changes", status: "draft"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/moderation/problem-drafts/two-sum-user/decision", strings.NewReader(`{"decision":"`+tc.decision+`","moderationNotes":"looks good"}`))
			request.Header.Set("Authorization", "Bearer good-token")
			recorder := httptest.NewRecorder()

			store := &stubProblemStore{draft: problems.DraftProblem{Slug: "two-sum-user", VersionNumber: 1, LifecycleStatus: tc.status, Title: "Two Sum User", TimeLimitMs: 1000, MemoryLimitMB: 256}}
			NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, store, stubUserStore{user: users.User{ID: "moderator-id", Subject: "user_123", Handle: "moderator_abcd", DisplayName: "Moderator abcd"}, hasRole: true}, nil).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("POST /v1/moderation/problem-drafts/{slug}/decision status = %d, want %d", recorder.Code, http.StatusOK)
			}
			if store.decisionSlug != "two-sum-user" || store.decisionReviewerID != "moderator-id" || string(store.decisionValue) != tc.decision {
				t.Fatalf("POST /v1/moderation/problem-drafts/{slug}/decision used unexpected input: %#v", store)
			}
		})
	}
}

func TestModerationDecisionRejectsNonModerators(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/moderation/problem-drafts/two-sum-user/decision", strings.NewReader(`{"decision":"approve","moderationNotes":"looks good"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, &stubProblemStore{}, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}, hasRole: false}, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("POST /v1/moderation/problem-drafts/{slug}/decision non-moderator status = %d, want %d", recorder.Code, http.StatusForbidden)
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

func TestCreateSubmissionRejectsOversizedBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/submissions", strings.NewReader(`{"problemSlug":"two-sum","language":"cpp17","sourceCode":"`+strings.Repeat("a", maxSubmissionBodyBytes)+`"}`))
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	store := &stubSubmissionStore{}
	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, store).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/submissions oversized body status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if store.createInput.ProblemSlug != "" {
		t.Fatalf("create submission should not run for oversized request, got %#v", store.createInput)
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
	store := &stubSubmissionStore{created: submissions.Summary{ID: "submission-id", ProblemSlug: "two-sum", Language: "cpp17", Status: "queued"}}
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

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{submission: submissions.Detail{ID: "submission-id", ProblemSlug: "two-sum", Language: "cpp17", Status: "wrong_answer", TotalTests: 3, PassedTests: 2, CompileOutputExcerpt: "", Results: []submissions.Result{{TestIndex: 2, Verdict: "wrong_answer", ExecutionTimeMS: 13}}}}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/submissions/{id} status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response submissions.Detail
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.TotalTests != 3 || response.PassedTests != 2 {
		t.Fatalf("GET /v1/submissions/{id} returned unexpected aggregates: %#v", response)
	}
	if len(response.Results) != 1 || response.Results[0].Verdict != "wrong_answer" {
		t.Fatalf("GET /v1/submissions/{id} returned unexpected results: %#v", response.Results)
	}
}

func TestGetSubmissionReturnsCompileOutputExcerpt(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/submissions/submission-id", nil)
	request.Header.Set("Authorization", "Bearer good-token")
	recorder := httptest.NewRecorder()

	NewMux(stubVerifier{principal: auth.Principal{Subject: "user_123"}}, nil, stubUserStore{user: users.User{ID: "user-id", Subject: "user_123", Handle: "user_abcd", DisplayName: "User abcd"}}, &stubSubmissionStore{submission: submissions.Detail{ID: "submission-id", ProblemSlug: "broken-solution", Language: "cpp17", Status: "compile_error", TotalTests: 0, PassedTests: 0, CompileOutputExcerpt: "main.cpp:1: error: expected ';'", Results: []submissions.Result{}}}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /v1/submissions/{id} status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response submissions.Detail
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.CompileOutputExcerpt != "main.cpp:1: error: expected ';'" {
		t.Fatalf("GET /v1/submissions/{id} returned unexpected compile output: %#v", response)
	}
}
