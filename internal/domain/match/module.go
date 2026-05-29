// Package match wires the match domain into the fx application graph.
//
// The module provides the match handler, service and repository. The clock.Clock
// is consumed from the shared graph (provided by auth.Module) — it is not
// re-provided here to avoid a type collision in the fx graph.
package match

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/service"
)

// Module wires the match domain. Constructors are provided as concrete types:
//   - *repository.MatchRepository — thin sqlc wrapper.
//   - service.MatchRepository — adapter binding the concrete repository to the
//     service-layer interface (no sentinel translation needed; match has none).
//   - *service.MatchService — read-only orchestration, consuming the repository
//     and the clock.Clock from the shared graph.
//   - *handler.MatchHandler — gRPC presenter, injected with a narrow interface
//     of *service.MatchService.
var Module = fx.Module("match",
	fx.Provide(
		repository.NewMatchRepository,
		provideServiceRepository,
		service.NewMatchService,
		provideMatchHandler,
	),
)

// provideServiceRepository binds the concrete *repository.MatchRepository to
// the service.MatchRepository interface. No sentinel translation is needed
// because the match domain defines no repository-level sentinel errors.
func provideServiceRepository(repo *repository.MatchRepository) service.MatchRepository {
	return matchRepositoryAdapter{repo: repo}
}

// provideMatchHandler adapts the concrete *service.MatchService to the narrow
// handler.MatchService interface.
func provideMatchHandler(svc *service.MatchService) *handler.MatchHandler {
	return handler.NewMatchHandler(svc)
}

// matchRepositoryAdapter bridges the concrete *repository.MatchRepository to
// the service.MatchRepository interface. Because the match domain has no
// sentinel errors at the repository layer, this adapter is a pure delegation.
type matchRepositoryAdapter struct {
	repo *repository.MatchRepository
}

func (a matchRepositoryAdapter) ListUpcomingMatchesByUser(
	ctx context.Context,
	userID string,
	cutoff time.Time,
) ([]repository.Match, error) {
	return a.repo.ListUpcomingMatchesByUser(ctx, userID, cutoff)
}

// Compile-time assertion: matchRepositoryAdapter satisfies service.MatchRepository.
var _ service.MatchRepository = matchRepositoryAdapter{}
