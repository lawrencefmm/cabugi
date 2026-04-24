package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultDatabaseURL               = "postgres://cabugi:cabugi@127.0.0.1:5432/cabugi?sslmode=disable"
	defaultObjectStorageEndpoint     = "http://127.0.0.1:9000"
	defaultObjectStorageRegion       = "us-east-1"
	defaultObjectStorageBucket       = "cabugi-hidden-tests"
	defaultObjectStorageAccessKeyID  = "minioadmin"
	defaultObjectStorageSecretKey    = "minioadmin"
	defaultObjectStorageUsePathStyle = true
	defaultMaxJobAttempts            = 3
	defaultJobLeaseDuration          = 30 * time.Second
	defaultJobLeaseRenewInterval     = 10 * time.Second
	defaultWorkerPollInterval        = 3 * time.Second
	defaultWorkerRetryDelay          = 5 * time.Second
)

type ObjectStorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
}

type Config struct {
	DatabaseURL        string
	ObjectStorage      ObjectStorageConfig
	MaxJobAttempts     int
	JobLeaseDuration   time.Duration
	JobLeaseRenewAfter time.Duration
	WorkerPollInterval time.Duration
	WorkerRetryDelay   time.Duration
}

func Load() Config {
	jobLeaseDuration := durationEnvOrDefault("JUDGE_JOB_LEASE_DURATION", defaultJobLeaseDuration)
	jobLeaseRenewAfter := durationEnvOrDefault("JUDGE_JOB_LEASE_RENEW_INTERVAL", defaultJobLeaseRenewInterval)
	if jobLeaseRenewAfter >= jobLeaseDuration {
		jobLeaseRenewAfter = jobLeaseDuration / 2
		if jobLeaseRenewAfter <= 0 {
			jobLeaseRenewAfter = defaultJobLeaseRenewInterval
		}
	}

	return Config{
		DatabaseURL: envOrDefault("DATABASE_URL", defaultDatabaseURL),
		ObjectStorage: ObjectStorageConfig{
			Endpoint:        envOrDefault("OBJECT_STORAGE_ENDPOINT", defaultObjectStorageEndpoint),
			Region:          envOrDefault("OBJECT_STORAGE_REGION", defaultObjectStorageRegion),
			Bucket:          envOrDefault("OBJECT_STORAGE_BUCKET", defaultObjectStorageBucket),
			AccessKeyID:     envOrDefault("OBJECT_STORAGE_ACCESS_KEY_ID", defaultObjectStorageAccessKeyID),
			SecretAccessKey: envOrDefault("OBJECT_STORAGE_SECRET_ACCESS_KEY", defaultObjectStorageSecretKey),
			UsePathStyle:    boolEnvOrDefault("OBJECT_STORAGE_USE_PATH_STYLE", defaultObjectStorageUsePathStyle),
		},
		MaxJobAttempts:     intEnvOrDefault("JUDGE_MAX_JOB_ATTEMPTS", defaultMaxJobAttempts),
		JobLeaseDuration:   jobLeaseDuration,
		JobLeaseRenewAfter: jobLeaseRenewAfter,
		WorkerPollInterval: durationEnvOrDefault("JUDGE_POLL_INTERVAL", defaultWorkerPollInterval),
		WorkerRetryDelay:   durationEnvOrDefault("JUDGE_RETRY_DELAY", defaultWorkerRetryDelay),
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

func intEnvOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func durationEnvOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
