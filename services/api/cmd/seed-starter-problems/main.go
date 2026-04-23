package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	appconfig "github.com/lawrencefmm/cabugi/services/api/internal/config"
	"github.com/lawrencefmm/cabugi/services/api/internal/starterproblems"
)

type objectStorageClient interface {
	CreateBucket(context.Context, *s3.CreateBucketInput, ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
	HeadBucket(context.Context, *s3.HeadBucketInput, ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

const seedAuthorSubject = "seed/official-problems"
const seedAuthorHandle = "official_seed"
const seedAuthorDisplayName = "Cabugi Official"

func main() {
	ctx := context.Background()
	cfg := appconfig.Load()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create postgres pool: %v", err)
	}
	defer pool.Close()

	storage, err := newObjectStorageClient(cfg.ObjectStorage)
	if err != nil {
		log.Fatalf("create object storage client: %v", err)
	}

	seeds, err := starterproblems.StarterProblems()
	if err != nil {
		log.Fatalf("load starter problem definitions: %v", err)
	}

	if err := ensureBucket(ctx, storage, cfg.ObjectStorage.Bucket); err != nil {
		log.Fatalf("ensure hidden test bucket: %v", err)
	}

	seedAuthorID, err := ensureSeedAuthor(ctx, pool)
	if err != nil {
		log.Fatalf("ensure seed author: %v", err)
	}

	for _, seed := range seeds {
		if err := uploadBundle(ctx, storage, cfg.ObjectStorage.Bucket, seed.HiddenTestBundleKey, seed.HiddenTestBundleJSON); err != nil {
			log.Fatalf("upload %s hidden bundle: %v", seed.Slug, err)
		}
		if err := seedProblem(ctx, pool, seedAuthorID, seed); err != nil {
			log.Fatalf("seed %s: %v", seed.Slug, err)
		}
		log.Printf("seeded starter problem %s", seed.Slug)
	}
}

func newObjectStorageClient(cfg appconfig.ObjectStorageConfig) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, err
	}
	if cfg.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(cfg.Endpoint)
	}

	return s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
	}), nil
}

func ensureBucket(ctx context.Context, client objectStorageClient, bucket string) error {
	if _, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err == nil {
		return nil
	}

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}

	if _, headErr := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); headErr == nil {
		return nil
	}

	return err
}

func uploadBundle(ctx context.Context, client objectStorageClient, bucket string, key string, contents []byte) error {
	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(contents),
		ContentType: aws.String("application/json"),
	})
	return err
}

func ensureSeedAuthor(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var userID string
	err := pool.QueryRow(ctx, `
INSERT INTO users (auth_subject, handle, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (auth_subject)
DO UPDATE SET updated_at = NOW()
RETURNING id::text
`, seedAuthorSubject, seedAuthorHandle, seedAuthorDisplayName).Scan(&userID)
	if err != nil {
		return "", err
	}

	_, err = pool.Exec(ctx, `
INSERT INTO user_roles (user_id, role)
VALUES ($1::uuid, 'admin')
ON CONFLICT (user_id, role) DO NOTHING
`, userID)
	if err != nil {
		return "", err
	}

	return userID, nil
}

func seedProblem(ctx context.Context, pool *pgxpool.Pool, seedAuthorID string, seed starterproblems.ProblemSeed) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var problemID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM problems WHERE slug = $1 LIMIT 1`, seed.Slug).Scan(&problemID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
INSERT INTO problems (slug, created_by_user_id)
VALUES ($1, $2::uuid)
RETURNING id::text
`, seed.Slug, seedAuthorID).Scan(&problemID)
	}
	if err != nil {
		return err
	}

	var publishedVersionNumber int
	err = tx.QueryRow(ctx, `
SELECT version_number
FROM problem_versions
WHERE problem_id = $1::uuid AND lifecycle_status = 'published'
LIMIT 1
`, problemID).Scan(&publishedVersionNumber)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if err == nil && publishedVersionNumber != 1 {
		return fmt.Errorf("problem %s already has a published version %d; reseed expects version 1", seed.Slug, publishedVersionNumber)
	}

	_, err = tx.Exec(ctx, `
INSERT INTO problem_versions (
  problem_id,
  version_number,
  lifecycle_status,
  title,
  statement_markdown,
  input_markdown,
  output_markdown,
  constraints_markdown,
  notes_markdown,
  time_limit_ms,
  memory_limit_mb,
  hidden_test_bundle_key,
  hidden_test_bundle_sha256,
  created_by_user_id,
  reviewer_user_id,
  moderation_notes,
  published_at
)
VALUES (
  $1::uuid,
  1,
  'published',
  $2,
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  $9,
  $10,
  $11,
  $12::uuid,
  $12::uuid,
  'Seeded official starter problem',
  NOW()
)
ON CONFLICT (problem_id, version_number)
DO UPDATE SET
  lifecycle_status = 'published',
  title = EXCLUDED.title,
  statement_markdown = EXCLUDED.statement_markdown,
  input_markdown = EXCLUDED.input_markdown,
  output_markdown = EXCLUDED.output_markdown,
  constraints_markdown = EXCLUDED.constraints_markdown,
  notes_markdown = EXCLUDED.notes_markdown,
  time_limit_ms = EXCLUDED.time_limit_ms,
  memory_limit_mb = EXCLUDED.memory_limit_mb,
  hidden_test_bundle_key = EXCLUDED.hidden_test_bundle_key,
  hidden_test_bundle_sha256 = EXCLUDED.hidden_test_bundle_sha256,
  reviewer_user_id = EXCLUDED.reviewer_user_id,
  moderation_notes = EXCLUDED.moderation_notes,
  published_at = COALESCE(problem_versions.published_at, NOW()),
  archived_at = NULL,
  updated_at = NOW()
`, problemID, seed.Title, seed.StatementMarkdown, seed.InputMarkdown, seed.OutputMarkdown, seed.ConstraintsMarkdown, seed.NotesMarkdown, seed.TimeLimitMs, seed.MemoryLimitMb, seed.HiddenTestBundleKey, seed.HiddenTestBundleSHA256, seedAuthorID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
