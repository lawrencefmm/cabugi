package submissions

import (
	"context"
	"testing"
	"time"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestCreateSubmissionRejectsInvalidLanguage(t *testing.T) {
	store := DisabledStore{}
	_, err := store.CreateSubmission(context.Background(), CreateInput{Language: "java"})
	if err != ErrStoreNotConfigured {
		t.Fatalf("DisabledStore.CreateSubmission() error = %v, want %v", err, ErrStoreNotConfigured)
	}

	postgresStore := NewPostgresStoreFromQuerier(nil)
	_, err = postgresStore.CreateSubmission(context.Background(), CreateInput{Language: "java"})
	if err != ErrInvalidLanguage {
		t.Fatalf("CreateSubmission() error = %v, want %v", err, ErrInvalidLanguage)
	}
}

func TestCreateSubmissionCreatesSubmissionAndJobForPublishedProblem(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	queuedAt := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at"}).
		AddRow("submission-id", "two-sum", "cpp17", "queued", queuedAt)
	mock.ExpectQuery(`WITH published_problem AS \((.|\n)*WHERE pv.lifecycle_status = 'published' AND p.slug = \$2(.|\n)*INSERT INTO submission_jobs`).
		WithArgs("00000000-0000-0000-0000-000000000001", "two-sum", "cpp17", "int main() {}\n").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	submission, err := store.CreateSubmission(context.Background(), CreateInput{
		UserID:      "00000000-0000-0000-0000-000000000001",
		ProblemSlug: "two-sum",
		Language:    "cpp17",
		SourceCode:  "int main() {}\n",
	})
	if err != nil {
		t.Fatalf("CreateSubmission() error = %v", err)
	}

	if submission.ID != "submission-id" || submission.Status != "queued" {
		t.Fatalf("CreateSubmission() returned unexpected submission: %#v", submission)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestCreateSubmissionReturnsProblemNotFoundForDraftOnlyOrMissingProblem(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`WITH published_problem AS \((.|\n)*WHERE pv.lifecycle_status = 'published' AND p.slug = \$2(.|\n)*INSERT INTO submission_jobs`).
		WithArgs("00000000-0000-0000-0000-000000000001", "draft-only", "python", "print(42)\n").
		WillReturnRows(pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at"}))

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.CreateSubmission(context.Background(), CreateInput{
		UserID:      "00000000-0000-0000-0000-000000000001",
		ProblemSlug: "draft-only",
		Language:    "python",
		SourceCode:  "print(42)\n",
	})
	if err != ErrProblemNotFound {
		t.Fatalf("CreateSubmission() error = %v, want %v", err, ErrProblemNotFound)
	}
}

func TestGetSubmissionByIDReturnsSubmissionForOwner(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	queuedAt := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at", "total_tests", "passed_tests"}).
		AddRow("submission-id", "two-sum", "cpp17", "wrong_answer", queuedAt, 3, 2)
	mock.ExpectQuery(`SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at, s.total_tests, s.passed_tests(.|\n)*WHERE s.id = \$1::uuid AND s.user_id = \$2::uuid`).
		WithArgs("submission-id", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	resultRows := pgxmock.NewRows([]string{"test_index", "verdict", "execution_time_ms", "memory_bytes", "stdout_excerpt", "stderr_excerpt"}).
		AddRow(0, "accepted", 11, int64(0), "42\n", "").
		AddRow(1, "accepted", 14, int64(0), "42\n", "").
		AddRow(2, "wrong_answer", 13, int64(0), "41\n", "")
	mock.ExpectQuery(`SELECT test_index, verdict::text, execution_time_ms, memory_bytes, stdout_excerpt, stderr_excerpt(.|\n)*WHERE submission_id = \$1::uuid(.|\n)*ORDER BY test_index ASC`).
		WithArgs("submission-id").
		WillReturnRows(resultRows)

	store := NewPostgresStoreFromQuerier(mock)
	submission, err := store.GetSubmissionByID(context.Background(), "submission-id", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("GetSubmissionByID() error = %v", err)
	}
	if submission.ProblemSlug != "two-sum" {
		t.Fatalf("GetSubmissionByID() returned unexpected submission: %#v", submission)
	}
	if submission.TotalTests != 3 || submission.PassedTests != 2 {
		t.Fatalf("GetSubmissionByID() returned unexpected aggregate counts: %#v", submission)
	}
	if len(submission.Results) != 3 {
		t.Fatalf("GetSubmissionByID() returned %d results, want 3", len(submission.Results))
	}
	if submission.Results[2].TestIndex != 2 || submission.Results[2].Verdict != "wrong_answer" {
		t.Fatalf("GetSubmissionByID() returned unexpected results: %#v", submission.Results)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGetSubmissionByIDReturnsNotFoundForUnknownID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at, s.total_tests, s.passed_tests(.|\n)*WHERE s.id = \$1::uuid AND s.user_id = \$2::uuid`).
		WithArgs("missing-id", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at", "total_tests", "passed_tests"}))

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.GetSubmissionByID(context.Background(), "missing-id", "00000000-0000-0000-0000-000000000001")
	if err != ErrSubmissionNotFound {
		t.Fatalf("GetSubmissionByID() error = %v, want %v", err, ErrSubmissionNotFound)
	}
}

func TestListSubmissionsReturnsOwnerHistoryNewestFirst(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	newest := time.Now().UTC()
	older := newest.Add(-time.Hour)
	rows := pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at"}).
		AddRow("submission-2", "two-sum", "python", "accepted", newest).
		AddRow("submission-1", "a-plus-b", "cpp17", "wrong_answer", older)
	mock.ExpectQuery(`SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at(.|\n)*WHERE s.user_id = \$1::uuid(.|\n)*ORDER BY s.queued_at DESC, s.id DESC`).
		WithArgs("00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	submissions, err := store.ListSubmissions(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("ListSubmissions() error = %v", err)
	}
	if len(submissions) != 2 {
		t.Fatalf("ListSubmissions() returned %d rows, want 2", len(submissions))
	}
	if submissions[0].ID != "submission-2" || submissions[1].ID != "submission-1" {
		t.Fatalf("ListSubmissions() returned unexpected ordering: %#v", submissions)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestQueueDepthCountsActiveSubmissionJobs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\)(.|\n)*FROM submission_jobs(.|\n)*WHERE available_at < 'infinity'::timestamptz`).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(4))

	store := NewPostgresStoreFromQuerier(mock)
	depth, err := store.QueueDepth(context.Background())
	if err != nil {
		t.Fatalf("QueueDepth() error = %v", err)
	}
	if depth != 4 {
		t.Fatalf("QueueDepth() = %d, want 4", depth)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
