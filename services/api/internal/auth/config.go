package auth

import (
	"os"
	"strings"
)

type ClerkConfig struct {
	PublicKeyPEM     string
	JWKSURL          string
	Issuer           string
	AllowedParties   []string
	AllowedAudiences []string
}

func LoadClerkConfig() ClerkConfig {
	config := ClerkConfig{
		PublicKeyPEM: strings.ReplaceAll(os.Getenv("CLERK_PEM_PUBLIC_KEY"), "\\n", "\n"),
		JWKSURL:      strings.TrimSpace(os.Getenv("CLERK_JWKS_URL")),
		Issuer:       strings.TrimSpace(os.Getenv("CLERK_ISSUER")),
	}
	usingLocalTestDefaults := localTestAuthEnabled() && strings.TrimSpace(config.PublicKeyPEM) == "" && strings.TrimSpace(config.JWKSURL) == "" && strings.TrimSpace(config.Issuer) == ""

	for _, party := range strings.Split(os.Getenv("CLERK_ALLOWED_PARTIES"), ",") {
		party = strings.TrimSpace(party)
		if party == "" {
			continue
		}

		config.AllowedParties = append(config.AllowedParties, party)
	}

	for _, audience := range strings.Split(os.Getenv("CLERK_ALLOWED_AUDIENCES"), ",") {
		audience = strings.TrimSpace(audience)
		if audience == "" {
			continue
		}

		config.AllowedAudiences = append(config.AllowedAudiences, audience)
	}

	if usingLocalTestDefaults {
		config.PublicKeyPEM = localTestAuthPublicKeyPEM
		config.Issuer = localTestAuthIssuer
		config.AllowedParties = nil
		if len(config.AllowedAudiences) == 0 {
			config.AllowedAudiences = []string{localTestAuthAudience}
		}
	}

	if config.JWKSURL == "" && config.Issuer != "" && strings.TrimSpace(config.PublicKeyPEM) == "" {
		config.JWKSURL = strings.TrimRight(config.Issuer, "/") + "/.well-known/jwks.json"
	}

	return config
}

func (config ClerkConfig) Enabled() bool {
	return strings.TrimSpace(config.PublicKeyPEM) != "" || strings.TrimSpace(config.JWKSURL) != ""
}
