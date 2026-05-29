package nationalteam_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// TestServiceRepositoryAdapter_NotFoundTranslation exercises the real wiring
// path: a concrete *repository.NationalTeamRepository wrapped by the
// nationalTeamRepositoryAdapter (as fx wires it) must translate the
// repository's not-found sentinel into service.ErrNationalTeamNotFound. This is
// the single point of translation (the service itself only propagates), so this
// test guards that contract against regression.
func TestServiceRepositoryAdapter_NotFoundTranslation(t *testing.T) {
	ctx := context.Background()

	db := testutil.TestNewDB(t)
	repo := repository.NewNationalTeamRepository(sqlc.New(db))
	adapter := nationalteam.ProvideServiceRepository(repo)

	_, err := adapter.GetNationalTeamByID(ctx, "00000000-0000-0000-0000-000000000000")

	require.Error(t, err)
	require.True(t, errors.Is(err, service.ErrNationalTeamNotFound),
		"adapter must translate repository not-found into service.ErrNationalTeamNotFound")
	require.False(t, errors.Is(err, repository.ErrNationalTeamNotFound),
		"adapter must not leak the repository-layer sentinel to the service")
}

// TestService_GetNationalTeamByID_NotFoundViaAdapter wires the real adapter into
// the service (as fx does) and verifies that an unknown id surfaces as
// service.ErrNationalTeamNotFound through errors.Is — proving the service
// propagates the adapter's translated sentinel rather than producing a raw
// error.
func TestService_GetNationalTeamByID_NotFoundViaAdapter(t *testing.T) {
	ctx := context.Background()

	db := testutil.TestNewDB(t)
	repo := repository.NewNationalTeamRepository(sqlc.New(db))
	svc := service.NewNationalTeamService(nationalteam.ProvideServiceRepository(repo))

	_, err := svc.GetNationalTeamByID(ctx, "00000000-0000-0000-0000-000000000000")

	require.Error(t, err)
	require.True(t, errors.Is(err, service.ErrNationalTeamNotFound),
		"service must surface service.ErrNationalTeamNotFound for an unknown id")
}
