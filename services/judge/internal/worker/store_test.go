package worker

import (
	"context"
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
		WithArgs("submission-id").
		WillReturnRows(pgxmock.NewRows([]string{"attempts"}).AddRow(2))
	mock.ExpectExec(`UPDATE submission_jobs(.|\n)*SET claimed_at = NULL, available_at = \$2, last_error = \$3`).
		WithArgs("submission-id", pgxmock.AnyArg(), "missing bundle").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE submissions(.|\n)*SET status = 'queued', started_at = NULL, finished_at = NULL, total_tests = 0, passed_tests = 0`).
		WithArgs("submission-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second)
	if err := store.HandleJobFailure(context.Background(), "submission-id", "missing bundle"); err != nil {
		t.Fatalf("HandleJobFailure() error = %v", err)
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
		WithArgs("submission-id").
		WillReturnRows(pgxmock.NewRows([]string{"attempts"}).AddRow(3))
	mock.ExpectExec(`UPDATE submission_jobs(.|\n)*SET last_error = \$2`).
		WithArgs("submission-id", "checksum mismatch").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`UPDATE submissions`).
		WithArgs("submission-id").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second)
	if err := store.HandleJobFailure(context.Background(), "submission-id", "checksum mismatch"); err != nil {
		t.Fatalf("HandleJobFailure() error = %v", err)
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
	mock.ExpectExec(`INSERT INTO submission_results`).
		WithArgs("submission-id", 0, "accepted", 25, "42\n", "").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`UPDATE submissions`).
		WithArgs("submission-id", "accepted", 1, 1).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectExec(`DELETE FROM submission_jobs WHERE submission_id = \$1::uuid`).
		WithArgs("submission-id").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	mock.ExpectCommit()

	store := NewPostgresStoreFromDatabase(mock, 3, time.Second)
	err = store.CompleteSubmission(context.Background(), "submission-id", "accepted", []CaseResult{{Verdict: "accepted", ExecutionTimeMS: 25, StdoutExcerpt: "42\n"}})
	if err != nil {
		t.Fatalf("CompleteSubmission() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
