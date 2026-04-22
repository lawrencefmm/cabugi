package auth

import (
	"errors"
	"net/http"
)

func RequireAuth(verifier Verifier, next http.Handler) http.Handler {
	if verifier == nil {
		verifier = DisabledVerifier{}
	}

	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		token, err := TokenFromRequest(request)
		if err != nil {
			writeError(writer, http.StatusUnauthorized, "unauthorized")
			return
		}

		principal, err := verifier.Verify(request.Context(), token)
		if err != nil {
			status := http.StatusUnauthorized
			message := "unauthorized"
			if errors.Is(err, ErrVerifierNotConfigured) {
				status = http.StatusServiceUnavailable
				message = "auth_not_configured"
			}

			writeError(writer, status, message)
			return
		}

		next.ServeHTTP(writer, request.WithContext(WithPrincipal(request.Context(), principal)))
	})
}

func writeError(writer http.ResponseWriter, status int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_, _ = writer.Write([]byte(`{"error":"` + message + `"}`))
}
