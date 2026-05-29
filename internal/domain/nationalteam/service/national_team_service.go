// Package service implements the nationalteam domain logic. NationalTeamService
// is a read-only orchestration layer that delegates directly to the repository.
//
// Following the project's idiomatic interface placement, the repository
// interface this service depends on is declared HERE, in the consumer package.
// The repository package provides the concrete type that satisfies it, and fx
// binds concrete to interface in the wiring.
package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
)

// ErrNationalTeamNotFound is the service-layer sentinel for a missing national
// team. The repository-to-service adapter (in the module wiring) translates the
// repository's own sentinel into this one, so that the service's callers can
// branch via errors.Is without coupling to the storage-specific error type.
// The service itself only propagates this sentinel; it performs no translation.
var ErrNationalTeamNotFound = errors.New("national team not found")

// NationalTeam is the domain representation of a national team as consumed
// by this service. The repository maps its storage rows into this shape.
type NationalTeam struct {
	ID      string
	Name    string
	FlagURL string
}

// NationalTeamRepository abstracts read access to national teams. Declared
// in the consumer package (service) per the project's interface-placement
// convention.
type NationalTeamRepository interface {
	// ListNationalTeams returns all national teams ordered by name.
	ListNationalTeams(ctx context.Context) ([]repository.NationalTeam, error)
	// GetNationalTeamByID returns the name of the national team with the
	// given id, or ErrNationalTeamNotFound when none exists.
	GetNationalTeamByID(ctx context.Context, id string) (string, error)
}

// NationalTeamService provides read-only access to national teams.
type NationalTeamService struct {
	repo NationalTeamRepository
}

// NewNationalTeamService constructs a NationalTeamService backed by the
// given repository.
func NewNationalTeamService(repo NationalTeamRepository) *NationalTeamService {
	return &NationalTeamService{repo: repo}
}

// ListNationalTeams returns all national teams delegating to the repository.
func (s *NationalTeamService) ListNationalTeams(ctx context.Context) ([]NationalTeam, error) {
	rows, err := s.repo.ListNationalTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("list national teams: %w", err)
	}

	teams := make([]NationalTeam, 0, len(rows))
	for _, r := range rows {
		teams = append(teams, NationalTeam{ID: r.ID, Name: r.Name, FlagURL: r.FlagURL})
	}
	return teams, nil
}

// GetNationalTeamByID returns the name of the national team with the given
// id. Returns ErrNationalTeamNotFound (wrapped) when no national team exists
// with that id; the not-found translation is performed by the repository
// adapter (module wiring), so the service simply propagates whatever the
// repository returns. This method is consumed by the AuthService (T7) for
// registration validation via the auth/service.NationalTeamRepository
// interface.
func (s *NationalTeamService) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	name, err := s.repo.GetNationalTeamByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get national team by id: %w", err)
	}
	return name, nil
}
