package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestLoadClerkConfigUsesLocalTestDefaults(t *testing.T) {
	t.Setenv("LOCAL_TEST_AUTH_ENABLED", "true")
	t.Setenv("CLERK_PEM_PUBLIC_KEY", "")
	t.Setenv("CLERK_JWKS_URL", "")
	t.Setenv("CLERK_ISSUER", "")
	t.Setenv("CLERK_ALLOWED_PARTIES", "")
	t.Setenv("CLERK_ALLOWED_AUDIENCES", "")

	config := LoadClerkConfig()
	if config.PublicKeyPEM != localTestAuthPublicKeyPEM {
		t.Fatalf("LoadClerkConfig() publicKeyPEM = %q, want local test public key", config.PublicKeyPEM)
	}
	if config.Issuer != localTestAuthIssuer {
		t.Fatalf("LoadClerkConfig() issuer = %q, want %q", config.Issuer, localTestAuthIssuer)
	}
	if config.JWKSURL != "" {
		t.Fatalf("LoadClerkConfig() jwksURL = %q, want empty when public key is configured", config.JWKSURL)
	}
	if len(config.AllowedAudiences) != 1 || config.AllowedAudiences[0] != localTestAuthAudience {
		t.Fatalf("LoadClerkConfig() allowedAudiences = %#v, want [%q]", config.AllowedAudiences, localTestAuthAudience)
	}
}

func TestTokenFromRequestAcceptsSessionCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-token"})

	token, err := TokenFromRequest(request)
	if err != nil {
		t.Fatalf("TokenFromRequest() error = %v", err)
	}
	if token != "cookie-token" {
		t.Fatalf("TokenFromRequest() token = %q, want %q", token, "cookie-token")
	}
}

func TestTokenFromRequestPrefersAuthorizationHeaderOverCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/me", nil)
	request.Header.Set("Authorization", "Bearer header-token")
	request.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "cookie-token"})

	token, err := TokenFromRequest(request)
	if err != nil {
		t.Fatalf("TokenFromRequest() error = %v", err)
	}
	if token != "header-token" {
		t.Fatalf("TokenFromRequest() token = %q, want %q", token, "header-token")
	}
}

func TestClerkVerifierRejectsInvalidIssuer(t *testing.T) {
	privateKey, config := newStaticVerifierConfig(t)
	verifier, err := NewClerkVerifier(config)
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	token := signedToken(t, privateKey, "static-key", jwt.MapClaims{
		"sub": "user_123",
		"iss": "https://wrong-issuer.example",
		"aud": []string{"cabugi-web"},
		"azp": "http://localhost:3000",
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), token); err != ErrInvalidToken {
		t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestClerkVerifierRejectsInvalidAudience(t *testing.T) {
	privateKey, config := newStaticVerifierConfig(t)
	verifier, err := NewClerkVerifier(config)
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	token := signedToken(t, privateKey, "static-key", jwt.MapClaims{
		"sub": "user_123",
		"iss": config.Issuer,
		"aud": []string{"different-audience"},
		"azp": "http://localhost:3000",
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	})

	if _, err := verifier.Verify(context.Background(), token); err != ErrInvalidToken {
		t.Fatalf("Verify() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestClerkVerifierUsesJWKSAndRefreshesOnKeyRotation(t *testing.T) {
	privateKeyOne := newRSAKey(t)
	privateKeyTwo := newRSAKey(t)

	var (
		mu      sync.RWMutex
		current []jwkKey
	)
	current = []jwkKey{jwkFromPublicKey("key-1", &privateKeyOne.PublicKey)}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		mu.RLock()
		payload := jwksDocument{Keys: append([]jwkKey(nil), current...)}
		mu.RUnlock()
		writer.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(writer).Encode(payload); err != nil {
			t.Fatalf("json.NewEncoder() error = %v", err)
		}
	}))
	defer server.Close()

	verifier, err := NewClerkVerifier(ClerkConfig{
		JWKSURL:          server.URL,
		Issuer:           "https://clerk.example.com",
		AllowedAudiences: []string{"cabugi-web"},
	})
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	firstToken := signedToken(t, privateKeyOne, "key-1", jwt.MapClaims{
		"sub": "user_123",
		"iss": "https://clerk.example.com",
		"aud": []string{"cabugi-web"},
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	})
	principal, err := verifier.Verify(context.Background(), firstToken)
	if err != nil {
		t.Fatalf("Verify() first token error = %v", err)
	}
	if principal.Subject != "user_123" {
		t.Fatalf("Verify() first principal = %#v, want subject user_123", principal)
	}

	mu.Lock()
	current = []jwkKey{jwkFromPublicKey("key-2", &privateKeyTwo.PublicKey)}
	mu.Unlock()

	secondToken := signedToken(t, privateKeyTwo, "key-2", jwt.MapClaims{
		"sub": "user_456",
		"iss": "https://clerk.example.com",
		"aud": []string{"cabugi-web"},
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(-time.Minute).Unix(),
	})
	principal, err = verifier.Verify(context.Background(), secondToken)
	if err != nil {
		t.Fatalf("Verify() rotated token error = %v", err)
	}
	if principal.Subject != "user_456" {
		t.Fatalf("Verify() rotated principal = %#v, want subject user_456", principal)
	}
}

func TestLoadClerkConfigDerivesJWKSURLFromIssuer(t *testing.T) {
	t.Setenv("CLERK_ISSUER", "https://clerk.example.com")
	t.Setenv("CLERK_JWKS_URL", "")
	t.Setenv("CLERK_ALLOWED_AUDIENCES", "cabugi-web")

	config := LoadClerkConfig()
	if config.JWKSURL != "https://clerk.example.com/.well-known/jwks.json" {
		t.Fatalf("LoadClerkConfig() jwksURL = %q, want derived jwks URL", config.JWKSURL)
	}
}

func TestClerkVerifierAcceptsLocalSmokeFixtureToken(t *testing.T) {
	verifier, err := NewClerkVerifier(ClerkConfig{
		PublicKeyPEM: `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAx6dXtLfQmM2j0iJbvfpQ
WZWVCXVcGSESpbP1+vFP9Z5hg+fEkwXRFmTQzg2bSfylONEyFWutPmhq3TCoCqVC
vVQBkF2fUL7ro7ChvfALguas7dSp43tFDwQISKbL5OP8UrAlD4I07UV29oieiwRb
3qyLRK8YJsJTzMsODrNWGN5RRo/tsPtxe9vza4rrSfJ7NB0NPUUWbk0yCLFXdtBx
/up86pYWD1BtEUOCru5ySfzV8nYL4PHcoF7RYZ4Xv0CnxuekGEqpC9ooqHHZRc86
M1KCccrEySxcgTnWXBrA3eleJf3WsWR9ZKvr12waiinY/wT5qYMcIbLMsLAb920J
vQIDAQAB
-----END PUBLIC KEY-----`,
		Issuer:           "https://cabugi.local.test",
		AllowedAudiences: []string{"cabugi-local-test"},
	})
	if err != nil {
		t.Fatalf("NewClerkVerifier() error = %v", err)
	}

	token := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImNhYnVnaS1sb2NhbC10ZXN0LWtleS0xIn0.eyJpc3MiOiJodHRwczovL2NhYnVnaS5sb2NhbC50ZXN0IiwiYXVkIjpbImNhYnVnaS1sb2NhbC10ZXN0Il0sImlhdCI6MTczNTY4OTYwMCwiZXhwIjo0MTAyNDQ0ODAwLCJzdWIiOiJ1c2VyX2UyZV9hdXRob3IifQ.tAwcNVNI-HmDBqfiFre1PIpphtN9eYU37KNPrBEaa1gsgFTmdOo-DOViNfqAluoqfJIp27oMJeweOd_7YrjYHC89LsibCTrK1KwzdSl_1wr14sIDNaAYMhViYP7Cy3-vzpGSlfacJjzQmo2dugWunh2qNnbJKX7asfY5niLQO55tRQLTTQp0QQuMG2Oz-fzwQ9WGOlQbhgsUdBn9dOXpa5RicAROSpDFALqEnguIHV9e7uKdjcgBJNV7AALavtvtRf9XWVVuF2St1ITbtdLG9HkaA32Xu6YQPfmakdptsxJUZJ-YvHBqYfyOItMNQNwT0xAqqNodNnFq0bIaKy-8sQ"

	principal, err := verifier.Verify(context.Background(), token)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if principal.Subject != "user_e2e_author" {
		t.Fatalf("Verify() principal = %#v, want subject user_e2e_author", principal)
	}
}

func newStaticVerifierConfig(t *testing.T) (*rsa.PrivateKey, ClerkConfig) {
	t.Helper()
	privateKey := newRSAKey(t)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("x509.MarshalPKIXPublicKey() error = %v", err)
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes})
	return privateKey, ClerkConfig{
		PublicKeyPEM:     string(publicKeyPEM),
		Issuer:           "https://clerk.example.com",
		AllowedParties:   []string{"http://localhost:3000"},
		AllowedAudiences: []string{"cabugi-web"},
	}
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	return privateKey
}

func signedToken(t *testing.T, privateKey *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}
	return signedToken
}

func jwkFromPublicKey(kid string, publicKey *rsa.PublicKey) jwkKey {
	return jwkKey{
		KTY: "RSA",
		KID: kid,
		Use: "sig",
		ALG: jwt.SigningMethodRS256.Alg(),
		N:   base64.RawURLEncoding.EncodeToString(publicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(rsaExponentBytes(publicKey.E)),
	}
}

func rsaExponentBytes(value int) []byte {
	if value == 0 {
		return []byte{0}
	}

	bytes := make([]byte, 0, 4)
	for current := value; current > 0; current >>= 8 {
		bytes = append([]byte{byte(current & 0xff)}, bytes...)
	}
	return bytes
}
