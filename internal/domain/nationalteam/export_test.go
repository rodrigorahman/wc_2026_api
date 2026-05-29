package nationalteam

import (
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
)

// ProvideServiceRepository exposes the unexported provideServiceRepository
// binding to external (_test) packages, so wiring tests can exercise the real
// repository→service adapter without living inside the package and creating an
// import cycle with internal/testutil (which now provides the bufconn E2E server
// and therefore imports this module).
func ProvideServiceRepository(repo *repository.NationalTeamRepository) service.NationalTeamRepository {
	return provideServiceRepository(repo)
}
