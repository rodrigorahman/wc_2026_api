// Package service implements the match domain logic. MatchService is a
// read-only orchestration layer that calculates the cutoff from the injected
// clock and delegates directly to the repository.
//
// Following the project's idiomatic interface placement, the repository
// interface this service depends on is declared HERE, in the consumer package.
// The repository package provides the concrete type that satisfies it, and fx
// binds concrete to interface in the wiring.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
)

// MatchRepository abstracts read access to upcoming matches. Declared in the
// consumer package (service) per the project's interface-placement convention.
type MatchRepository interface {
	// ListUpcomingMatchesByUser returns matches whose kickoff is strictly after
	// cutoff and whose home or away team is in the user's favorites.
	ListUpcomingMatchesByUser(ctx context.Context, userID string, cutoff time.Time) ([]repository.Match, error)
}

// Match is the domain representation of a scheduled match as consumed by this
// service. The repository maps its storage rows into this shape.
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

// MatchService provides read-only access to upcoming matches filtered by the
// user's favorite national teams.
type MatchService struct {
	repo MatchRepository
	clk  clock.Clock
}

// NewMatchService constructs a MatchService backed by the given repository and
// clock.
func NewMatchService(repo MatchRepository, clk clock.Clock) *MatchService {
	return &MatchService{repo: repo, clk: clk}
}

// ListUpcomingMatches returns matches whose kickoff is strictly after the
// current instant (from the injected clock) and whose home or away team is in
// the user's favorites. An empty result is not an error.
func (s *MatchService) ListUpcomingMatches(ctx context.Context, userID string) ([]Match, error) {
	cutoff := s.clk.Now()

	rows, err := s.repo.ListUpcomingMatchesByUser(ctx, userID, cutoff)
	if err != nil {
		return nil, fmt.Errorf("list upcoming matches: %w", err)
	}

	matches := make([]Match, 0, len(rows))
	for _, r := range rows {
		matches = append(matches, Match{
			ID:      r.ID,
			Kickoff: r.Kickoff,
			HomeTeam: MatchTeam{
				ID:      r.HomeTeam.ID,
				Name:    r.HomeTeam.Name,
				FlagURL: r.HomeTeam.FlagURL,
				Code:    r.HomeTeam.Code,
			},
			AwayTeam: MatchTeam{
				ID:      r.AwayTeam.ID,
				Name:    r.AwayTeam.Name,
				FlagURL: r.AwayTeam.FlagURL,
				Code:    r.AwayTeam.Code,
			},
			Stadium: r.Stadium,
			City:    r.City,
			Stage:   r.Stage,
		})
	}
	return matches, nil
}
