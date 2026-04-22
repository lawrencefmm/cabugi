package httpapi

import (
	"net/http"

	openapiasset "github.com/lawrencefmm/cabugi/services/api/openapi"
)

func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	mux.HandleFunc("GET /openapi/v1.yaml", openAPIHandler)

	return mux
}

func NewServer(address string) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: NewMux(),
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
