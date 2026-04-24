package config

import (
	"os"
	"strconv"
	"strings"
)

const defaultAddress = "127.0.0.1:8080"
const defaultDatabaseURL = "postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable"
const defaultObjectStorageEndpoint = "http://127.0.0.1:9000"
const defaultObjectStorageRegion = "us-east-1"
const defaultObjectStorageBucket = "cabugi-hidden-tests"
const defaultObjectStorageAccessKeyID = "minioadmin"
const defaultObjectStorageSecretKey = "minioadmin"
const defaultObjectStorageUsePathStyle = true

var defaultAllowedOrigins = []string{
	"http://127.0.0.1:3000",
	"http://localhost:3000",
}

type Config struct {
	Address                       string
	DatabaseURL                   string
	AllowedOrigins                []string
	ObjectStorage                 ObjectStorageConfig
	RequireAuth                   bool
	RequireDatabase               bool
	RequireHiddenBundleValidation bool
}

type ObjectStorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
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

	return Config{
		Address:        address,
		DatabaseURL:    databaseURL,
		AllowedOrigins: allowedOrigins,
		ObjectStorage: ObjectStorageConfig{
			Endpoint:        envOrDefault("OBJECT_STORAGE_ENDPOINT", defaultObjectStorageEndpoint),
			Region:          envOrDefault("OBJECT_STORAGE_REGION", defaultObjectStorageRegion),
			Bucket:          envOrDefault("OBJECT_STORAGE_BUCKET", defaultObjectStorageBucket),
			AccessKeyID:     envOrDefault("OBJECT_STORAGE_ACCESS_KEY_ID", defaultObjectStorageAccessKeyID),
			SecretAccessKey: envOrDefault("OBJECT_STORAGE_SECRET_ACCESS_KEY", defaultObjectStorageSecretKey),
			UsePathStyle:    boolEnvOrDefault("OBJECT_STORAGE_USE_PATH_STYLE", defaultObjectStorageUsePathStyle),
		},
		RequireAuth:                   boolEnvOrDefault("API_REQUIRE_AUTH", false),
		RequireDatabase:               boolEnvOrDefault("API_REQUIRE_DATABASE", false),
		RequireHiddenBundleValidation: boolEnvOrDefault("API_REQUIRE_HIDDEN_BUNDLE_VALIDATION", false),
	}
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func boolEnvOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
