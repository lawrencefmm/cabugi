package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestHandleJobFailureRequeuesSubmissionBeforeMaxAttempts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT attempts(.|\n)*FROM submission_jobs(.|\n)*FOR UPDATE`).
		WithArgs("submission-id", "lease-token").
		WillReturnRows(pgxmock.NewRows([]string{"attempts"}).AddRow(2))
	mock.ExpectExec(`UPDATE submission_jobs(.|\n)*SET claimed_at = NULL, lease_token = NULL, lease_expires_at = NULL, available_at = \$2, last_error = \$3`).
		WithArgs("submission-id", pgxmock.AnyArg(), "missing bundle").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE submissions(.|\n)*SET status = 'queued', started_at = NULL, finished_at = NULL, total_tests = 0, passed_tests = 0`).
		WithArgs("submission-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second, 30*time.Second)
	action, err := store.HandleJobFailure(context.Background(), "submission-id", "lease-token", "missing bundle")
	if err != nil {
		t.Fatalf("HandleJobFailure() error = %v", err)
	}
	if action != FailureActionRetried {
		t.Fatalf("HandleJobFailure() action = %q, want %q", action, FailureActionRetried)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestHandleJobFailureMarksSubmissionJudgeFailedAfterMaxAttempts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT attempts(.|\n)*FROM submission_jobs(.|\n)*FOR UPDATE`).
		WithArgs("submission-id", "lease-token").
		WillReturnRows(pgxmock.NewRows([]string{"attempts"}).AddRow(3))
	mock.ExpectExec(`UPDATE submission_jobs(.|\n)*last_error = \$2`).
		WithArgs("submission-id", "checksum mismatch").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE submissions`).
		WithArgs("submission-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second, 30*time.Second)
	action, err := store.HandleJobFailure(context.Background(), "submission-id", "lease-token", "checksum mismatch")
	if err != nil {
		t.Fatalf("HandleJobFailure() error = %v", err)
	}
	if action != FailureActionTerminal {
		t.Fatalf("HandleJobFailure() action = %q, want %q", action, FailureActionTerminal)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestCompleteSubmissionDeletesJobOnSuccess(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT 1(.|\n)*FROM submission_jobs(.|\n)*FOR UPDATE`).
		WithArgs("submission-id", "lease-token").
		WillReturnRows(pgxmock.NewRows([]string{"one"}).AddRow(1))
	mock.ExpectExec(`INSERT INTO submission_results`).
		WithArgs("submission-id", 0, "accepted", 25, "42\n", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`UPDATE submissions`).
		WithArgs("submission-id", "accepted", 1, 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`DELETE FROM submission_jobs WHERE submission_id = \$1::uuid AND lease_token = \$2::uuid`).
		WithArgs("submission-id", "lease-token").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second, 30*time.Second)
	err = store.CompleteSubmission(context.Background(), "submission-id", "lease-token", "accepted", []CaseResult{{Verdict: "accepted", ExecutionTimeMS: 25, StdoutExcerpt: "42\n"}})
	if err != nil {
		t.Fatalf("CompleteSubmission() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestRenewJobLeaseReturnsLeaseLostWhenClaimIsGone(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE submission_jobs(.|\n)*SET lease_expires_at = \$3`).
		WithArgs("submission-id", "lease-token", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second, 30*time.Second)
	err = store.RenewJobLease(context.Background(), "submission-id", "lease-token")
	if !errors.Is(err, ErrJobLeaseLost) {
		t.Fatalf("RenewJobLease() error = %v, want ErrJobLeaseLost", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
