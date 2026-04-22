package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	judgeconfig "github.com/lawrencefmm/cabugi/services/judge/internal/config"
	"github.com/lawrencefmm/cabugi/services/judge/internal/spike"
)

type BundleLoader interface {
	LoadCases(context.Context, string, string) ([]spike.TestCase, error)
}

type objectStorageClient interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type S3BundleLoader struct {
	client objectStorageClient
	bucket string
}

type fileBundle struct {
	Cases []spike.TestCase `json:"cases"`
}

func NewS3BundleLoader(cfg judgeconfig.ObjectStorageConfig) (S3BundleLoader, error) {
	if cfg.Bucket == "" {
		return S3BundleLoader{}, errors.New("object storage bucket is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return S3BundleLoader{}, fmt.Errorf("load object storage config: %w", err)
	}

	if cfg.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
	})

	return S3BundleLoader{client: client, bucket: cfg.Bucket}, nil
}

func (loader S3BundleLoader) LoadCases(ctx context.Context, key string, expectedSHA256 string) ([]spike.TestCase, error) {
	object, err := loader.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(loader.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("download test bundle %s: %w", key, err)
	}
	defer object.Body.Close()

	contents, err := io.ReadAll(object.Body)
	if err != nil {
		return nil, fmt.Errorf("read test bundle %s: %w", key, err)
	}

	if err := verifyBundleChecksum(contents, expectedSHA256); err != nil {
		return nil, fmt.Errorf("verify test bundle %s: %w", key, err)
	}

	var bundle fileBundle
	if err := json.Unmarshal(contents, &bundle); err != nil {
		return nil, fmt.Errorf("decode test bundle %s: %w", key, err)
	}

	return bundle.Cases, nil
}

func verifyBundleChecksum(contents []byte, expected string) error {
	if expected == "" {
		return errors.New("missing expected checksum")
	}

	sum := sha256.Sum256(contents)
	actual := hex.EncodeToString(sum[:])
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", actual, expected)
	}

	return nil
}
