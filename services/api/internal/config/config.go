package config

import "os"

const defaultAddress = "127.0.0.1:8080"
const defaultDatabaseURL = "postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable"

type Config struct {
	Address     string
	DatabaseURL string
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

	return Config{Address: address, DatabaseURL: databaseURL}
}
