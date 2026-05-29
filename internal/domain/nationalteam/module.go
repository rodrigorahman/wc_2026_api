// Package nationalteam wires the nationalteam domain into the fx application
// graph.
//
// The module provides the nationalteam handler, service and repository. The
// concrete *repository.NationalTeamRepository is exported from this module so
// that the composition root (T12) can bind it to the
// auth/service.NationalTeamRepository interface consumed by the AuthService.
//
// This module is the architectural reference (US-06): three layers (repository,
// service, handler) each behind a narrow interface declared in its consumer,
// bound together by fx.
package nationalteam

import (
	"context"
	"errors"

	"go.uber.org/fx"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
)

// Module wires the nationalteam domain. Constructors are provided as concrete
// types:
//   - *repository.NationalTeamRepository — thin sqlc wrapper, exported so the
//     composition root can bind it to auth/service.NationalTeamRepository.
//   - *service.NationalTeamService — read-only orchestration, injected with
//     the service-layer adapter that translates repository sentinels.
//   - *handler.NationalTeamHandler — gRPC presenter, injected with a narrow
//     interface of *service.NationalTeamService.
var Module = fx.Module("nationalteam",
	fx.Provide(
		repository.NewNationalTeamRepository,
		provideServiceRepository,
		service.NewNationalTeamService,
		provideNationalTeamHandler,
	),
)

// provideServiceRepository binds the concrete repository to the service's
// NationalTeamRepository interface via the translating adapter.
func provideServiceRepository(repo *repository.NationalTeamRepository) service.NationalTeamRepository {
	return nationalTeamRepositoryAdapter{repo: repo}
}

// provideNationalTeamHandler adapts the concrete *service.NationalTeamService
// to the narrow handler.NationalTeamService interface.
func provideNationalTeamHandler(svc *service.NationalTeamService) *handler.NationalTeamHandler {
	return handler.NewNationalTeamHandler(svc)
}

// nationalTeamRepositoryAdapter bridges the concrete
// repository.NationalTeamRepository to the service.NationalTeamRepository
// interface. It translates repository.ErrNationalTeamNotFound into
// service.ErrNationalTeamNotFound so that the service can branch via
// errors.Is on its own sentinel without coupling to the storage layer.
type nationalTeamRepositoryAdapter struct {
	repo *repository.NationalTeamRepository
}

func (a nationalTeamRepositoryAdapter) ListNationalTeams(ctx context.Context) ([]repository.NationalTeam, error) {
	return a.repo.ListNationalTeams(ctx)
}

func (a nationalTeamRepositoryAdapter) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	name, err := a.repo.GetNationalTeamByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNationalTeamNotFound) {
			return "", service.ErrNationalTeamNotFound
		}
		return "", err
	}
	return name, nil
}

// Compile-time assertion: nationalTeamRepositoryAdapter satisfies the
// service-layer interface.
var _ service.NationalTeamRepository = nationalTeamRepositoryAdapter{}
