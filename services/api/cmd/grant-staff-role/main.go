package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/lawrencefmm/cabugi/services/api/internal/staffroles"
	"github.com/lawrencefmm/cabugi/services/api/internal/users"
)

func main() {
	var (
		bootstrapFirstAdmin = flag.Bool("bootstrap-first-admin", false, "grant the first admin role when no admin exists yet")
		requesterSubject    = flag.String("requester-subject", "", "authenticated subject for an existing admin performing the grant")
		targetSubject       = flag.String("target-subject", "", "authenticated subject for the user receiving the role")
		roleValue           = flag.String("role", string(users.RoleModerator), "staff role to grant: moderator or admin")
	)
	flag.Parse()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	store, err := users.NewPostgresStore(databaseURL)
	if err != nil {
		log.Fatalf("connect user store: %v", err)
	}
	defer store.Close()

	grantedUser, err := staffroles.Grant(context.Background(), store, staffroles.GrantInput{
		BootstrapFirstAdmin: *bootstrapFirstAdmin,
		RequesterSubject:    *requesterSubject,
		TargetSubject:       *targetSubject,
		Role:                users.Role(strings.TrimSpace(*roleValue)),
	})
	if err != nil {
		switch {
		case errors.Is(err, staffroles.ErrRequesterSubjectRequired),
			errors.Is(err, staffroles.ErrTargetSubjectRequired),
			errors.Is(err, staffroles.ErrInvalidRole),
			errors.Is(err, staffroles.ErrBootstrapRoleMustBeAdmin),
			errors.Is(err, staffroles.ErrAdminRequired),
			errors.Is(err, staffroles.ErrFirstAdminAlreadyBootstrapped):
			log.Fatal(err)
		default:
			log.Fatalf("grant staff role: %v", err)
		}
	}

	fmt.Printf("granted %s to %s (%s)\n", strings.TrimSpace(*roleValue), grantedUser.Subject, grantedUser.ID)
}
