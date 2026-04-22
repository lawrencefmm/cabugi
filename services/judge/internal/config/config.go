package config

import "os"

const defaultDatabaseURL = "postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable"

type Config struct {
	DatabaseURL string
}

func Load() Config {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = defaultDatabaseURL
	}

	return Config{DatabaseURL: databaseURL}
}
