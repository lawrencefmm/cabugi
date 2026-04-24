package worker

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultJobLeaseDuration = 30 * time.Second

var ErrJobLeaseLost = errors.New("submission job lease lost")

type FailureAction string

const (
	FailureActionRetried  FailureAction = "retried"
	FailureActionTerminal FailureAction = "terminal"
)

type database interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgresStore struct {
	pool           *pgxpool.Pool
	db             database
	maxJobAttempts int
	leaseDuration  time.Duration
	retryDelay     time.Duration
}

const claimNextJobSQL = `
WITH next_job AS (
  SELECT submission_id
  FROM submission_jobs
  WHERE available_at <= NOW() AND (claimed_at IS NULL OR lease_expires_at IS NULL OR lease_expires_at <= NOW())
  ORDER BY created_at ASC
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
UPDATE submission_jobs sj
SET claimed_at = NOW(), lease_token = gen_random_uuid(), lease_expires_at = $1, attempts = sj.attempts + 1
FROM next_job
WHERE sj.submission_id = next_job.submission_id
RETURNING sj.submission_id::text, sj.lease_token::text
`

const loadSubmissionJobSQL = `
SELECT
  s.id::text,
  s.language::text,
  s.source_code,
  pv.hidden_test_bundle_key,
  pv.hidden_test_bundle_sha256,
  pv.time_limit_ms
FROM submissions s
JOIN problem_versions pv ON pv.id = s.problem_version_id
WHERE s.id = $1::uuid
`

const selectJobAttemptsSQL = `
SELECT attempts
FROM submission_jobs
WHERE submission_id = $1::uuid AND lease_token = $2::uuid AND lease_expires_at > NOW()
FOR UPDATE
`

const renewJobLeaseSQL = `
UPDATE submission_jobs
SET lease_expires_at = $3
WHERE submission_id = $1::uuid AND lease_token = $2::uuid AND lease_expires_at > NOW()
`

const requeueSubmissionJobSQL = `
UPDATE submission_jobs
SET claimed_at = NULL, lease_token = NULL, lease_expires_at = NULL, available_at = $2, last_error = $3
WHERE submission_id = $1::uuid
`

const poisonSubmissionJobSQL = `
UPDATE submission_jobs
SET claimed_at = NULL, lease_token = NULL, lease_expires_at = NULL, available_at = 'infinity'::timestamptz, last_error = $2
WHERE submission_id = $1::uuid
`

const lockSubmissionJobLeaseSQL = `
SELECT 1
FROM submission_jobs
WHERE submission_id = $1::uuid AND lease_token = $2::uuid AND lease_expires_at > NOW()
FOR UPDATE
`

const resetSubmissionForRetrySQL = `
UPDATE submissions
SET status = 'queued', started_at = NULL, finished_at = NULL, total_tests = 0, passed_tests = 0
WHERE id = $1::uuid
`

const markSubmissionJudgeFailedSQL = `
UPDATE submissions
SET status = 'judge_failed', finished_at = NOW(), total_tests = 0, passed_tests = 0
WHERE id = $1::uuid
`

func NewPostgresStore(databaseURL string, maxJobAttempts int, retryDelay time.Duration, leaseDuration time.Duration) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if maxJobAttempts <= 0 {
		maxJobAttempts = 3
	}
	if leaseDuration <= 0 {
		leaseDuration = defaultJobLeaseDuration
	}
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	return &PostgresStore{pool: pool, db: pool, maxJobAttempts: maxJobAttempts, leaseDuration: leaseDuration, retryDelay: retryDelay}, nil
}

func NewPostgresStoreFromDatabase(db database, maxJobAttempts int, retryDelay time.Duration, leaseDuration time.Duration) *PostgresStore {
	if maxJobAttempts <= 0 {
		maxJobAttempts = 3
	}
	if leaseDuration <= 0 {
		leaseDuration = defaultJobLeaseDuration
	}
	if retryDelay <= 0 {
		retryDelay = 5 * time.Second
	}

	return &PostgresStore{db: db, maxJobAttempts: maxJobAttempts, leaseDuration: leaseDuration, retryDelay: retryDelay}
}

func (store *PostgresStore) Close() {
	if store.pool != nil {
		store.pool.Close()
	}
}

func (store *PostgresStore) ClaimNextJob(ctx context.Context) (SubmissionJob, error) {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SubmissionJob{}, err
	}
	defer tx.Rollback(ctx)

	var submissionID string
	var leaseToken string
	err = tx.QueryRow(ctx, claimNextJobSQL, time.Now().Add(store.leaseDuration)).Scan(&submissionID, &leaseToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionJob{}, ErrNoJobs
	}
	if err != nil {
		return SubmissionJob{}, err
	}

	var job SubmissionJob
	var timeLimitMS int
	err = tx.QueryRow(ctx, loadSubmissionJobSQL, submissionID).Scan(&job.SubmissionID, &job.Language, &job.SourceCode, &job.BundleKey, &job.BundleSHA256, &timeLimitMS)
	if err != nil {
		return SubmissionJob{}, err
	}
	job.LeaseToken = leaseToken
	job.TimeLimit = time.Duration(timeLimitMS) * time.Millisecond

	if err := tx.Commit(ctx); err != nil {
		return SubmissionJob{}, err
	}

	return job, nil
}

func (store *PostgresStore) RenewJobLease(ctx context.Context, submissionID string, leaseToken string) error {
	result, err := store.db.Exec(ctx, renewJobLeaseSQL, submissionID, leaseToken, time.Now().Add(store.leaseDuration))
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrJobLeaseLost
	}

	return nil
}

func (store *PostgresStore) MarkSubmissionRunning(ctx context.Context, submissionID string) error {
	_, err := store.db.Exec(ctx, `UPDATE submissions SET status = 'running', started_at = NOW() WHERE id = $1::uuid`, submissionID)
	return err
}

func (store *PostgresStore) HandleJobFailure(ctx context.Context, submissionID string, leaseToken string, lastError string) (FailureAction, error) {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var attempts int
	err = tx.QueryRow(ctx, selectJobAttemptsSQL, submissionID, leaseToken).Scan(&attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrJobLeaseLost
	}
	if err != nil {
		return "", err
	}

	if attempts >= store.maxJobAttempts {
		_, err = tx.Exec(ctx, poisonSubmissionJobSQL, submissionID, lastError)
		if err != nil {
			return "", err
		}

		_, err = tx.Exec(ctx, markSubmissionJudgeFailedSQL, submissionID)
		if err != nil {
			return "", err
		}

		return FailureActionTerminal, tx.Commit(ctx)
	}

	_, err = tx.Exec(ctx, requeueSubmissionJobSQL, submissionID, time.Now().Add(store.retryDelay), lastError)
	if err != nil {
		return "", err
	}

	_, err = tx.Exec(ctx, resetSubmissionForRetrySQL, submissionID)
	if err != nil {
		return "", err
	}

	return FailureActionRetried, tx.Commit(ctx)
}

func (store *PostgresStore) CompleteSubmission(ctx context.Context, submissionID string, leaseToken string, status string, results []CaseResult) error {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var locked int
	err = tx.QueryRow(ctx, lockSubmissionJobLeaseSQL, submissionID, leaseToken).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrJobLeaseLost
	}
	if err != nil {
		return err
	}

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

	result, err := tx.Exec(ctx, `DELETE FROM submission_jobs WHERE submission_id = $1::uuid AND lease_token = $2::uuid`, submissionID, leaseToken)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrJobLeaseLost
	}

	return tx.Commit(ctx)
}
