package match

import (
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/service"
)

// ProvideServiceRepository exposes the unexported provideServiceRepository
// binding to external (_test) packages, so wiring tests can exercise the real
// repository→service adapter without living inside the package.
func ProvideServiceRepository(repo *repository.MatchRepository) service.MatchRepository {
	return provideServiceRepository(repo)
}
