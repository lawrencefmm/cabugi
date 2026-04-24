package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken          = errors.New("missing session token")
	ErrInvalidToken          = errors.New("invalid session token")
	ErrVerifierNotConfigured = errors.New("clerk verifier not configured")
)

type Principal struct {
	Subject string
}

type Verifier interface {
	Verify(context.Context, string) (Principal, error)
}

type ClerkVerifier struct {
	publicKey      *rsa.PublicKey
	allowedParties []string
}

type DisabledVerifier struct{}

type principalContextKey struct{}

func NewClerkVerifier(config ClerkConfig) (*ClerkVerifier, error) {
	if !config.Enabled() {
		return nil, ErrVerifierNotConfigured
	}

	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.PublicKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("parse Clerk public key: %w", err)
	}

	return &ClerkVerifier{
		publicKey:      publicKey,
		allowedParties: slices.Clone(config.AllowedParties),
	}, nil
}

func (verifier *ClerkVerifier) Verify(_ context.Context, token string) (Principal, error) {
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (any, error) {
		return verifier.publicKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil || !parsedToken.Valid {
		return Principal{}, ErrInvalidToken
	}

	if len(verifier.allowedParties) > 0 {
		if azp, _ := claims["azp"].(string); azp != "" && !slices.Contains(verifier.allowedParties, azp) {
			return Principal{}, ErrInvalidToken
		}
	}

	subject, _ := claims["sub"].(string)
	if strings.TrimSpace(subject) == "" {
		return Principal{}, ErrInvalidToken
	}

	return Principal{Subject: subject}, nil
}

func (DisabledVerifier) Verify(context.Context, string) (Principal, error) {
	return Principal{}, ErrVerifierNotConfigured
}

func TokenFromRequest(request *http.Request) (string, error) {
	authorization := strings.TrimSpace(request.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") {
		token := strings.TrimSpace(authorization[len("Bearer "):])
		if token != "" {
			return token, nil
		}
	}

	if cookie, err := request.Cookie("__session"); err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token, nil
		}
	}

	return "", ErrMissingToken
}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
