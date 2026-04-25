package problems

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestListPublishedProblemsUsesPublishedFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "title", "time_limit_ms", "memory_limit_mb"}).
		AddRow("two-sum", "Two Sum", 1000, 256).
		AddRow("a-plus-b", "A + B", 1000, 256)
	mock.ExpectQuery("SELECT(.|\n)*WHERE pv.lifecycle_status = 'published'(.|\n)*ORDER BY p.slug ASC").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	problems, err := store.ListPublishedProblems(context.Background())
	if err != nil {
		t.Fatalf("ListPublishedProblems() error = %v", err)
	}

	if len(problems) != 2 {
		t.Fatalf("ListPublishedProblems() length = %d, want 2", len(problems))
	}
	if problems[0].Slug != "two-sum" || problems[1].Slug != "a-plus-b" {
		t.Fatalf("ListPublishedProblems() returned unexpected slugs: %#v", problems)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGetPublishedProblemBySlugUsesPublishedFilter(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb"}).
		AddRow("two-sum", "Two Sum", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256)
	mock.ExpectQuery("SELECT(.|\n)*WHERE pv.lifecycle_status = 'published' AND p.slug = \\$1(.|\n)*").
		WithArgs("two-sum").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.GetPublishedProblemBySlug(context.Background(), "two-sum")
	if err != nil {
		t.Fatalf("GetPublishedProblemBySlug() error = %v", err)
	}

	if problem.Slug != "two-sum" || !strings.Contains(problem.StatementMarkdown, "Solve") {
		t.Fatalf("GetPublishedProblemBySlug() returned unexpected problem: %#v", problem)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGetPublishedProblemBySlugReturnsNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT(.|\n)*WHERE pv.lifecycle_status = 'published' AND p.slug = \\$1(.|\n)*").
		WithArgs("missing-problem").
		WillReturnRows(pgxmock.NewRows([]string{"slug", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb"}))

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.GetPublishedProblemBySlug(context.Background(), "missing-problem")
	if err != ErrNotFound {
		t.Fatalf("GetPublishedProblemBySlug() error = %v, want %v", err, ErrNotFound)
	}
}

func TestCreateDraftCreatesProblemAndInitialDraftVersion(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "", "")
	mock.ExpectQuery(`WITH inserted_problem AS \((.|\n)*INSERT INTO problem_versions(.|\n)*RETURNING problem_id, version_number, lifecycle_status::text`).
		WithArgs("two-sum-user", "00000000-0000-0000-0000-000000000001", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "", "").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.CreateDraft(context.Background(), CreateDraftInput{
		UserID:              "00000000-0000-0000-0000-000000000001",
		Slug:                "two-sum-user",
		Title:               "Two Sum User",
		StatementMarkdown:   "Solve it",
		InputMarkdown:       "Input",
		OutputMarkdown:      "Output",
		ConstraintsMarkdown: "Constraints",
		NotesMarkdown:       "Notes",
		TimeLimitMs:         1000,
		MemoryLimitMB:       256,
	})
	if err != nil {
		t.Fatalf("CreateDraft() error = %v", err)
	}
	if problem.Slug != "two-sum-user" || problem.VersionNumber != 1 || problem.LifecycleStatus != "draft" {
		t.Fatalf("CreateDraft() returned unexpected problem: %#v", problem)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestCreateDraftReturnsProblemSlugTakenOnUniqueViolation(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`WITH inserted_problem AS \((.|\n)*INSERT INTO problem_versions`).
		WithArgs("two-sum", "00000000-0000-0000-0000-000000000001", "Two Sum", "", "", "", "", "", 1000, 256, "", "").
		WillReturnError(&pgconn.PgError{Code: "23505", ConstraintName: "problems_slug_key"})

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.CreateDraft(context.Background(), CreateDraftInput{
		UserID:        "00000000-0000-0000-0000-000000000001",
		Slug:          "two-sum",
		Title:         "Two Sum",
		TimeLimitMs:   1000,
		MemoryLimitMB: 256,
	})
	if err != ErrProblemSlugTaken {
		t.Fatalf("CreateDraft() error = %v, want %v", err, ErrProblemSlugTaken)
	}
}

func TestListDraftsByOwnerReturnsLatestUpdatedFirst(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	newer := time.Now().UTC()
	submitted := newer.Add(-30 * time.Minute)
	older := newer.Add(-time.Hour)
	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "updated_at", "submitted_for_review_at"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", newer, nil).
		AddRow("a-plus-b-user", 2, "in_review", "A + B User", older, submitted)
	mock.ExpectQuery(`SELECT(.|\n)*WHERE pv.created_by_user_id = \$1::uuid AND pv.lifecycle_status IN \('draft', 'in_review'\)(.|\n)*ORDER BY pv.updated_at DESC, p.slug ASC`).
		WithArgs("00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	items, err := store.ListDraftsByOwner(context.Background(), "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("ListDraftsByOwner() error = %v", err)
	}
	if len(items) != 2 || items[0].Slug != "two-sum-user" || items[1].Slug != "a-plus-b-user" {
		t.Fatalf("ListDraftsByOwner() returned unexpected items: %#v", items)
	}
	if items[0].SubmittedForReviewAt != nil {
		t.Fatalf("ListDraftsByOwner() draft item should not have submitted timestamp: %#v", items[0])
	}
	if items[1].SubmittedForReviewAt == nil || !items[1].SubmittedForReviewAt.Equal(submitted) {
		t.Fatalf("ListDraftsByOwner() in-review item missing submitted timestamp: %#v", items[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGetDraftBySlugReturnsOwnerDraft(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "", "")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status IN \('draft', 'in_review'\) AND \(\$2 OR pv.created_by_user_id = \$3::uuid\)(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.GetDraftBySlug(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000001", false)
	if err != nil {
		t.Fatalf("GetDraftBySlug() error = %v", err)
	}
	if problem.Title != "Two Sum User" {
		t.Fatalf("GetDraftBySlug() returned unexpected problem: %#v", problem)
	}
}

func TestUpdateDraftReturnsUpdatedDraftForOwner(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Updated Title", "Solve it better", "Input", "Output", "Constraints", "Notes", 1500, 512, "", "")
	mock.ExpectQuery(`WITH target_version AS \((.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status = 'draft' AND \(\$2 OR pv.created_by_user_id = \$3::uuid\)(.|\n)*UPDATE problem_versions pv`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000001", "Updated Title", "Solve it better", "Input", "Output", "Constraints", "Notes", 1500, 512, "", "").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.UpdateDraft(context.Background(), UpdateDraftInput{
		ActorUserID:         "00000000-0000-0000-0000-000000000001",
		Slug:                "two-sum-user",
		Title:               "Updated Title",
		StatementMarkdown:   "Solve it better",
		InputMarkdown:       "Input",
		OutputMarkdown:      "Output",
		ConstraintsMarkdown: "Constraints",
		NotesMarkdown:       "Notes",
		TimeLimitMs:         1500,
		MemoryLimitMB:       512,
	})
	if err != nil {
		t.Fatalf("UpdateDraft() error = %v", err)
	}
	if problem.Title != "Updated Title" || problem.TimeLimitMs != 1500 {
		t.Fatalf("UpdateDraft() returned unexpected problem: %#v", problem)
	}
}

func TestUpdateDraftReturnsNotFoundForUnownedDraft(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`WITH target_version AS \((.|\n)*UPDATE problem_versions pv`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000002", "Updated Title", "", "", "", "", "", 1000, 256, "", "").
		WillReturnRows(pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}))

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.UpdateDraft(context.Background(), UpdateDraftInput{
		ActorUserID:   "00000000-0000-0000-0000-000000000002",
		Slug:          "two-sum-user",
		Title:         "Updated Title",
		TimeLimitMs:   1000,
		MemoryLimitMB: 256,
	})
	if err != ErrDraftNotFound {
		t.Fatalf("UpdateDraft() error = %v, want %v", err, ErrDraftNotFound)
	}
}

func TestSubmitDraftForReviewTransitionsOwnedDraftToInReview(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	currentRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status IN \('draft', 'in_review'\) AND \(\$2 OR pv.created_by_user_id = \$3::uuid\)(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000001").
		WillReturnRows(currentRows)

	updatedRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "in_review", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`WITH target_version AS \((.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status = 'draft' AND pv.created_by_user_id = \$2::uuid(.|\n)*UPDATE problem_versions pv`).
		WithArgs("two-sum-user", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(updatedRows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.SubmitDraftForReview(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("SubmitDraftForReview() error = %v", err)
	}
	if problem.LifecycleStatus != "in_review" {
		t.Fatalf("SubmitDraftForReview() returned unexpected lifecycle status: %#v", problem)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestSubmitDraftForReviewRejectsInvalidLifecycleTransition(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "in_review", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status IN \('draft', 'in_review'\) AND \(\$2 OR pv.created_by_user_id = \$3::uuid\)(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.SubmitDraftForReview(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000001")
	if err != ErrInvalidLifecycleTransition {
		t.Fatalf("SubmitDraftForReview() error = %v, want %v", err, ErrInvalidLifecycleTransition)
	}
}

func TestSubmitDraftForReviewRejectsDraftMissingBundleMetadata(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "", "")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status IN \('draft', 'in_review'\) AND \(\$2 OR pv.created_by_user_id = \$3::uuid\)(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user", false, "00000000-0000-0000-0000-000000000001").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.SubmitDraftForReview(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000001")
	if err != ErrDraftNotReadyForReview {
		t.Fatalf("SubmitDraftForReview() error = %v, want %v", err, ErrDraftNotReadyForReview)
	}
}

func TestListDraftsInReviewReturnsOldestFirst(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	older := time.Now().Add(-time.Hour).UTC()
	newer := time.Now().UTC()
	rows := pgxmock.NewRows([]string{"slug", "version_number", "title", "submitted_for_review_at"}).
		AddRow("a-plus-b-user", 1, "A + B User", older).
		AddRow("two-sum-user", 2, "Two Sum User", newer)
	mock.ExpectQuery(`SELECT(.|\n)*WHERE pv.lifecycle_status = 'in_review'(.|\n)*ORDER BY pv.submitted_for_review_at ASC, p.slug ASC`).
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	items, err := store.ListDraftsInReview(context.Background())
	if err != nil {
		t.Fatalf("ListDraftsInReview() error = %v", err)
	}
	if len(items) != 2 || items[0].Slug != "a-plus-b-user" || items[1].Slug != "two-sum-user" {
		t.Fatalf("ListDraftsInReview() returned unexpected items: %#v", items)
	}
}

func TestApplyModerationDecisionApprovesInReviewDraft(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	currentRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "in_review", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user").
		WillReturnRows(currentRows)

	updatedRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "published", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`WITH target_version AS \((.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status = 'in_review'(.|\n)*UPDATE problem_versions pv(.|\n)*lifecycle_status = 'published'`).
		WithArgs("two-sum-user", "00000000-0000-0000-0000-000000000099", "looks good").
		WillReturnRows(updatedRows)

	store := NewPostgresStoreFromQuerier(mock)
	problem, err := store.ApplyModerationDecision(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000099", DecisionApprove, "looks good")
	if err != nil {
		t.Fatalf("ApplyModerationDecision() error = %v", err)
	}
	if problem.LifecycleStatus != "published" {
		t.Fatalf("ApplyModerationDecision() returned unexpected status: %#v", problem)
	}
}

func TestApplyModerationDecisionRejectsInvalidLifecycleTransition(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	rows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
		AddRow("two-sum-user", 1, "draft", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1(.|\n)*ORDER BY pv.version_number DESC`).
		WithArgs("two-sum-user").
		WillReturnRows(rows)

	store := NewPostgresStoreFromQuerier(mock)
	_, err = store.ApplyModerationDecision(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000099", DecisionApprove, "looks good")
	if err != ErrInvalidLifecycleTransition {
		t.Fatalf("ApplyModerationDecision() error = %v, want %v", err, ErrInvalidLifecycleTransition)
	}
}

func TestApplyModerationDecisionSupportsRejectAndRequestChanges(t *testing.T) {
	for _, tc := range []struct {
		name       string
		decision   ModerationDecision
		wantStatus string
		pattern    string
	}{
		{name: "reject", decision: DecisionReject, wantStatus: "archived", pattern: `lifecycle_status = 'archived'`},
		{name: "request changes", decision: DecisionRequestChanges, wantStatus: "draft", pattern: `lifecycle_status = 'draft'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatalf("pgxmock.NewPool() error = %v", err)
			}
			defer mock.Close()

			currentRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
				AddRow("two-sum-user", 1, "in_review", "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
			mock.ExpectQuery(`SELECT(.|\n)*WHERE p.slug = \$1(.|\n)*ORDER BY pv.version_number DESC`).
				WithArgs("two-sum-user").
				WillReturnRows(currentRows)

			updatedRows := pgxmock.NewRows([]string{"slug", "version_number", "lifecycle_status", "title", "statement_markdown", "input_markdown", "output_markdown", "constraints_markdown", "notes_markdown", "time_limit_ms", "memory_limit_mb", "hidden_test_bundle_key", "hidden_test_bundle_sha256"}).
				AddRow("two-sum-user", 1, tc.wantStatus, "Two Sum User", "Solve it", "Input", "Output", "Constraints", "Notes", 1000, 256, "bundles/two-sum.json", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
			mock.ExpectQuery(`WITH target_version AS \((.|\n)*WHERE p.slug = \$1 AND pv.lifecycle_status = 'in_review'(.|\n)*UPDATE problem_versions pv(.|\n)*`+tc.pattern).
				WithArgs("two-sum-user", "00000000-0000-0000-0000-000000000099", "needs work").
				WillReturnRows(updatedRows)

			store := NewPostgresStoreFromQuerier(mock)
			problem, err := store.ApplyModerationDecision(context.Background(), "two-sum-user", "00000000-0000-0000-0000-000000000099", tc.decision, "needs work")
			if err != nil {
				t.Fatalf("ApplyModerationDecision() error = %v", err)
			}
			if problem.LifecycleStatus != tc.wantStatus {
				t.Fatalf("ApplyModerationDecision() returned unexpected status: %#v", problem)
			}
		})
	}
}
