package staffroles

import (
	"context"
	"errors"
	"testing"

	"github.com/lawrencefmm/cabugi/services/api/internal/users"
)

type stubStore struct {
	usersBySubject map[string]users.User
	adminCount     int
	hasAdminRole   bool
	grantUserID    string
	grantRole      users.Role
	grantErr       error
	countErr       error
	hasRoleErr     error
	userErr        error
	checkedUserID  string
}

func (store *stubStore) GetOrCreateBySubject(_ context.Context, subject string) (users.User, error) {
	if store.userErr != nil {
		return users.User{}, store.userErr
	}
	if user, ok := store.usersBySubject[subject]; ok {
		return user, nil
	}

	user := users.User{ID: subject + "-id", Subject: subject, Handle: subject + "-handle", DisplayName: subject}
	if store.usersBySubject == nil {
		store.usersBySubject = map[string]users.User{}
	}
	store.usersBySubject[subject] = user
	return user, nil
}

func (store *stubStore) HasAnyRole(_ context.Context, userID string, _ ...users.Role) (bool, error) {
	store.checkedUserID = userID
	if store.hasRoleErr != nil {
		return false, store.hasRoleErr
	}
	return store.hasAdminRole, nil
}

func (store *stubStore) CountUsersWithRole(context.Context, users.Role) (int, error) {
	if store.countErr != nil {
		return 0, store.countErr
	}
	return store.adminCount, nil
}

func (store *stubStore) GrantRole(_ context.Context, userID string, role users.Role) error {
	store.grantUserID = userID
	store.grantRole = role
	return store.grantErr
}

func TestGrantBootstrapsFirstAdminWhenNoAdminsExist(t *testing.T) {
	store := &stubStore{}
	grantedUser, err := Grant(context.Background(), store, GrantInput{BootstrapFirstAdmin: true, TargetSubject: "first-admin", Role: users.RoleAdmin})
	if err != nil {
		t.Fatalf("Grant() error = %v", err)
	}
	if grantedUser.Subject != "first-admin" {
		t.Fatalf("Grant() returned unexpected user: %#v", grantedUser)
	}
	if store.grantRole != users.RoleAdmin || store.grantUserID == "" {
		t.Fatalf("Grant() stored unexpected role assignment: %#v", store)
	}
}

func TestGrantRejectsBootstrapAfterFirstAdminExists(t *testing.T) {
	store := &stubStore{adminCount: 1}
	_, err := Grant(context.Background(), store, GrantInput{BootstrapFirstAdmin: true, TargetSubject: "another-admin", Role: users.RoleAdmin})
	if !errors.Is(err, ErrFirstAdminAlreadyBootstrapped) {
		t.Fatalf("Grant() error = %v, want %v", err, ErrFirstAdminAlreadyBootstrapped)
	}
}

func TestGrantRejectsNonAdminRequester(t *testing.T) {
	store := &stubStore{hasAdminRole: false}
	_, err := Grant(context.Background(), store, GrantInput{RequesterSubject: "regular-user", TargetSubject: "moderator-user", Role: users.RoleModerator})
	if !errors.Is(err, ErrAdminRequired) {
		t.Fatalf("Grant() error = %v, want %v", err, ErrAdminRequired)
	}
	if store.grantUserID != "" {
		t.Fatalf("Grant() should not grant a role for non-admin requester, store = %#v", store)
	}
}

func TestGrantAllowsExistingAdminToGrantModeratorRole(t *testing.T) {
	store := &stubStore{hasAdminRole: true}
	grantedUser, err := Grant(context.Background(), store, GrantInput{RequesterSubject: "admin-user", TargetSubject: "moderator-user", Role: users.RoleModerator})
	if err != nil {
		t.Fatalf("Grant() error = %v", err)
	}
	if grantedUser.Subject != "moderator-user" {
		t.Fatalf("Grant() returned unexpected user: %#v", grantedUser)
	}
	if store.grantRole != users.RoleModerator {
		t.Fatalf("Grant() granted unexpected role: %#v", store)
	}
}

func TestGrantRejectsMissingRequesterForNonBootstrapGrant(t *testing.T) {
	store := &stubStore{}
	_, err := Grant(context.Background(), store, GrantInput{TargetSubject: "moderator-user", Role: users.RoleModerator})
	if !errors.Is(err, ErrRequesterSubjectRequired) {
		t.Fatalf("Grant() error = %v, want %v", err, ErrRequesterSubjectRequired)
	}
}
