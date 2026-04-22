package problems

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound           = errors.New("published problem not found")
	ErrDraftNotFound      = errors.New("problem draft not found")
	ErrProblemSlugTaken   = errors.New("problem slug already exists")
	ErrStoreNotConfigured = errors.New("problem store not configured")
)

type PublishedProblemSummary struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	TimeLimitMs   int    `json:"timeLimitMs"`
	MemoryLimitMB int    `json:"memoryLimitMb"`
}

type PublishedProblemDetail struct {
	Slug                string `json:"slug"`
	Title               string `json:"title"`
	StatementMarkdown   string `json:"statementMarkdown"`
	InputMarkdown       string `json:"inputMarkdown"`
	OutputMarkdown      string `json:"outputMarkdown"`
	ConstraintsMarkdown string `json:"constraintsMarkdown"`
	NotesMarkdown       string `json:"notesMarkdown"`
	TimeLimitMs         int    `json:"timeLimitMs"`
	MemoryLimitMB       int    `json:"memoryLimitMb"`
}

type DraftProblem struct {
	Slug                   string `json:"slug"`
	VersionNumber          int    `json:"versionNumber"`
	LifecycleStatus        string `json:"lifecycleStatus"`
	Title                  string `json:"title"`
	StatementMarkdown      string `json:"statementMarkdown"`
	InputMarkdown          string `json:"inputMarkdown"`
	OutputMarkdown         string `json:"outputMarkdown"`
	ConstraintsMarkdown    string `json:"constraintsMarkdown"`
	NotesMarkdown          string `json:"notesMarkdown"`
	TimeLimitMs            int    `json:"timeLimitMs"`
	MemoryLimitMB          int    `json:"memoryLimitMb"`
	HiddenTestBundleKey    string `json:"hiddenTestBundleKey"`
	HiddenTestBundleSHA256 string `json:"hiddenTestBundleSha256"`
}

type CreateDraftInput struct {
	UserID                 string
	Slug                   string
	Title                  string
	StatementMarkdown      string
	InputMarkdown          string
	OutputMarkdown         string
	ConstraintsMarkdown    string
	NotesMarkdown          string
	TimeLimitMs            int
	MemoryLimitMB          int
	HiddenTestBundleKey    string
	HiddenTestBundleSHA256 string
}

type UpdateDraftInput struct {
	ActorUserID            string
	AllowStaff             bool
	Slug                   string
	Title                  string
	StatementMarkdown      string
	InputMarkdown          string
	OutputMarkdown         string
	ConstraintsMarkdown    string
	NotesMarkdown          string
	TimeLimitMs            int
	MemoryLimitMB          int
	HiddenTestBundleKey    string
	HiddenTestBundleSHA256 string
}

type Store interface {
	ListPublishedProblems(context.Context) ([]PublishedProblemSummary, error)
	GetPublishedProblemBySlug(context.Context, string) (PublishedProblemDetail, error)
	CreateDraft(context.Context, CreateDraftInput) (DraftProblem, error)
	GetDraftBySlug(context.Context, string, string, bool) (DraftProblem, error)
	UpdateDraft(context.Context, UpdateDraftInput) (DraftProblem, error)
}

type DisabledStore struct{}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresStore struct {
	pool *pgxpool.Pool
	db   queryer
}

const listPublishedProblemsSQL = `
SELECT
  p.slug,
  pv.title,
  pv.time_limit_ms,
  pv.memory_limit_mb
FROM problem_versions pv
JOIN problems p ON p.id = pv.problem_id
WHERE pv.lifecycle_status = 'published'
ORDER BY p.slug ASC
`

const getPublishedProblemBySlugSQL = `
SELECT
  p.slug,
  pv.title,
  pv.statement_markdown,
  pv.input_markdown,
  pv.output_markdown,
  pv.constraints_markdown,
  pv.notes_markdown,
  pv.time_limit_ms,
  pv.memory_limit_mb
FROM problem_versions pv
JOIN problems p ON p.id = pv.problem_id
WHERE pv.lifecycle_status = 'published' AND p.slug = $1
LIMIT 1
`

const createDraftSQL = `
WITH inserted_problem AS (
  INSERT INTO problems (slug, created_by_user_id)
  VALUES ($1, $2::uuid)
  RETURNING id
), inserted_version AS (
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
    created_by_user_id
  )
  SELECT
    id,
    1,
    'draft',
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $2::uuid
  FROM inserted_problem
  RETURNING problem_id, version_number, lifecycle_status::text, title, statement_markdown, input_markdown, output_markdown, constraints_markdown, notes_markdown, time_limit_ms, memory_limit_mb, hidden_test_bundle_key, hidden_test_bundle_sha256
)
SELECT
  p.slug,
  inserted_version.version_number,
  inserted_version.lifecycle_status,
  inserted_version.title,
  inserted_version.statement_markdown,
  inserted_version.input_markdown,
  inserted_version.output_markdown,
  inserted_version.constraints_markdown,
  inserted_version.notes_markdown,
  inserted_version.time_limit_ms,
  inserted_version.memory_limit_mb,
  inserted_version.hidden_test_bundle_key,
  inserted_version.hidden_test_bundle_sha256
FROM inserted_version
JOIN problems p ON p.id = inserted_version.problem_id
`

const getDraftBySlugSQL = `
SELECT
  p.slug,
  pv.version_number,
  pv.lifecycle_status::text,
  pv.title,
  pv.statement_markdown,
  pv.input_markdown,
  pv.output_markdown,
  pv.constraints_markdown,
  pv.notes_markdown,
  pv.time_limit_ms,
  pv.memory_limit_mb,
  pv.hidden_test_bundle_key,
  pv.hidden_test_bundle_sha256
FROM problem_versions pv
JOIN problems p ON p.id = pv.problem_id
WHERE p.slug = $1 AND pv.lifecycle_status = 'draft' AND ($2 OR pv.created_by_user_id = $3::uuid)
ORDER BY pv.version_number DESC
LIMIT 1
`

const updateDraftSQL = `
WITH target_version AS (
  SELECT pv.id
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE p.slug = $1 AND pv.lifecycle_status = 'draft' AND ($2 OR pv.created_by_user_id = $3::uuid)
  ORDER BY pv.version_number DESC
  LIMIT 1
)
UPDATE problem_versions pv
SET
  title = $4,
  statement_markdown = $5,
  input_markdown = $6,
  output_markdown = $7,
  constraints_markdown = $8,
  notes_markdown = $9,
  time_limit_ms = $10,
  memory_limit_mb = $11,
  hidden_test_bundle_key = $12,
  hidden_test_bundle_sha256 = $13,
  updated_at = NOW()
FROM target_version, problems p
WHERE pv.id = target_version.id AND p.id = pv.problem_id
RETURNING
  p.slug,
  pv.version_number,
  pv.lifecycle_status::text,
  pv.title,
  pv.statement_markdown,
  pv.input_markdown,
  pv.output_markdown,
  pv.constraints_markdown,
  pv.notes_markdown,
  pv.time_limit_ms,
  pv.memory_limit_mb,
  pv.hidden_test_bundle_key,
  pv.hidden_test_bundle_sha256
`

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	return &PostgresStore{pool: pool, db: pool}, nil
}

func NewPostgresStoreFromQuerier(db queryer) *PostgresStore {
	return &PostgresStore{db: db}
}

func (store *PostgresStore) ListPublishedProblems(ctx context.Context) ([]PublishedProblemSummary, error) {
	rows, err := store.db.Query(ctx, listPublishedProblemsSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	problems := make([]PublishedProblemSummary, 0)
	for rows.Next() {
		var problem PublishedProblemSummary
		if err := rows.Scan(&problem.Slug, &problem.Title, &problem.TimeLimitMs, &problem.MemoryLimitMB); err != nil {
			return nil, err
		}

		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return problems, nil
}

func (store *PostgresStore) GetPublishedProblemBySlug(ctx context.Context, slug string) (PublishedProblemDetail, error) {
	var problem PublishedProblemDetail
	err := store.db.QueryRow(ctx, getPublishedProblemBySlugSQL, slug).Scan(
		&problem.Slug,
		&problem.Title,
		&problem.StatementMarkdown,
		&problem.InputMarkdown,
		&problem.OutputMarkdown,
		&problem.ConstraintsMarkdown,
		&problem.NotesMarkdown,
		&problem.TimeLimitMs,
		&problem.MemoryLimitMB,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PublishedProblemDetail{}, ErrNotFound
	}

	return problem, err
}

func (store *PostgresStore) CreateDraft(ctx context.Context, input CreateDraftInput) (DraftProblem, error) {
	var problem DraftProblem
	err := store.db.QueryRow(
		ctx,
		createDraftSQL,
		input.Slug,
		input.UserID,
		input.Title,
		input.StatementMarkdown,
		input.InputMarkdown,
		input.OutputMarkdown,
		input.ConstraintsMarkdown,
		input.NotesMarkdown,
		input.TimeLimitMs,
		input.MemoryLimitMB,
		input.HiddenTestBundleKey,
		input.HiddenTestBundleSHA256,
	).Scan(
		&problem.Slug,
		&problem.VersionNumber,
		&problem.LifecycleStatus,
		&problem.Title,
		&problem.StatementMarkdown,
		&problem.InputMarkdown,
		&problem.OutputMarkdown,
		&problem.ConstraintsMarkdown,
		&problem.NotesMarkdown,
		&problem.TimeLimitMs,
		&problem.MemoryLimitMB,
		&problem.HiddenTestBundleKey,
		&problem.HiddenTestBundleSHA256,
	)
	if isProblemSlugConflict(err) {
		return DraftProblem{}, ErrProblemSlugTaken
	}

	return problem, err
}

func (store *PostgresStore) GetDraftBySlug(ctx context.Context, slug string, actorUserID string, allowStaff bool) (DraftProblem, error) {
	var problem DraftProblem
	err := store.db.QueryRow(ctx, getDraftBySlugSQL, slug, allowStaff, actorUserID).Scan(
		&problem.Slug,
		&problem.VersionNumber,
		&problem.LifecycleStatus,
		&problem.Title,
		&problem.StatementMarkdown,
		&problem.InputMarkdown,
		&problem.OutputMarkdown,
		&problem.ConstraintsMarkdown,
		&problem.NotesMarkdown,
		&problem.TimeLimitMs,
		&problem.MemoryLimitMB,
		&problem.HiddenTestBundleKey,
		&problem.HiddenTestBundleSHA256,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DraftProblem{}, ErrDraftNotFound
	}

	return problem, err
}

func (store *PostgresStore) UpdateDraft(ctx context.Context, input UpdateDraftInput) (DraftProblem, error) {
	var problem DraftProblem
	err := store.db.QueryRow(
		ctx,
		updateDraftSQL,
		input.Slug,
		input.AllowStaff,
		input.ActorUserID,
		input.Title,
		input.StatementMarkdown,
		input.InputMarkdown,
		input.OutputMarkdown,
		input.ConstraintsMarkdown,
		input.NotesMarkdown,
		input.TimeLimitMs,
		input.MemoryLimitMB,
		input.HiddenTestBundleKey,
		input.HiddenTestBundleSHA256,
	).Scan(
		&problem.Slug,
		&problem.VersionNumber,
		&problem.LifecycleStatus,
		&problem.Title,
		&problem.StatementMarkdown,
		&problem.InputMarkdown,
		&problem.OutputMarkdown,
		&problem.ConstraintsMarkdown,
		&problem.NotesMarkdown,
		&problem.TimeLimitMs,
		&problem.MemoryLimitMB,
		&problem.HiddenTestBundleKey,
		&problem.HiddenTestBundleSHA256,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DraftProblem{}, ErrDraftNotFound
	}

	return problem, err
}

func (store *PostgresStore) Close() {
	if store.pool != nil {
		store.pool.Close()
	}
}

func (DisabledStore) ListPublishedProblems(context.Context) ([]PublishedProblemSummary, error) {
	return nil, ErrStoreNotConfigured
}

func (DisabledStore) GetPublishedProblemBySlug(context.Context, string) (PublishedProblemDetail, error) {
	return PublishedProblemDetail{}, ErrStoreNotConfigured
}

func (DisabledStore) CreateDraft(context.Context, CreateDraftInput) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func (DisabledStore) GetDraftBySlug(context.Context, string, string, bool) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func (DisabledStore) UpdateDraft(context.Context, UpdateDraftInput) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func isProblemSlugConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}

	return false
}
