// Package repository provides the concrete data-access layer for the
// nationalteam domain. NationalTeamRepository is a thin wrapper around
// sqlc-generated queries — no business logic lives here.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
)

// ErrNationalTeamNotFound is returned by GetNationalTeamByID when no row
// matches the given id. Callers use errors.Is to distinguish "not found"
// from other database errors.
var ErrNationalTeamNotFound = errors.New("national team not found")

// NationalTeamRepository retrieves national teams using sqlc-generated
// queries. It is a concrete type; the interface it satisfies is declared in
// the consuming packages (internal/auth/service,
// internal/nationalteam/service) following Go interface-placement idioms.
type NationalTeamRepository struct {
	q *sqlc.Queries
}

// NewNationalTeamRepository constructs a NationalTeamRepository backed by
// the given sqlc Queries handle.
func NewNationalTeamRepository(q *sqlc.Queries) *NationalTeamRepository {
	return &NationalTeamRepository{q: q}
}

// NationalTeam is the domain representation of a national team. Field names
// follow Go (English) conventions, matching the national_teams table columns
// (name, flag_url) via sqlc-generated code.
type NationalTeam struct {
	ID      string
	Name    string
	FlagURL string
}

// ListNationalTeams returns all national teams ordered by name.
func (r *NationalTeamRepository) ListNationalTeams(ctx context.Context) ([]NationalTeam, error) {
	rows, err := r.q.ListNationalTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("list national teams: %w", err)
	}

	teams := make([]NationalTeam, 0, len(rows))
	for _, row := range rows {
		teams = append(teams, toNationalTeam(row))
	}
	return teams, nil
}

// GetNationalTeamByID returns the national team with the given id.
// Returns ErrNationalTeamNotFound (wrapped) when no row exists so callers
// can branch with errors.Is(err, ErrNationalTeamNotFound).
func (r *NationalTeamRepository) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	row, err := r.q.GetNationalTeamByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("get national team by id: %w", ErrNationalTeamNotFound)
		}
		return "", fmt.Errorf("get national team by id: %w", err)
	}
	return row.Name, nil
}

// toNationalTeam converts a sqlc-generated NationalTeam row to the domain
// NationalTeam type.
func toNationalTeam(row sqlc.NationalTeam) NationalTeam {
	return NationalTeam{
		ID:      row.ID,
		Name:    row.Name,
		FlagURL: row.FlagUrl,
	}
}
