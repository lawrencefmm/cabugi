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

	if config.JWKSURL == "" && config.Issuer != "" {
		config.JWKSURL = strings.TrimRight(config.Issuer, "/") + "/.well-known/jwks.json"
	}

	return config
}

func (config ClerkConfig) Enabled() bool {
	return strings.TrimSpace(config.PublicKeyPEM) != "" || strings.TrimSpace(config.JWKSURL) != ""
}
