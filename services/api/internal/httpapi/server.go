package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	"github.com/lawrencefmm/cabugi/services/api/internal/problems"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
	openapiasset "github.com/lawrencefmm/cabugi/services/api/openapi"
)

func NewMux(verifier auth.Verifier, problemStore problems.Store, userStore users.Store) *http.ServeMux {
	if problemStore == nil {
		problemStore = problems.DisabledStore{}
	}
	if userStore == nil {
		userStore = users.DisabledStore{}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /openapi/v1.yaml", openAPIHandler)
	mux.HandleFunc("GET /v1/problems", listPublishedProblemsHandler(problemStore))
	mux.HandleFunc("GET /v1/problems/{slug}", getPublishedProblemHandler(problemStore))
	mux.Handle("GET /v1/me", auth.RequireAuth(verifier, currentUserHandler(userStore)))

	return mux
}

func NewServer(address string, verifier auth.Verifier, problemStore problems.Store, userStore users.Store) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: NewMux(verifier, problemStore, userStore),
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

func writeProblemStoreError(writer http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, problems.ErrStoreNotConfigured):
		writeError(writer, http.StatusServiceUnavailable, "problem_store_not_configured")
	case errors.Is(err, problems.ErrNotFound):
		writeError(writer, http.StatusNotFound, "problem_not_found")
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
