package submissions

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSubmissionNotFound = errors.New("submission not found")
	ErrProblemNotFound    = errors.New("published problem not found")
	ErrInvalidLanguage    = errors.New("invalid submission language")
	ErrStoreNotConfigured = errors.New("submission store not configured")
)

type Summary struct {
	ID          string    `json:"id"`
	ProblemSlug string    `json:"problemSlug"`
	Language    string    `json:"language"`
	Status      string    `json:"status"`
	QueuedAt    time.Time `json:"queuedAt"`
}

type Result struct {
	TestIndex       int    `json:"testIndex"`
	Verdict         string `json:"verdict"`
	ExecutionTimeMS int    `json:"executionTimeMs"`
	MemoryBytes     int64  `json:"memoryBytes"`
	StdoutExcerpt   string `json:"stdoutExcerpt"`
	StderrExcerpt   string `json:"stderrExcerpt"`
}

type Detail struct {
	ID                   string    `json:"id"`
	ProblemSlug          string    `json:"problemSlug"`
	Language             string    `json:"language"`
	Status               string    `json:"status"`
	QueuedAt             time.Time `json:"queuedAt"`
	TotalTests           int       `json:"totalTests"`
	PassedTests          int       `json:"passedTests"`
	CompileOutputExcerpt string    `json:"compileOutputExcerpt"`
	Results              []Result  `json:"results"`
}

type CreateInput struct {
	UserID      string
	ProblemSlug string
	Language    string
	SourceCode  string
}

type Store interface {
	CreateSubmission(context.Context, CreateInput) (Summary, error)
	ListSubmissions(context.Context, string) ([]Summary, error)
	GetSubmissionByID(context.Context, string, string) (Detail, error)
}

type DisabledStore struct{}

type rowQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresStore struct {
	pool *pgxpool.Pool
	db   rowQueryer
}

const queueDepthSQL = `
SELECT COUNT(*)
FROM submission_jobs
WHERE available_at < 'infinity'::timestamptz
`

const createSubmissionSQL = `
WITH published_problem AS (
  SELECT pv.id AS problem_version_id, p.slug
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE pv.lifecycle_status = 'published' AND p.slug = $2
  LIMIT 1
), inserted_submission AS (
  INSERT INTO submissions (user_id, problem_version_id, language, source_code)
  SELECT $1::uuid, published_problem.problem_version_id, $3::submission_language, $4
  FROM published_problem
  RETURNING id, problem_version_id, language::text AS language, status::text AS status, queued_at
), inserted_job AS (
  INSERT INTO submission_jobs (submission_id)
  SELECT id FROM inserted_submission
)
SELECT inserted_submission.id::text, published_problem.slug, inserted_submission.language, inserted_submission.status, inserted_submission.queued_at
FROM inserted_submission
JOIN published_problem ON TRUE
`

const getSubmissionByIDSQL = `
SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at, s.total_tests, s.passed_tests, s.compile_output_excerpt
FROM submissions s
JOIN problem_versions pv ON pv.id = s.problem_version_id
JOIN problems p ON p.id = pv.problem_id
WHERE s.id = $1::uuid AND s.user_id = $2::uuid
LIMIT 1
`

const listSubmissionResultsSQL = `
SELECT test_index, verdict::text, execution_time_ms, memory_bytes, stdout_excerpt, stderr_excerpt
FROM submission_results
WHERE submission_id = $1::uuid
ORDER BY test_index ASC
`

const listSubmissionsSQL = `
SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at
FROM submissions s
JOIN problem_versions pv ON pv.id = s.problem_version_id
JOIN problems p ON p.id = pv.problem_id
WHERE s.user_id = $1::uuid
ORDER BY s.queued_at DESC, s.id DESC
`

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool, db: pool}, nil
}

func NewPostgresStoreFromQuerier(db rowQueryer) *PostgresStore {
	return &PostgresStore{db: db}
}

func (store *PostgresStore) CreateSubmission(ctx context.Context, input CreateInput) (Summary, error) {
	if !SupportedLanguage(input.Language) {
		return Summary{}, ErrInvalidLanguage
	}

	var submission Summary
	err := store.db.QueryRow(ctx, createSubmissionSQL, input.UserID, input.ProblemSlug, input.Language, input.SourceCode).Scan(
		&submission.ID,
		&submission.ProblemSlug,
		&submission.Language,
		&submission.Status,
		&submission.QueuedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrProblemNotFound
	}

	return submission, err
}

func (store *PostgresStore) GetSubmissionByID(ctx context.Context, submissionID string, userID string) (Detail, error) {
	submission := Detail{Results: make([]Result, 0)}
	err := store.db.QueryRow(ctx, getSubmissionByIDSQL, submissionID, userID).Scan(
		&submission.ID,
		&submission.ProblemSlug,
		&submission.Language,
		&submission.Status,
		&submission.QueuedAt,
		&submission.TotalTests,
		&submission.PassedTests,
		&submission.CompileOutputExcerpt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detail{}, ErrSubmissionNotFound
	}
	if err != nil {
		return Detail{}, err
	}

	rows, err := store.db.Query(ctx, listSubmissionResultsSQL, submissionID)
	if err != nil {
		return Detail{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var result Result
		if err := rows.Scan(
			&result.TestIndex,
			&result.Verdict,
			&result.ExecutionTimeMS,
			&result.MemoryBytes,
			&result.StdoutExcerpt,
			&result.StderrExcerpt,
		); err != nil {
			return Detail{}, err
		}

		submission.Results = append(submission.Results, result)
	}

	if err := rows.Err(); err != nil {
		return Detail{}, err
	}

	return submission, nil
}

func (store *PostgresStore) ListSubmissions(ctx context.Context, userID string) ([]Summary, error) {
	rows, err := store.db.Query(ctx, listSubmissionsSQL, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	submissions := make([]Summary, 0)
	for rows.Next() {
		var submission Summary
		if err := rows.Scan(&submission.ID, &submission.ProblemSlug, &submission.Language, &submission.Status, &submission.QueuedAt); err != nil {
			return nil, err
		}

		submissions = append(submissions, submission)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return submissions, nil
}

func (store *PostgresStore) QueueDepth(ctx context.Context) (int, error) {
	var depth int
	err := store.db.QueryRow(ctx, queueDepthSQL).Scan(&depth)
	return depth, err
}

func (store *PostgresStore) Close() {
	if store.pool != nil {
		store.pool.Close()
	}
}

func (DisabledStore) CreateSubmission(context.Context, CreateInput) (Summary, error) {
	return Summary{}, ErrStoreNotConfigured
}

func (DisabledStore) ListSubmissions(context.Context, string) ([]Summary, error) {
	return nil, ErrStoreNotConfigured
}

func (DisabledStore) GetSubmissionByID(context.Context, string, string) (Detail, error) {
	return Detail{}, ErrStoreNotConfigured
}

func (DisabledStore) QueueDepth(context.Context) (int, error) {
	return 0, ErrStoreNotConfigured
}
