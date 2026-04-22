package users

import (
	"context"
	"testing"

	pgxmock "github.com/pashagolub/pgxmock/v4"
)

func TestTemporaryProfileForSubjectReturnsUniqueHandle(t *testing.T) {
	first := TemporaryProfileForSubject("user_a")
	second := TemporaryProfileForSubject("user_b")

	if first.Handle == second.Handle {
		t.Fatalf("TemporaryProfileForSubject() returned duplicate handles %q", first.Handle)
	}
	if first.DisplayName == "" || second.DisplayName == "" {
		t.Fatal("TemporaryProfileForSubject() must return a non-empty display name")
	}
}

func TestGetOrCreateBySubjectCreatesOrReusesUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	profile := TemporaryProfileForSubject("clerk_user_123")
	firstRows := pgxmock.NewRows([]string{"id", "auth_subject", "handle", "display_name"}).
		AddRow("user-id-1", "clerk_user_123", profile.Handle, profile.DisplayName)
	secondRows := pgxmock.NewRows([]string{"id", "auth_subject", "handle", "display_name"}).
		AddRow("user-id-1", "clerk_user_123", profile.Handle, profile.DisplayName)
	pattern := `INSERT INTO users(.|\n)*ON CONFLICT \(auth_subject\)(.|\n)*RETURNING id::text, auth_subject, handle, display_name`
	mock.ExpectQuery(pattern).
		WithArgs("clerk_user_123", profile.Handle, profile.DisplayName).
		WillReturnRows(firstRows)
	mock.ExpectQuery(pattern).
		WithArgs("clerk_user_123", profile.Handle, profile.DisplayName).
		WillReturnRows(secondRows)

	store := NewPostgresStoreFromQuerier(mock)
	firstUser, err := store.GetOrCreateBySubject(context.Background(), "clerk_user_123")
	if err != nil {
		t.Fatalf("first GetOrCreateBySubject() error = %v", err)
	}
	secondUser, err := store.GetOrCreateBySubject(context.Background(), "clerk_user_123")
	if err != nil {
		t.Fatalf("second GetOrCreateBySubject() error = %v", err)
	}

	if firstUser.ID != "user-id-1" || firstUser.Subject != "clerk_user_123" {
		t.Fatalf("first GetOrCreateBySubject() returned unexpected user: %#v", firstUser)
	}
	if secondUser.ID != firstUser.ID || secondUser.Handle != firstUser.Handle {
		t.Fatalf("repeated GetOrCreateBySubject() should reuse the same stored user, got %#v and %#v", firstUser, secondUser)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
