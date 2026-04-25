package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken          = errors.New("missing session token")
	ErrInvalidToken          = errors.New("invalid session token")
	ErrVerifierNotConfigured = errors.New("clerk verifier not configured")
)

const sessionCookieName = "__session"

const defaultJWKSRefreshInterval = 5 * time.Minute
const tokenClockSkewLeeway = 30 * time.Second

type Principal struct {
	Subject string
}

type Verifier interface {
	Verify(context.Context, string) (Principal, error)
}

type ClerkVerifier struct {
	publicKey           *rsa.PublicKey
	jwksURL             string
	issuer              string
	allowedParties      []string
	allowedAudiences    []string
	httpClient          *http.Client
	jwksRefreshInterval time.Duration

	mu         sync.RWMutex
	jwksKeys   map[string]*rsa.PublicKey
	lastJWKSAt time.Time
}

type DisabledVerifier struct{}

type principalContextKey struct{}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	KTY string `json:"kty"`
	KID string `json:"kid"`
	Use string `json:"use"`
	ALG string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewClerkVerifier(config ClerkConfig) (*ClerkVerifier, error) {
	if !config.Enabled() {
		return nil, ErrVerifierNotConfigured
	}
	if strings.TrimSpace(config.Issuer) == "" {
		return nil, errors.New("clerk issuer not configured")
	}
	if len(config.AllowedParties) == 0 && len(config.AllowedAudiences) == 0 {
		return nil, errors.New("clerk authorized party or audience not configured")
	}

	verifier := &ClerkVerifier{
		jwksURL:             strings.TrimSpace(config.JWKSURL),
		issuer:              strings.TrimSpace(config.Issuer),
		allowedParties:      slices.Clone(config.AllowedParties),
		allowedAudiences:    slices.Clone(config.AllowedAudiences),
		httpClient:          &http.Client{Timeout: 5 * time.Second},
		jwksRefreshInterval: defaultJWKSRefreshInterval,
		jwksKeys:            make(map[string]*rsa.PublicKey),
	}

	if strings.TrimSpace(config.PublicKeyPEM) != "" {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(config.PublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("parse Clerk public key: %w", err)
		}

		verifier.publicKey = publicKey
	}

	return verifier, nil
}

func (verifier *ClerkVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))

	claims := jwt.MapClaims{}
	unverifiedToken, _, err := parser.ParseUnverified(token, claims)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	publicKey, err := verifier.publicKeyForToken(ctx, unverifiedToken)
	if err != nil {
		return Principal{}, ErrInvalidToken
	}

	claims = jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) {
		return publicKey, nil
	}, jwt.WithLeeway(tokenClockSkewLeeway), jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil || !parsedToken.Valid {
		return Principal{}, ErrInvalidToken
	}

	issuer, _ := claims["iss"].(string)
	if strings.TrimSpace(issuer) != verifier.issuer {
		return Principal{}, ErrInvalidToken
	}

	if len(verifier.allowedParties) > 0 {
		azp, _ := claims["azp"].(string)
		if azp == "" || !slices.Contains(verifier.allowedParties, azp) {
			return Principal{}, ErrInvalidToken
		}
	}

	if len(verifier.allowedAudiences) > 0 {
		if !audiencesContainExpected(claimAudiences(claims["aud"]), verifier.allowedAudiences) {
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

	if cookie, err := request.Cookie(sessionCookieName); err == nil {
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

func (verifier *ClerkVerifier) publicKeyForToken(ctx context.Context, token *jwt.Token) (*rsa.PublicKey, error) {
	if verifier.jwksURL == "" {
		return verifier.publicKey, nil
	}

	kid, _ := token.Header["kid"].(string)
	if strings.TrimSpace(kid) == "" {
		return nil, ErrInvalidToken
	}

	if key := verifier.cachedJWKSKey(kid); key != nil {
		return key, nil
	}

	if err := verifier.refreshJWKS(ctx); err != nil {
		return nil, err
	}

	if key := verifier.cachedJWKSKey(kid); key != nil {
		return key, nil
	}

	return nil, ErrInvalidToken
}

func (verifier *ClerkVerifier) cachedJWKSKey(kid string) *rsa.PublicKey {
	verifier.mu.RLock()
	defer verifier.mu.RUnlock()

	if time.Since(verifier.lastJWKSAt) > verifier.jwksRefreshInterval {
		return nil
	}

	return verifier.jwksKeys[kid]
}

func (verifier *ClerkVerifier) refreshJWKS(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, verifier.jwksURL, nil)
	if err != nil {
		return err
	}

	response, err := verifier.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected jwks status %d", response.StatusCode)
	}

	var document jwksDocument
	if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
		return err
	}

	keys := make(map[string]*rsa.PublicKey, len(document.Keys))
	for _, key := range document.Keys {
		if parsedKey, ok := parseJWKSRSAKey(key); ok {
			keys[key.KID] = parsedKey
		}
	}

	verifier.mu.Lock()
	verifier.jwksKeys = keys
	verifier.lastJWKSAt = time.Now().UTC()
	verifier.mu.Unlock()

	return nil
}

func parseJWKSRSAKey(key jwkKey) (*rsa.PublicKey, bool) {
	if key.KTY != "RSA" || strings.TrimSpace(key.KID) == "" {
		return nil, false
	}
	if key.Use != "" && key.Use != "sig" {
		return nil, false
	}
	if key.ALG != "" && key.ALG != jwt.SigningMethodRS256.Alg() {
		return nil, false
	}

	modulusBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil || len(modulusBytes) == 0 {
		return nil, false
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil || len(exponentBytes) == 0 {
		return nil, false
	}

	exponent := 0
	for _, value := range exponentBytes {
		exponent = (exponent << 8) | int(value)
	}
	if exponent == 0 {
		return nil, false
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulusBytes), E: exponent}, true
}

func claimAudiences(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	case []string:
		return slices.Clone(typed)
	case []any:
		audiences := make([]string, 0, len(typed))
		for _, entry := range typed {
			audience, _ := entry.(string)
			if strings.TrimSpace(audience) == "" {
				continue
			}
			audiences = append(audiences, audience)
		}
		return audiences
	default:
		return nil
	}
}

func audiencesContainExpected(claimAudiences []string, allowedAudiences []string) bool {
	for _, claimAudience := range claimAudiences {
		if slices.Contains(allowedAudiences, claimAudience) {
			return true
		}
	}

	return false
}
