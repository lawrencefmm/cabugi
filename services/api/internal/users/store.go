package users

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrStoreNotConfigured = errors.New("user store not configured")

type Role string

const (
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

type User struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

type Store interface {
	GetOrCreateBySubject(context.Context, string) (User, error)
	HasAnyRole(context.Context, string, ...Role) (bool, error)
}

type DisabledStore struct{}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type PostgresStore struct {
	pool *pgxpool.Pool
	db   queryer
}

const getOrCreateUserBySubjectSQL = `
INSERT INTO users (auth_subject, handle, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (auth_subject)
DO UPDATE SET updated_at = NOW()
RETURNING id::text, auth_subject, handle, display_name
`

const hasAnyRoleSQL = `
SELECT EXISTS(
  SELECT 1
  FROM user_roles
  WHERE user_id = $1::uuid AND role::text = ANY($2::text[])
)
`

func NewPostgresStore(databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStore{pool: pool, db: pool}, nil
}

func NewPostgresStoreFromQuerier(db queryer) *PostgresStore {
	return &PostgresStore{db: db}
}

func (store *PostgresStore) GetOrCreateBySubject(ctx context.Context, subject string) (User, error) {
	profile := TemporaryProfileForSubject(subject)

	var user User
	err := store.db.QueryRow(ctx, getOrCreateUserBySubjectSQL, subject, profile.Handle, profile.DisplayName).Scan(
		&user.ID,
		&user.Subject,
		&user.Handle,
		&user.DisplayName,
	)
	return user, err
}

func (store *PostgresStore) HasAnyRole(ctx context.Context, userID string, roles ...Role) (bool, error) {
	roleValues := make([]string, 0, len(roles))
	for _, role := range roles {
		roleValues = append(roleValues, string(role))
	}

	var hasRole bool
	err := store.db.QueryRow(ctx, hasAnyRoleSQL, userID, roleValues).Scan(&hasRole)
	return hasRole, err
}

func (store *PostgresStore) Close() {
	if store.pool != nil {
		store.pool.Close()
	}
}

func (DisabledStore) GetOrCreateBySubject(context.Context, string) (User, error) {
	return User{}, ErrStoreNotConfigured
}

func (DisabledStore) HasAnyRole(context.Context, string, ...Role) (bool, error) {
	return false, ErrStoreNotConfigured
}

func TemporaryProfileForSubject(subject string) User {
	hash := sha256.Sum256([]byte(subject))
	suffix := hex.EncodeToString(hash[:])
	handleSuffix := suffix[:12]
	displaySuffix := suffix[:6]

	return User{
		Subject:     subject,
		Handle:      fmt.Sprintf("user_%s", handleSuffix),
		DisplayName: fmt.Sprintf("User %s", displaySuffix),
	}
}
