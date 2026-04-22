package problems

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound           = errors.New("published problem not found")
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

type Store interface {
	ListPublishedProblems(context.Context) ([]PublishedProblemSummary, error)
	GetPublishedProblemBySlug(context.Context, string) (PublishedProblemDetail, error)
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
