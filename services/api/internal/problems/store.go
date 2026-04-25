package problems

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound                   = errors.New("published problem not found")
	ErrDraftNotFound              = errors.New("problem draft not found")
	ErrProblemSlugTaken           = errors.New("problem slug already exists")
	ErrInvalidLifecycleTransition = errors.New("invalid problem lifecycle transition")
	ErrDraftNotReadyForReview     = errors.New("problem draft not ready for review")
	ErrInvalidModerationDecision  = errors.New("invalid moderation decision")
	ErrStoreNotConfigured         = errors.New("problem store not configured")
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

type DraftSummary struct {
	Slug                 string     `json:"slug"`
	VersionNumber        int        `json:"versionNumber"`
	LifecycleStatus      string     `json:"lifecycleStatus"`
	Title                string     `json:"title"`
	UpdatedAt            time.Time  `json:"updatedAt"`
	SubmittedForReviewAt *time.Time `json:"submittedForReviewAt"`
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

type ModerationQueueItem struct {
	Slug                 string    `json:"slug"`
	VersionNumber        int       `json:"versionNumber"`
	Title                string    `json:"title"`
	SubmittedForReviewAt time.Time `json:"submittedForReviewAt"`
}

type ModerationDecision string

const (
	DecisionApprove        ModerationDecision = "approve"
	DecisionReject         ModerationDecision = "reject"
	DecisionRequestChanges ModerationDecision = "request_changes"
)

type Store interface {
	ListPublishedProblems(context.Context) ([]PublishedProblemSummary, error)
	GetPublishedProblemBySlug(context.Context, string) (PublishedProblemDetail, error)
	CreateDraft(context.Context, CreateDraftInput) (DraftProblem, error)
	ListDraftsByOwner(context.Context, string) ([]DraftSummary, error)
	GetDraftBySlug(context.Context, string, string, bool) (DraftProblem, error)
	UpdateDraft(context.Context, UpdateDraftInput) (DraftProblem, error)
	SubmitDraftForReview(context.Context, string, string) (DraftProblem, error)
	ListDraftsInReview(context.Context) ([]ModerationQueueItem, error)
	ApplyModerationDecision(context.Context, string, string, ModerationDecision, string) (DraftProblem, error)
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
  RETURNING id, slug
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
JOIN inserted_problem p ON p.id = inserted_version.problem_id
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
WHERE p.slug = $1 AND pv.lifecycle_status IN ('draft', 'in_review') AND ($2 OR pv.created_by_user_id = $3::uuid)
ORDER BY pv.version_number DESC
LIMIT 1
`

const listDraftsByOwnerSQL = `
SELECT
  p.slug,
  pv.version_number,
  pv.lifecycle_status::text,
  pv.title,
  pv.updated_at,
  pv.submitted_for_review_at
FROM problem_versions pv
JOIN problems p ON p.id = pv.problem_id
WHERE pv.created_by_user_id = $1::uuid AND pv.lifecycle_status IN ('draft', 'in_review')
ORDER BY pv.updated_at DESC, p.slug ASC
`

const submitDraftForReviewSQL = `
WITH target_version AS (
  SELECT pv.id
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE p.slug = $1 AND pv.lifecycle_status = 'draft' AND pv.created_by_user_id = $2::uuid
  ORDER BY pv.version_number DESC
  LIMIT 1
)
UPDATE problem_versions pv
SET
  lifecycle_status = 'in_review',
  submitted_for_review_at = NOW(),
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

const listDraftsInReviewSQL = `
SELECT
  p.slug,
  pv.version_number,
  pv.title,
  pv.submitted_for_review_at
FROM problem_versions pv
JOIN problems p ON p.id = pv.problem_id
WHERE pv.lifecycle_status = 'in_review'
ORDER BY pv.submitted_for_review_at ASC, p.slug ASC
`

const getProblemVersionBySlugForModerationSQL = `
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
WHERE p.slug = $1
ORDER BY pv.version_number DESC
LIMIT 1
`

const approveModerationDecisionSQL = `
WITH target_version AS (
  SELECT pv.id
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE p.slug = $1 AND pv.lifecycle_status = 'in_review'
  ORDER BY pv.version_number DESC
  LIMIT 1
)
UPDATE problem_versions pv
SET
  lifecycle_status = 'published',
  reviewer_user_id = $2::uuid,
  moderation_notes = $3,
  published_at = NOW(),
  archived_at = NULL,
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

const rejectModerationDecisionSQL = `
WITH target_version AS (
  SELECT pv.id
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE p.slug = $1 AND pv.lifecycle_status = 'in_review'
  ORDER BY pv.version_number DESC
  LIMIT 1
)
UPDATE problem_versions pv
SET
  lifecycle_status = 'archived',
  reviewer_user_id = $2::uuid,
  moderation_notes = $3,
  published_at = NULL,
  archived_at = NOW(),
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

const requestChangesModerationDecisionSQL = `
WITH target_version AS (
  SELECT pv.id
  FROM problem_versions pv
  JOIN problems p ON p.id = pv.problem_id
  WHERE p.slug = $1 AND pv.lifecycle_status = 'in_review'
  ORDER BY pv.version_number DESC
  LIMIT 1
)
UPDATE problem_versions pv
SET
  lifecycle_status = 'draft',
  reviewer_user_id = $2::uuid,
  moderation_notes = $3,
  submitted_for_review_at = NULL,
  published_at = NULL,
  archived_at = NULL,
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
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
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

func (store *PostgresStore) ListDraftsByOwner(ctx context.Context, ownerUserID string) ([]DraftSummary, error) {
	rows, err := store.db.Query(ctx, listDraftsByOwnerSQL, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]DraftSummary, 0)
	for rows.Next() {
		var item DraftSummary
		var submittedForReviewAt sql.NullTime
		if err := rows.Scan(&item.Slug, &item.VersionNumber, &item.LifecycleStatus, &item.Title, &item.UpdatedAt, &submittedForReviewAt); err != nil {
			return nil, err
		}
		if submittedForReviewAt.Valid {
			submittedAt := submittedForReviewAt.Time
			item.SubmittedForReviewAt = &submittedAt
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
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

func (store *PostgresStore) SubmitDraftForReview(ctx context.Context, slug string, ownerUserID string) (DraftProblem, error) {
	existing, err := store.GetDraftBySlug(ctx, slug, ownerUserID, false)
	if err != nil {
		return DraftProblem{}, err
	}
	if existing.LifecycleStatus != "draft" {
		return DraftProblem{}, ErrInvalidLifecycleTransition
	}
	if existing.HiddenTestBundleKey == "" || existing.HiddenTestBundleSHA256 == "" {
		return DraftProblem{}, ErrDraftNotReadyForReview
	}

	var problem DraftProblem
	err = store.db.QueryRow(ctx, submitDraftForReviewSQL, slug, ownerUserID).Scan(
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
		return DraftProblem{}, ErrInvalidLifecycleTransition
	}

	return problem, err
}

func (store *PostgresStore) ListDraftsInReview(ctx context.Context) ([]ModerationQueueItem, error) {
	rows, err := store.db.Query(ctx, listDraftsInReviewSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ModerationQueueItem, 0)
	for rows.Next() {
		var item ModerationQueueItem
		if err := rows.Scan(&item.Slug, &item.VersionNumber, &item.Title, &item.SubmittedForReviewAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (store *PostgresStore) ApplyModerationDecision(ctx context.Context, slug string, reviewerUserID string, decision ModerationDecision, moderationNotes string) (DraftProblem, error) {
	current, err := store.getProblemVersionForModeration(ctx, slug)
	if err != nil {
		return DraftProblem{}, err
	}
	if current.LifecycleStatus != "in_review" {
		return DraftProblem{}, ErrInvalidLifecycleTransition
	}

	query := moderationDecisionSQL(decision)
	if query == "" {
		return DraftProblem{}, ErrInvalidModerationDecision
	}

	var problem DraftProblem
	err = store.db.QueryRow(ctx, query, slug, reviewerUserID, moderationNotes).Scan(
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
		return DraftProblem{}, ErrInvalidLifecycleTransition
	}

	return problem, err
}

func (store *PostgresStore) getProblemVersionForModeration(ctx context.Context, slug string) (DraftProblem, error) {
	var problem DraftProblem
	err := store.db.QueryRow(ctx, getProblemVersionBySlugForModerationSQL, slug).Scan(
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

func (DisabledStore) ListDraftsByOwner(context.Context, string) ([]DraftSummary, error) {
	return nil, ErrStoreNotConfigured
}

func (DisabledStore) GetDraftBySlug(context.Context, string, string, bool) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func (DisabledStore) UpdateDraft(context.Context, UpdateDraftInput) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func (DisabledStore) SubmitDraftForReview(context.Context, string, string) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func (DisabledStore) ListDraftsInReview(context.Context) ([]ModerationQueueItem, error) {
	return nil, ErrStoreNotConfigured
}

func (DisabledStore) ApplyModerationDecision(context.Context, string, string, ModerationDecision, string) (DraftProblem, error) {
	return DraftProblem{}, ErrStoreNotConfigured
}

func isProblemSlugConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}

	return false
}

func moderationDecisionSQL(decision ModerationDecision) string {
	switch decision {
	case DecisionApprove:
		return approveModerationDecisionSQL
	case DecisionReject:
		return rejectModerationDecisionSQL
	case DecisionRequestChanges:
		return requestChangesModerationDecisionSQL
	default:
		return ""
	}
}
