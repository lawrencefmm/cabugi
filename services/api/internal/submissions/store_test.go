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
	rows := pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at"}).
		AddRow("submission-id", "two-sum", "cpp17", "queued", queuedAt)
	mock.ExpectQuery(`SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at(.|\n)*WHERE s.id = \$1::uuid AND s.user_id = \$2::uuid`).
		WithArgs("submission-id", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	submission, err := store.GetSubmissionByID(context.Background(), "submission-id", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("GetSubmissionByID() error = %v", err)
	}
	if submission.ProblemSlug != "two-sum" {
		t.Fatalf("GetSubmissionByID() returned unexpected submission: %#v", submission)
	}
}

func TestGetSubmissionByIDReturnsNotFoundForUnknownID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT s.id::text, p.slug, s.language::text, s.status::text, s.queued_at(.|\n)*WHERE s.id = \$1::uuid AND s.user_id = \$2::uuid`).
		WithArgs("missing-id", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(pgxmock.NewRows([]string{"id", "slug", "language", "status", "queued_at"}))

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
