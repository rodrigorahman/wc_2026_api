// Package repository provides the concrete data-access layer for the auth
// domain. Each type in this package is a thin wrapper around sqlc-generated
// queries — no business logic lives here.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
)

// ErrUserNotFound is returned by GetUserByEmail when no row matches the given
// e-mail address. Callers use errors.Is to distinguish "not found" from other
// database errors.
var ErrUserNotFound = errors.New("user not found")

// User is the domain representation of a registered user. Field names follow
// Go (English) conventions; the mapping to/from pt-BR database columns is
// performed by sqlc-generated code.
type User struct {
	ID                    string
	FullName              string
	Email                 string
	PasswordHash          string
	TempPasswordHash      string
	TempPasswordExpiresAt time.Time
	NationalTeamIDs       []string
	CreatedAt             time.Time
}

// UserRepository persists and retrieves users using sqlc-generated queries.
// It is a concrete type; the interface it satisfies is declared in the
// consuming package (internal/auth/service) following Go placement idioms.
type UserRepository struct {
	db *sql.DB
	q  *sqlc.Queries
}

// NewUserRepository constructs a UserRepository backed by the given sqlc
// Queries handle. The *sql.DB is required so CreateUser can persist the user
// row and its national-team rows inside a single transaction.
func NewUserRepository(db *sql.DB, q *sqlc.Queries) *UserRepository {
	return &UserRepository{db: db, q: q}
}

// CreateUser inserts a new user row together with one user_national_teams row per
// chosen national team, atomically: any failure rolls back the whole insertion.
// The caller is responsible for providing a pre-hashed password (password_hash) and
// a unique UUID (id). Database-level UNIQUE and FK constraints are the final
// safety net — errors from those constraints propagate as-is to the caller.
func (r *UserRepository) CreateUser(ctx context.Context, u User) (User, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("create user: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	qtx := r.q.WithTx(tx)

	row, err := qtx.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           u.ID,
		FullName:     u.FullName,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
	})
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	for _, teamID := range u.NationalTeamIDs {
		if err := qtx.AddUserNationalTeam(ctx, sqlc.AddUserNationalTeamParams{
			UserID:         u.ID,
			NationalTeamID: teamID,
		}); err != nil {
			return User{}, fmt.Errorf("create user: add national team: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("create user: commit: %w", err)
	}

	created := toUser(row)
	created.NationalTeamIDs = u.NationalTeamIDs
	return created, nil
}

// GetUserByEmail returns the user whose e-mail matches the given address.
// Returns ErrUserNotFound (wrapped) when no row exists so callers can branch
// with errors.Is(err, ErrUserNotFound).
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, fmt.Errorf("get user by email: %w", ErrUserNotFound)
		}
		return User{}, fmt.Errorf("get user by email: %w", err)
	}

	return r.withNationalTeams(ctx, toUser(row))
}

// GetUserByID returns the user whose id matches the given value. Returns
// ErrUserNotFound (wrapped) when no row exists so callers can branch with
// errors.Is(err, ErrUserNotFound).
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, fmt.Errorf("get user by id: %w", ErrUserNotFound)
		}
		return User{}, fmt.Errorf("get user by id: %w", err)
	}

	return r.withNationalTeams(ctx, toUser(row))
}

// SetTempPassword stores a temporary-password hash and its expiration for the
// user with the given id. Returns ErrUserNotFound (wrapped) when no row is
// updated so callers can branch with errors.Is(err, ErrUserNotFound).
func (r *UserRepository) SetTempPassword(ctx context.Context, id string, hash string, expiresAt time.Time) error {
	rows, err := r.q.SetTempPassword(ctx, sqlc.SetTempPasswordParams{
		TempPasswordHash:      sql.NullString{String: hash, Valid: true},
		TempPasswordExpiresAt: sql.NullTime{Time: expiresAt, Valid: true},
		ID:                    id,
	})
	if err != nil {
		return fmt.Errorf("set temp password: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("set temp password: %w", ErrUserNotFound)
	}
	return nil
}

// UpdatePassword replaces the permanent password hash for the user with the
// given id and clears any pending temporary password atomically. Returns
// ErrUserNotFound (wrapped) when no row is updated.
func (r *UserRepository) UpdatePassword(ctx context.Context, id string, hash string) error {
	rows, err := r.q.UpdatePassword(ctx, sqlc.UpdatePasswordParams{
		PasswordHash: hash,
		ID:           id,
	})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("update password: %w", ErrUserNotFound)
	}
	return nil
}

// withNationalTeams populates u.NationalTeamIDs from the user_national_teams table.
func (r *UserRepository) withNationalTeams(ctx context.Context, u User) (User, error) {
	teamIDs, err := r.q.ListUserNationalTeams(ctx, u.ID)
	if err != nil {
		return User{}, fmt.Errorf("list user national teams: %w", err)
	}
	u.NationalTeamIDs = teamIDs
	return u, nil
}

// toUser converts a sqlc-generated User row to the domain User type. The
// national-team list is populated separately via ListUserNationalTeams.
func toUser(row sqlc.User) User {
	return User{
		ID:                    row.ID,
		FullName:              row.FullName,
		Email:                 row.Email,
		PasswordHash:          row.PasswordHash,
		TempPasswordHash:      row.TempPasswordHash.String,
		TempPasswordExpiresAt: row.TempPasswordExpiresAt.Time,
		CreatedAt:             row.CreatedAt,
	}
}
