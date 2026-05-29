// Package repository provides the concrete data-access layer for the match
// domain. MatchRepository is a thin wrapper around sqlc-generated queries —
// no business logic lives here.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
)

// Match is the domain representation of a scheduled match.
type Match struct {
	ID       string
	Kickoff  time.Time
	HomeTeam MatchTeam
	AwayTeam MatchTeam
	Stadium  string
	City     string
	Stage    string
}

// MatchTeam is the domain representation of a team participating in a match.
type MatchTeam struct {
	ID      string
	Name    string
	FlagURL string
	Code    string
}

// MatchRepository retrieves matches using sqlc-generated queries. It is a
// concrete type; the interface it satisfies is declared in the consuming
// packages following Go interface-placement idioms.
type MatchRepository struct {
	q *sqlc.Queries
}

// NewMatchRepository constructs a MatchRepository backed by the given sqlc
// Queries handle.
func NewMatchRepository(q *sqlc.Queries) *MatchRepository {
	return &MatchRepository{q: q}
}

// ListUpcomingMatchesByUser returns matches whose kickoff is strictly after
// cutoff and whose home or away team is in the user's favorites. The result
// is ordered by kickoff ascending. Returns an empty (non-nil) slice when no
// matches are found.
func (r *MatchRepository) ListUpcomingMatchesByUser(ctx context.Context, userID string, cutoff time.Time) ([]Match, error) {
	rows, err := r.q.ListUpcomingMatchesByUser(ctx, sqlc.ListUpcomingMatchesByUserParams{
		Cutoff: cutoff,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("list upcoming matches by user: %w", err)
	}

	matches := make([]Match, 0, len(rows))
	for _, row := range rows {
		matches = append(matches, toMatch(row))
	}
	return matches, nil
}

// toMatch converts a sqlc-generated ListUpcomingMatchesByUserRow to the
// domain Match type. FlagUrl (sqlc) is mapped to FlagURL (Go idiom).
func toMatch(row sqlc.ListUpcomingMatchesByUserRow) Match {
	return Match{
		ID:      row.ID,
		Kickoff: row.Kickoff,
		HomeTeam: MatchTeam{
			ID:      row.HomeID,
			Name:    row.HomeName,
			FlagURL: row.HomeFlagUrl,
			Code:    row.HomeCode,
		},
		AwayTeam: MatchTeam{
			ID:      row.AwayID,
			Name:    row.AwayName,
			FlagURL: row.AwayFlagUrl,
			Code:    row.AwayCode,
		},
		Stadium: row.Stadium,
		City:    row.City,
		Stage:   row.Stage,
	}
}
