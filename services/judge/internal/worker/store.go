package worker

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

const claimNextJobSQL = `
WITH next_job AS (
  SELECT submission_id
  FROM submission_jobs
  WHERE claimed_at IS NULL AND available_at <= NOW()
  ORDER BY created_at ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE submission_jobs sj
SET claimed_at = NOW(), attempts = sj.attempts + 1
FROM next_job
WHERE sj.submission_id = next_job.submission_id
RETURNING sj.submission_id::text
`

const loadSubmissionJobSQL = `
SELECT
  s.id::text,
  s.language::text,
  s.source_code,
  pv.hidden_test_bundle_key,
  pv.time_limit_ms
FROM submissions s
JOIN problem_versions pv ON pv.id = s.problem_version_id
WHERE s.id = $1::uuid
`

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}

	return &PostgresStore{pool: pool}, nil
}

func (store *PostgresStore) Close() {
	if store.pool != nil {
		store.pool.Close()
	}
}

func (store *PostgresStore) ClaimNextJob(ctx context.Context) (SubmissionJob, error) {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SubmissionJob{}, err
	}
	defer tx.Rollback(ctx)

	var submissionID string
	err = tx.QueryRow(ctx, claimNextJobSQL).Scan(&submissionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionJob{}, ErrNoJobs
	}
	if err != nil {
		return SubmissionJob{}, err
	}

	var job SubmissionJob
	var timeLimitMS int
	err = tx.QueryRow(ctx, loadSubmissionJobSQL, submissionID).Scan(&job.SubmissionID, &job.Language, &job.SourceCode, &job.BundleKey, &timeLimitMS)
	if err != nil {
		return SubmissionJob{}, err
	}
	job.TimeLimit = time.Duration(timeLimitMS) * time.Millisecond

	if err := tx.Commit(ctx); err != nil {
		return SubmissionJob{}, err
	}

	return job, nil
}

func (store *PostgresStore) MarkSubmissionRunning(ctx context.Context, submissionID string) error {
	_, err := store.pool.Exec(ctx, `UPDATE submissions SET status = 'running', started_at = NOW() WHERE id = $1::uuid`, submissionID)
	return err
}

func (store *PostgresStore) CompleteSubmission(ctx context.Context, submissionID string, status string, results []CaseResult) error {
	tx, err := store.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for index, result := range results {
		_, err := tx.Exec(ctx, `INSERT INTO submission_results (submission_id, test_index, verdict, execution_time_ms, stdout_excerpt, stderr_excerpt) VALUES ($1::uuid, $2, $3::submission_case_status, $4, $5, $6)`, submissionID, index, result.Verdict, result.ExecutionTimeMS, result.StdoutExcerpt, result.StderrExcerpt)
		if err != nil {
			return err
		}
	}

	passedTests := 0
	for _, result := range results {
		if result.Verdict == "accepted" {
			passedTests++
		}
	}

	_, err = tx.Exec(ctx, `UPDATE submissions SET status = $2::submission_status, finished_at = NOW(), total_tests = $3, passed_tests = $4 WHERE id = $1::uuid`, submissionID, status, len(results), passedTests)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `DELETE FROM submission_jobs WHERE submission_id = $1::uuid`, submissionID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
