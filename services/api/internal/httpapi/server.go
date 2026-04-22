package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
	"github.com/lawrencefmm/cabugi/services/api/internal/submissions"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
	openapiasset "github.com/lawrencefmm/cabugi/services/api/openapi"
)

func NewMux(verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, bundleValidators ...problems.BundleValidator) *http.ServeMux {
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
	mux.HandleFunc("GET /openapi/v1.yaml", openAPIHandler)
	mux.HandleFunc("GET /v1/problems", listPublishedProblemsHandler(problemStore))
	mux.HandleFunc("GET /v1/problems/{slug}", getPublishedProblemHandler(problemStore))
	mux.Handle("POST /v1/problem-drafts", auth.RequireAuth(verifier, createProblemDraftHandler(userStore, problemStore, bundleValidator)))
	mux.Handle("GET /v1/problem-drafts/{slug}", auth.RequireAuth(verifier, getProblemDraftHandler(userStore, problemStore)))
	mux.Handle("PATCH /v1/problem-drafts/{slug}", auth.RequireAuth(verifier, updateProblemDraftHandler(userStore, problemStore, bundleValidator)))
	mux.Handle("GET /v1/submissions", auth.RequireAuth(verifier, listSubmissionsHandler(userStore, submissionStore)))
	mux.Handle("POST /v1/submissions", auth.RequireAuth(verifier, createSubmissionHandler(userStore, submissionStore)))
	mux.Handle("GET /v1/submissions/{id}", auth.RequireAuth(verifier, getSubmissionHandler(userStore, submissionStore)))
	mux.Handle("GET /v1/me", auth.RequireAuth(verifier, currentUserHandler(userStore)))

	return mux
}

func NewServer(address string, verifier auth.Verifier, problemStore problems.Store, userStore users.Store, submissionStore submissions.Store, allowedOrigins []string, bundleValidators ...problems.BundleValidator) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: withCORS(NewMux(verifier, problemStore, userStore, submissionStore, bundleValidators...), allowedOrigins),
	}
}

func healthzHandler(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
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
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body")
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
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body")
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
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			writeError(writer, http.StatusBadRequest, "invalid_request_body")
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
	case errors.Is(err, problems.ErrBundleValidatorNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "hidden_test_bundle_validator_not_configured")
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

func writeJSON(writer http.ResponseWriter, status int, payload any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(payload)
}
