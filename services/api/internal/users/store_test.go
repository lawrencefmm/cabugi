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

func TestHasAnyRoleReturnsTrueWhenUserHasStaffRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS\((.|\n)*FROM user_roles(.|\n)*role::text = ANY\(\$2::text\[\]\)(.|\n)*\)`).
		WithArgs("user-id-1", []string{"moderator", "admin"}).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(true))

	store := NewPostgresStoreFromQuerier(mock)
	hasRole, err := store.HasAnyRole(context.Background(), "user-id-1", RoleModerator, RoleAdmin)
	if err != nil {
		t.Fatalf("HasAnyRole() error = %v", err)
	}
	if !hasRole {
		t.Fatal("HasAnyRole() = false, want true")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestHasAnyRoleReturnsFalseWhenUserHasNoRequestedRoles(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS\((.|\n)*FROM user_roles(.|\n)*role::text = ANY\(\$2::text\[\]\)(.|\n)*\)`).
		WithArgs("user-id-2", []string{"moderator", "admin"}).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))

	store := NewPostgresStoreFromQuerier(mock)
	hasRole, err := store.HasAnyRole(context.Background(), "user-id-2", RoleModerator, RoleAdmin)
	if err != nil {
		t.Fatalf("HasAnyRole() error = %v", err)
	}
	if hasRole {
		t.Fatal("HasAnyRole() = true, want false")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestCountUsersWithRoleReturnsAdminCount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT COUNT\(\*\)(.|\n)*FROM user_roles(.|\n)*role = \$1::user_role`).
		WithArgs(RoleAdmin).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	store := NewPostgresStoreFromQuerier(mock)
	count, err := store.CountUsersWithRole(context.Background(), RoleAdmin)
	if err != nil {
		t.Fatalf("CountUsersWithRole() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountUsersWithRole() = %d, want 2", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestGrantRoleInsertsModeratorRoleIdempotently(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool() error = %v", err)
	}
	defer mock.Close()

	mock.ExpectExec(`INSERT INTO user_roles(.|\n)*ON CONFLICT \(user_id, role\) DO NOTHING`).
		WithArgs("user-id-3", RoleModerator).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	store := NewPostgresStoreFromQuerier(mock)
	err = store.GrantRole(context.Background(), "user-id-3", RoleModerator)
	if err != nil {
		t.Fatalf("GrantRole() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
