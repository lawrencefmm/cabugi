package auth

import (
	"os"
	"strings"
)

type ClerkConfig struct {
	PublicKeyPEM   string
	AllowedParties []string
}

func LoadClerkConfig() ClerkConfig {
	config := ClerkConfig{
		PublicKeyPEM: strings.ReplaceAll(os.Getenv("CLERK_PEM_PUBLIC_KEY"), "\\n", "\n"),
	}

	for _, party := range strings.Split(os.Getenv("CLERK_ALLOWED_PARTIES"), ",") {
		party = strings.TrimSpace(party)
		if party == "" {
			continue
		}

		config.AllowedParties = append(config.AllowedParties, party)
	}

	return config
}

func (config ClerkConfig) Enabled() bool {
	return strings.TrimSpace(config.PublicKeyPEM) != ""
}
