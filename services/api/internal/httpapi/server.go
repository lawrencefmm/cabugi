package httpapi

import (
	"net/http"

	"github.com/lawrencefmm/cabugi/services/api/internal/auth"
	openapiasset "github.com/lawrencefmm/cabugi/services/api/openapi"
)

func NewMux(verifier auth.Verifier) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /openapi/v1.yaml", openAPIHandler)
	mux.Handle("GET /v1/me", auth.RequireAuth(verifier, http.HandlerFunc(currentUserHandler)))

	return mux
}

func NewServer(address string, verifier auth.Verifier) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: NewMux(verifier),
	}
}

func healthzHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}

func openAPIHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/yaml")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(openapiasset.V1)
}

func currentUserHandler(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"missing_principal"}`))
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"subject":"` + principal.Subject + `"}`))
}
