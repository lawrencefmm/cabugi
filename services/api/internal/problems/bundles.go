package problems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/lawrencefmm/cabugi/services/api/internal/config"
)

var (
	ErrBundleValidatorNotConfigured = errors.New("hidden test bundle validator not configured")
	ErrBundleNotFound               = errors.New("hidden test bundle not found")
	ErrBundleChecksumMismatch       = errors.New("hidden test bundle checksum mismatch")
	ErrInvalidBundleChecksum        = errors.New("invalid hidden test bundle checksum")
)

type BundleValidator interface {
	ValidateBundle(context.Context, string, string) error
}

type objectStorageClient interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3BundleValidator struct {
	client objectStorageClient
	bucket string
}

type DisabledBundleValidator struct{}

func NewS3BundleValidator(cfg appconfig.ObjectStorageConfig) (S3BundleValidator, error) {
	if cfg.Bucket == "" {
		return S3BundleValidator{}, errors.New("object storage bucket is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return S3BundleValidator{}, fmt.Errorf("load object storage config: %w", err)
	}

	if cfg.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
	})

	return S3BundleValidator{client: client, bucket: cfg.Bucket}, nil
}

func (validator S3BundleValidator) ValidateBundle(ctx context.Context, key string, expectedSHA256 string) error {
	object, err := validator.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(validator.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("%w: %s", ErrBundleNotFound, key)
	}
	defer object.Body.Close()

	contents, err := io.ReadAll(object.Body)
	if err != nil {
		return err
	}

	if err := validateBundleChecksum(contents, expectedSHA256); err != nil {
		return err
	}

	return nil
}

func (DisabledBundleValidator) ValidateBundle(context.Context, string, string) error {
	return ErrBundleValidatorNotConfigured
}

func ValidateBundleMetadata(key string, checksum string) error {
	trimmedKey := strings.TrimSpace(key)
	trimmedChecksum := strings.TrimSpace(checksum)

	if trimmedKey == "" && trimmedChecksum == "" {
		return nil
	}
	if trimmedKey == "" || trimmedChecksum == "" {
		return ErrInvalidBundleChecksum
	}
	if len(trimmedChecksum) != 64 {
		return ErrInvalidBundleChecksum
	}
	for _, char := range trimmedChecksum {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return ErrInvalidBundleChecksum
		}
	}

	return nil
}

func validateBundleChecksum(contents []byte, expected string) error {
	if err := ValidateBundleMetadata("bundle", expected); err != nil {
		return err
	}

	sum := sha256.Sum256(contents)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("%w: got %s want %s", ErrBundleChecksumMismatch, actual, expected)
	}

	return nil
}
