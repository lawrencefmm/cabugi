package problems

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	appconfig "github.com/lawrencefmm/cabugi/services/api/internal/config"
)

var (
	ErrBundleValidatorNotConfigured = errors.New("hidden test bundle validator not configured")
	ErrBundleUploaderNotConfigured  = errors.New("hidden test bundle uploader not configured")
	ErrBundleNotFound               = errors.New("hidden test bundle not found")
	ErrBundleChecksumMismatch       = errors.New("hidden test bundle checksum mismatch")
	ErrInvalidBundleChecksum        = errors.New("invalid hidden test bundle checksum")
	ErrInvalidBundleContents        = errors.New("invalid hidden test bundle contents")
)

type BundleValidator interface {
	ValidateBundle(context.Context, string, string) error
	UploadBundle(context.Context, string, string, []byte) (UploadedBundle, error)
}

type UploadedBundle struct {
	Key    string `json:"hiddenTestBundleKey"`
	SHA256 string `json:"hiddenTestBundleSha256"`
}

type objectStorageClient interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

type S3BundleValidator struct {
	client objectStorageClient
	bucket string
}

type DisabledBundleValidator struct{}

type fileBundle struct {
	Cases []fileBundleCase `json:"cases"`
}

type fileBundleCase struct {
	Input          string
	ExpectedOutput string
}

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

func (validator S3BundleValidator) CheckReady(ctx context.Context) error {
	_, err := validator.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(validator.bucket)})
	return err
}

func (validator S3BundleValidator) UploadBundle(ctx context.Context, userID string, fileName string, contents []byte) (UploadedBundle, error) {
	if err := ValidateBundleContents(contents); err != nil {
		return UploadedBundle{}, err
	}

	checksum := bundleSHA256(contents)
	key, err := bundleObjectKey(userID, fileName, checksum)
	if err != nil {
		return UploadedBundle{}, err
	}

	_, err = validator.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(validator.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(contents),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return UploadedBundle{}, err
	}

	return UploadedBundle{Key: key, SHA256: checksum}, nil
}

func (DisabledBundleValidator) ValidateBundle(context.Context, string, string) error {
	return ErrBundleValidatorNotConfigured
}

func (DisabledBundleValidator) UploadBundle(context.Context, string, string, []byte) (UploadedBundle, error) {
	return UploadedBundle{}, ErrBundleUploaderNotConfigured
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

func ValidateBundleContents(contents []byte) error {
	var bundle fileBundle
	if err := json.Unmarshal(contents, &bundle); err != nil {
		return ErrInvalidBundleContents
	}
	if len(bundle.Cases) == 0 {
		return ErrInvalidBundleContents
	}

	return nil
}

func bundleSHA256(contents []byte) string {
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

func bundleObjectKey(userID string, fileName string, checksum string) (string, error) {
	randomSuffix := make([]byte, 6)
	if _, err := rand.Read(randomSuffix); err != nil {
		return "", err
	}

	userSegment := sanitizeBundleKeySegment(userID)
	nameSegment := sanitizeBundleFileName(fileName)
	return fmt.Sprintf("problem-drafts/%s/%s-%s-%s", userSegment, checksum[:12], hex.EncodeToString(randomSuffix), nameSegment), nil
}

func sanitizeBundleKeySegment(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return "unknown-user"
	}

	var builder strings.Builder
	for _, char := range trimmed {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('-')
	}

	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "unknown-user"
	}

	return result
}

func sanitizeBundleFileName(fileName string) string {
	name := strings.TrimSpace(filepath.Base(fileName))
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "hidden-tests.json"
	}

	name = strings.ToLower(name)
	var builder strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('-')
	}

	result := strings.Trim(builder.String(), "-.")
	if result == "" {
		result = "hidden-tests"
	}
	if filepath.Ext(result) == "" {
		result += ".json"
	}

	return result
}
