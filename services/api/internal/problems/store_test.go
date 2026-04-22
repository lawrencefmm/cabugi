package problems

import (
	"context"
	"strings"
	"testing"

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
