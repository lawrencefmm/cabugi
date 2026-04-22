package config

import (
	"os"
	"strings"
)

const defaultAddress = "127.0.0.1:8080"
const defaultDatabaseURL = "postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable"

var defaultAllowedOrigins = []string{
	"http://127.0.0.1:3000",
	"http://localhost:3000",
}

type Config struct {
	Address        string
	DatabaseURL    string
	AllowedOrigins []string
}

func Load() Config {
	address := os.Getenv("API_ADDRESS")
	if address == "" {
		address = defaultAddress
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	allowedOrigins := defaultAllowedOrigins
	if configuredOrigins := strings.TrimSpace(os.Getenv("WEB_ALLOWED_ORIGINS")); configuredOrigins != "" {
		allowedOrigins = make([]string, 0)
		for _, origin := range strings.Split(configuredOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin == "" {
				continue
			}

			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	return Config{Address: address, DatabaseURL: databaseURL, AllowedOrigins: allowedOrigins}
}
