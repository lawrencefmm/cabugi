package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	apiobs "github.com/lawrencefmm/cabugi/services/api/internal/observability"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
	"github.com/lawrencefmm/cabugi/services/api/internal/submissions"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
	openapiasset "github.com/lawrencefmm/cabugi/services/api/openapi"
)

const maxHiddenBundleUploadBytes = 1 << 20

const (
	maxProblemDraftBodyBytes       = 1 << 20
	maxSubmissionBodyBytes         = 256 << 10
	maxModerationDecisionBodyBytes = 64 << 10
	defaultReadTimeout             = 15 * time.Second
	defaultReadHeaderTimeout       = 5 * time.Second
	defaultWriteTimeout            = 30 * time.Second
	defaultIdleTimeout             = 60 * time.Second
	defaultMaxHeaderBytes          = 16 << 10
)

func NewMux(verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, bundleValidators ...problems.BundleValidator) *http.ServeMux {
	return newMux(verifier, problemStore, userStore, submissionStore, apiobs.New(nil), bundleValidators...)
}

func newMux(verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, observer *apiobs.Observer, bundleValidators ...problems.BundleValidator) *http.ServeMux {
	if problemStore == nil {
		problemStore = problems.DisabledStore{}
	}
	if userStore == nil {
		userStore = users.DisabledStore{}
	}
	if submissionStore == nil {
		submissionStore = submissions.DisabledStore{}
	}
	bundleValidator := problems.BundleValidator(problems.DisabledBundleValidator{})
	if len(bundleValidators) > 0 && bundleValidators[0] != nil {
		bundleValidator = bundleValidators[0]
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /readyz", readyzHandler(verifier, problemStore, userStore, submissionStore, bundleValidator))
	mux.HandleFunc("GET /metricsz", observer.MetricsHandler(submissionQueueDepthProvider(submissionStore)))
	mux.HandleFunc("GET /openapi/v1.yaml", openAPIHandler)
	mux.HandleFunc("GET /v1/problems", listPublishedProblemsHandler(problemStore))
	mux.HandleFunc("GET /v1/problems/{slug}", getPublishedProblemHandler(problemStore))
	mux.Handle("POST /v1/problem-drafts/hidden-test-bundles", auth.RequireAuth(verifier, uploadProblemDraftBundleHandler(userStore, bundleValidator)))
	mux.Handle("POST /v1/problem-drafts", auth.RequireAuth(verifier, createProblemDraftHandler(userStore, problemStore, bundleValidator)))
	mux.Handle("GET /v1/problem-drafts/{slug}", auth.RequireAuth(verifier, getProblemDraftHandler(userStore, problemStore)))
	mux.Handle("PATCH /v1/problem-drafts/{slug}", auth.RequireAuth(verifier, updateProblemDraftHandler(userStore, problemStore, bundleValidator)))
	mux.Handle("POST /v1/problem-drafts/{slug}/submit-for-review", auth.RequireAuth(verifier, submitProblemDraftForReviewHandler(userStore, problemStore)))
	mux.Handle("GET /v1/moderation/problem-drafts", auth.RequireAuth(verifier, listModerationProblemDraftsHandler(userStore, problemStore)))
	mux.Handle("POST /v1/moderation/problem-drafts/{slug}/decision", auth.RequireAuth(verifier, applyModerationDecisionHandler(userStore, problemStore)))
	mux.Handle("GET /v1/submissions", auth.RequireAuth(verifier, listSubmissionsHandler(userStore, submissionStore)))
	mux.Handle("POST /v1/submissions", auth.RequireAuth(verifier, createSubmissionHandler(userStore, submissionStore)))
	mux.Handle("GET /v1/submissions/{id}", auth.RequireAuth(verifier, getSubmissionHandler(userStore, submissionStore)))
	mux.Handle("GET /v1/me", auth.RequireAuth(verifier, currentUserHandler(userStore)))

	return mux
}

func NewServer(address string, logger *slog.Logger, verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, allowedOrigins []string, bundleValidators ...problems.BundleValidator) *http.Server {
	observer := apiobs.New(logger)
	return &http.Server{
		Addr:              address,
		Handler:           withCORS(observer.Middleware(newMux(verifier, problemStore, userStore, submissionStore, observer, bundleValidators...)), allowedOrigins),
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}
}

func healthzHandler(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

func readyzHandler(verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, bundleValidator problems.BundleValidator) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		dependencies := map[string]bool{
			"auth":                       authDependencyReady(verifier),
			"database":                   databaseDependencyReady(problemStore, userStore, submissionStore),
			"hiddenTestBundleValidation": bundleValidationDependencyReady(bundleValidator),
		}

		status := http.StatusOK
		state := "ready"
		for _, ready := range dependencies {
			if !ready {
				status = http.StatusServiceUnavailable
				state = "not_ready"
				break
			}
		}

		writeJSON(writer, status, map[string]any{"status": state, "dependencies": dependencies})
	}
}

func openAPIHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/yaml")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(openapiasset.V1)
}

func currentUserHandler(userStore users.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusInternalServerError, "missing_principal")
			return
		}

		user, err := userStore.GetOrCreateBySubject(request.Context(), principal.Subject)
		if err != nil {
			switch {
			case errors.Is(err, users.ErrStoreNotConfigured):
				writeError(writer, http.StatusServiceUnavailable, "user_store_not_configured")
			default:
				writeError(writer, http.StatusInternalServerError, "internal_server_error")
			}
			return
		}

		writeJSON(writer, http.StatusOK, user)
	})
}

func listPublishedProblemsHandler(problemStore problems.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		publishedProblems, err := problemStore.ListPublishedProblems(request.Context())
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{"problems": publishedProblems})
	}
}

func getPublishedProblemHandler(problemStore problems.Store) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		problem, err := problemStore.GetPublishedProblemBySlug(request.Context(), request.PathValue("slug"))
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, problem)
	}
}

func uploadProblemDraftBundleHandler(userStore users.Store, bundleValidator problems.BundleValidator) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := currentUserFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		request.Body = http.MaxBytesReader(writer, request.Body, maxHiddenBundleUploadBytes+1024)
		if err := request.ParseMultipartForm(maxHiddenBundleUploadBytes); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_hidden_test_bundle_upload")
			return
		}

		bundleFile, header, err := request.FormFile("bundle")
		if err != nil {
			writeError(writer, http.StatusBadRequest, "missing_hidden_test_bundle_file")
			return
		}
		defer bundleFile.Close()

		contents, err := io.ReadAll(bundleFile)
		if err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_hidden_test_bundle_upload")
			return
		}

		uploadedBundle, err := bundleValidator.UploadBundle(request.Context(), user.ID, header.Filename, contents)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusCreated, uploadedBundle)
	})
}

func createProblemDraftHandler(userStore users.Store, problemStore problems.Store, bundleValidator problems.BundleValidator) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := currentUserFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		var body struct {
			Slug                string  `json:"slug"`
			Title               string  `json:"title"`
			StatementMarkdown   string  `json:"statementMarkdown"`
			InputMarkdown       string  `json:"inputMarkdown"`
			OutputMarkdown      string  `json:"outputMarkdown"`
			ConstraintsMarkdown string  `json:"constraintsMarkdown"`
			NotesMarkdown       string  `json:"notesMarkdown"`
			TimeLimitMs         int     `json:"timeLimitMs"`
			MemoryLimitMB       int     `json:"memoryLimitMb"`
			HiddenTestBundleKey *string `json:"hiddenTestBundleKey"`
			HiddenTestBundleSHA *string `json:"hiddenTestBundleSha256"`
		}
		if !decodeJSONBody(writer, request, maxProblemDraftBodyBytes, &body) {
			return
		}

		body.Slug = strings.TrimSpace(body.Slug)
		body.Title = strings.TrimSpace(body.Title)
		if body.Slug == "" || body.Title == "" || body.TimeLimitMs <= 0 || body.MemoryLimitMB <= 0 {
			writeError(writer, http.StatusBadRequest, "invalid_problem_draft")
			return
		}

		hiddenTestBundleKey, hiddenTestBundleSHA, err := resolveDraftBundleMetadata(request.Context(), bundleValidator, nil, body.HiddenTestBundleKey, body.HiddenTestBundleSHA)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		problem, err := problemStore.CreateDraft(request.Context(), problems.CreateDraftInput{
			UserID:                 user.ID,
			Slug:                   body.Slug,
			Title:                  body.Title,
			StatementMarkdown:      body.StatementMarkdown,
			InputMarkdown:          body.InputMarkdown,
			OutputMarkdown:         body.OutputMarkdown,
			ConstraintsMarkdown:    body.ConstraintsMarkdown,
			NotesMarkdown:          body.NotesMarkdown,
			TimeLimitMs:            body.TimeLimitMs,
			MemoryLimitMB:          body.MemoryLimitMB,
			HiddenTestBundleKey:    hiddenTestBundleKey,
			HiddenTestBundleSHA256: hiddenTestBundleSHA,
		})
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusCreated, problem)
	})
}

func getProblemDraftHandler(userStore users.Store, problemStore problems.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := currentUserFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		allowStaff, ok := currentUserCanManageDrafts(writer, request, userStore, user.ID)
		if !ok {
			return
		}

		problem, err := problemStore.GetDraftBySlug(request.Context(), request.PathValue("slug"), user.ID, allowStaff)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, problem)
	})
}

func updateProblemDraftHandler(userStore users.Store, problemStore problems.Store, bundleValidator problems.BundleValidator) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := currentUserFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		allowStaff, ok := currentUserCanManageDrafts(writer, request, userStore, user.ID)
		if !ok {
			return
		}

		existingDraft, err := problemStore.GetDraftBySlug(request.Context(), request.PathValue("slug"), user.ID, allowStaff)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		var body struct {
			Title               string  `json:"title"`
			StatementMarkdown   string  `json:"statementMarkdown"`
			InputMarkdown       string  `json:"inputMarkdown"`
			OutputMarkdown      string  `json:"outputMarkdown"`
			ConstraintsMarkdown string  `json:"constraintsMarkdown"`
			NotesMarkdown       string  `json:"notesMarkdown"`
			TimeLimitMs         int     `json:"timeLimitMs"`
			MemoryLimitMB       int     `json:"memoryLimitMb"`
			HiddenTestBundleKey *string `json:"hiddenTestBundleKey"`
			HiddenTestBundleSHA *string `json:"hiddenTestBundleSha256"`
		}
		if !decodeJSONBody(writer, request, maxProblemDraftBodyBytes, &body) {
			return
		}

		body.Title = strings.TrimSpace(body.Title)
		if body.Title == "" || body.TimeLimitMs <= 0 || body.MemoryLimitMB <= 0 {
			writeError(writer, http.StatusBadRequest, "invalid_problem_draft")
			return
		}

		hiddenTestBundleKey, hiddenTestBundleSHA, err := resolveDraftBundleMetadata(request.Context(), bundleValidator, &existingDraft, body.HiddenTestBundleKey, body.HiddenTestBundleSHA)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		problem, err := problemStore.UpdateDraft(request.Context(), problems.UpdateDraftInput{
			ActorUserID:            user.ID,
			AllowStaff:             allowStaff,
			Slug:                   request.PathValue("slug"),
			Title:                  body.Title,
			StatementMarkdown:      body.StatementMarkdown,
			InputMarkdown:          body.InputMarkdown,
			OutputMarkdown:         body.OutputMarkdown,
			ConstraintsMarkdown:    body.ConstraintsMarkdown,
			NotesMarkdown:          body.NotesMarkdown,
			TimeLimitMs:            body.TimeLimitMs,
			MemoryLimitMB:          body.MemoryLimitMB,
			HiddenTestBundleKey:    hiddenTestBundleKey,
			HiddenTestBundleSHA256: hiddenTestBundleSHA,
		})
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, problem)
	})
}

func submitProblemDraftForReviewHandler(userStore users.Store, problemStore problems.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		user, ok := currentUserFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		problem, err := problemStore.SubmitDraftForReview(request.Context(), request.PathValue("slug"), user.ID)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, problem)
	})
}

func listModerationProblemDraftsHandler(userStore users.Store, problemStore problems.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, ok := currentModeratorFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		items, err := problemStore.ListDraftsInReview(request.Context())
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{"drafts": items})
	})
}

func applyModerationDecisionHandler(userStore users.Store, problemStore problems.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		moderator, ok := currentModeratorFromRequest(writer, request, userStore)
		if !ok {
			return
		}

		var body struct {
			Decision        string `json:"decision"`
			ModerationNotes string `json:"moderationNotes"`
		}
		if !decodeJSONBody(writer, request, maxModerationDecisionBodyBytes, &body) {
			return
		}

		decision := problems.ModerationDecision(strings.TrimSpace(body.Decision))
		if decision != problems.DecisionApprove && decision != problems.DecisionReject && decision != problems.DecisionRequestChanges {
			writeError(writer, http.StatusBadRequest, "invalid_moderation_decision")
			return
		}

		problem, err := problemStore.ApplyModerationDecision(request.Context(), request.PathValue("slug"), moderator.ID, decision, body.ModerationNotes)
		if err != nil {
			writeProblemStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, problem)
	})
}

func listSubmissionsHandler(userStore users.Store, submissionStore submissions.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusInternalServerError, "missing_principal")
			return
		}

		user, err := userStore.GetOrCreateBySubject(request.Context(), principal.Subject)
		if err != nil {
			writeUserStoreError(writer, err)
			return
		}

		items, err := submissionStore.ListSubmissions(request.Context(), user.ID)
		if err != nil {
			writeSubmissionStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, map[string]any{"submissions": items})
	})
}

func createSubmissionHandler(userStore users.Store, submissionStore submissions.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusInternalServerError, "missing_principal")
			return
		}

		user, err := userStore.GetOrCreateBySubject(request.Context(), principal.Subject)
		if err != nil {
			writeUserStoreError(writer, err)
			return
		}

		var body struct {
			ProblemSlug string `json:"problemSlug"`
			Language    string `json:"language"`
			SourceCode  string `json:"sourceCode"`
		}
		if !decodeJSONBody(writer, request, maxSubmissionBodyBytes, &body) {
			return
		}

		submission, err := submissionStore.CreateSubmission(request.Context(), submissions.CreateInput{
			UserID:      user.ID,
			ProblemSlug: body.ProblemSlug,
			Language:    body.Language,
			SourceCode:  body.SourceCode,
		})
		if err != nil {
			writeSubmissionStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusCreated, submission)
	})
}

func getSubmissionHandler(userStore users.Store, submissionStore submissions.Store) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok {
			writeError(writer, http.StatusInternalServerError, "missing_principal")
			return
		}

		user, err := userStore.GetOrCreateBySubject(request.Context(), principal.Subject)
		if err != nil {
			writeUserStoreError(writer, err)
			return
		}

		submission, err := submissionStore.GetSubmissionByID(request.Context(), request.PathValue("id"), user.ID)
		if err != nil {
			writeSubmissionStoreError(writer, err)
			return
		}

		writeJSON(writer, http.StatusOK, submission)
	})
}

func currentUserFromRequest(writer http.ResponseWriter, request *http.Request, userStore users.Store) (users.User, bool) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeError(writer, http.StatusInternalServerError, "missing_principal")
		return users.User{}, false
	}

	user, err := userStore.GetOrCreateBySubject(request.Context(), principal.Subject)
	if err != nil {
		writeUserStoreError(writer, err)
		return users.User{}, false
	}

	return user, true
}

func currentUserCanManageDrafts(writer http.ResponseWriter, request *http.Request, userStore users.Store, userID string) (bool, bool) {
	hasRole, err := userStore.HasAnyRole(request.Context(), userID, users.RoleModerator, users.RoleAdmin)
	if err != nil {
		writeUserStoreError(writer, err)
		return false, false
	}

	return hasRole, true
}

func currentModeratorFromRequest(writer http.ResponseWriter, request *http.Request, userStore users.Store) (users.User, bool) {
	user, ok := currentUserFromRequest(writer, request, userStore)
	if !ok {
		return users.User{}, false
	}

	hasRole, ok := currentUserCanManageDrafts(writer, request, userStore, user.ID)
	if !ok {
		return users.User{}, false
	}
	if !hasRole {
		writeError(writer, http.StatusForbidden, "forbidden")
		return users.User{}, false
	}

	return user, true
}

func resolveDraftBundleMetadata(ctx context.Context, bundleValidator problems.BundleValidator, existing *problems.DraftProblem, keyValue *string, checksumValue *string) (string, string, error) {
	bundleKey := ""
	bundleChecksum := ""
	if existing != nil {
		bundleKey = existing.HiddenTestBundleKey
		bundleChecksum = existing.HiddenTestBundleSHA256
	}

	if keyValue == nil && checksumValue == nil {
		return bundleKey, bundleChecksum, nil
	}
	if keyValue == nil || checksumValue == nil {
		return "", "", problems.ErrInvalidBundleChecksum
	}

	bundleKey = strings.TrimSpace(*keyValue)
	bundleChecksum = strings.TrimSpace(*checksumValue)
	if err := problems.ValidateBundleMetadata(bundleKey, bundleChecksum); err != nil {
		return "", "", err
	}
	if bundleKey == "" && bundleChecksum == "" {
		return "", "", nil
	}
	if err := bundleValidator.ValidateBundle(ctx, bundleKey, bundleChecksum); err != nil {
		return "", "", err
	}

	return bundleKey, bundleChecksum, nil
}

func writeProblemStoreError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, problems.ErrStoreNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "problem_store_not_configured")
	case errors.Is(err, problems.ErrProblemSlugTaken):
		writeError(writer, http.StatusConflict, "problem_slug_taken")
	case errors.Is(err, problems.ErrDraftNotFound):
		writeError(writer, http.StatusNotFound, "problem_draft_not_found")
	case errors.Is(err, problems.ErrInvalidLifecycleTransition):
		writeError(writer, http.StatusConflict, "invalid_problem_lifecycle_transition")
	case errors.Is(err, problems.ErrDraftNotReadyForReview):
		writeError(writer, http.StatusConflict, "problem_draft_not_ready_for_review")
	case errors.Is(err, problems.ErrInvalidModerationDecision):
		writeError(writer, http.StatusBadRequest, "invalid_moderation_decision")
	case errors.Is(err, problems.ErrBundleValidatorNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "hidden_test_bundle_validator_not_configured")
	case errors.Is(err, problems.ErrBundleUploaderNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "hidden_test_bundle_uploader_not_configured")
	case errors.Is(err, problems.ErrInvalidBundleContents):
		writeError(writer, http.StatusBadRequest, "invalid_hidden_test_bundle_upload")
	case errors.Is(err, problems.ErrBundleNotFound), errors.Is(err, problems.ErrBundleChecksumMismatch), errors.Is(err, problems.ErrInvalidBundleChecksum):
		writeError(writer, http.StatusBadRequest, "invalid_hidden_test_bundle")
	case errors.Is(err, problems.ErrNotFound):
		writeError(writer, http.StatusNotFound, "problem_not_found")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_server_error")
	}
}

func writeUserStoreError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, users.ErrStoreNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "user_store_not_configured")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_server_error")
	}
}

func writeSubmissionStoreError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, submissions.ErrStoreNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "submission_store_not_configured")
	case errors.Is(err, submissions.ErrProblemNotFound), errors.Is(err, submissions.ErrSubmissionNotFound):
		writeError(writer, http.StatusNotFound, "not_found")
	case errors.Is(err, submissions.ErrInvalidLanguage):
		writeError(writer, http.StatusBadRequest, "invalid_language")
	default:
		writeError(writer, http.StatusInternalServerError, "internal_server_error")
	}
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writeJSON(writer, status, map[string]string{"error": message})
}

func decodeJSONBody(writer http.ResponseWriter, request *http.Request, limit int64, target any) bool {
	request.Body = http.MaxBytesReader(writer, request.Body, limit)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request_body")
		return false
	}

	var extra struct{}
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(writer, http.StatusBadRequest, "invalid_request_body")
		return false
	}

	return true
}

func authDependencyReady(verifier auth.Verifier) bool {
	if verifier == nil {
		return false
	}
	_, disabled := verifier.(auth.DisabledVerifier)
	return !disabled
}

func databaseDependencyReady(problemStore problems.Store, userStore users.Store, submissionStore submissions.Store) bool {
	_, problemDisabled := problemStore.(problems.DisabledStore)
	_, userDisabled := userStore.(users.DisabledStore)
	_, submissionDisabled := submissionStore.(submissions.DisabledStore)
	return !problemDisabled && !userDisabled && !submissionDisabled
}

func bundleValidationDependencyReady(bundleValidator problems.BundleValidator) bool {
	if bundleValidator == nil {
		return false
	}
	_, disabled := bundleValidator.(problems.DisabledBundleValidator)
	return !disabled
}

func submissionQueueDepthProvider(submissionStore submissions.Store) apiobs.QueueDepthProvider {
	provider, ok := any(submissionStore).(apiobs.QueueDepthProvider)
	if !ok {
		return nil
	}

	return provider
}

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
