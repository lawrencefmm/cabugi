package staffroles

import (
	"context"
	"errors"
	"strings"

	"github.com/lawrencefmm/cabugi/services/api/internal/users"
)

var (
	ErrRequesterSubjectRequired      = errors.New("requester subject is required")
	ErrTargetSubjectRequired         = errors.New("target subject is required")
	ErrInvalidRole                   = errors.New("invalid staff role")
	ErrAdminRequired                 = errors.New("admin role required")
	ErrBootstrapRoleMustBeAdmin      = errors.New("bootstrap role must be admin")
	ErrFirstAdminAlreadyBootstrapped = errors.New("first admin already exists")
)

type Store interface {
	GetOrCreateBySubject(context.Context, string) (users.User, error)
	HasAnyRole(context.Context, string, ...users.Role) (bool, error)
	CountUsersWithRole(context.Context, users.Role) (int, error)
	GrantRole(context.Context, string, users.Role) error
}

type GrantInput struct {
	BootstrapFirstAdmin bool
	RequesterSubject    string
	TargetSubject       string
	Role                users.Role
}

func Grant(ctx context.Context, store Store, input GrantInput) (users.User, error) {
	targetSubject := strings.TrimSpace(input.TargetSubject)
	if targetSubject == "" {
		return users.User{}, ErrTargetSubjectRequired
	}
	if input.Role != users.RoleModerator && input.Role != users.RoleAdmin {
		return users.User{}, ErrInvalidRole
	}

	targetUser, err := store.GetOrCreateBySubject(ctx, targetSubject)
	if err != nil {
		return users.User{}, err
	}

	if input.BootstrapFirstAdmin {
		if input.Role != users.RoleAdmin {
			return users.User{}, ErrBootstrapRoleMustBeAdmin
		}

		adminCount, err := store.CountUsersWithRole(ctx, users.RoleAdmin)
		if err != nil {
			return users.User{}, err
		}
		if adminCount > 0 {
			return users.User{}, ErrFirstAdminAlreadyBootstrapped
		}

		return targetUser, store.GrantRole(ctx, targetUser.ID, users.RoleAdmin)
	}

	requesterSubject := strings.TrimSpace(input.RequesterSubject)
	if requesterSubject == "" {
		return users.User{}, ErrRequesterSubjectRequired
	}

	requesterUser, err := store.GetOrCreateBySubject(ctx, requesterSubject)
	if err != nil {
		return users.User{}, err
	}

	hasAdminRole, err := store.HasAnyRole(ctx, requesterUser.ID, users.RoleAdmin)
	if err != nil {
		return users.User{}, err
	}
	if !hasAdminRole {
		return users.User{}, ErrAdminRequired
	}

	return targetUser, store.GrantRole(ctx, targetUser.ID, input.Role)
}
