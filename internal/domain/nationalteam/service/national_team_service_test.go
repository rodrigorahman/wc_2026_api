package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
)

// --- Manual mock -------------------------------------------------------------

type nationalTeamRepoMock struct {
	listFn    func(ctx context.Context) ([]repository.NationalTeam, error)
	getByIDFn func(ctx context.Context, id string) (string, error)

	listCalled    bool
	getByIDCalled bool
	getByIDArg    string
}

func (m *nationalTeamRepoMock) ListNationalTeams(ctx context.Context) ([]repository.NationalTeam, error) {
	m.listCalled = true
	return m.listFn(ctx)
}

func (m *nationalTeamRepoMock) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	m.getByIDCalled = true
	m.getByIDArg = id
	return m.getByIDFn(ctx, id)
}

// --- CT-015 ------------------------------------------------------------------

// CT-015: TestListNationalTeams_ReturnsList — service retorna a lista do
// repository. Valida: (a) o repository é invocado, (b) o service não dropa
// o retorno, (c) os campos são mapeados corretamente.
func TestListNationalTeams_ReturnsList(t *testing.T) {
	ctx := context.Background()

	repoTeams := []repository.NationalTeam{
		{ID: "id-1", Name: "Brasil", FlagURL: "https://flagcdn.com/w320/br.png"},
		{ID: "id-2", Name: "Argentina", FlagURL: "https://flagcdn.com/w320/ar.png"},
	}

	mock := &nationalTeamRepoMock{
		listFn: func(context.Context) ([]repository.NationalTeam, error) {
			return repoTeams, nil
		},
	}

	svc := service.NewNationalTeamService(mock)

	teams, err := svc.ListNationalTeams(ctx)

	require.NoError(t, err)
	require.True(t, mock.listCalled, "service must call ListNationalTeams on the repository")
	require.Len(t, teams, len(repoTeams))
	require.Equal(t, repoTeams[0].ID, teams[0].ID, "service must map the repository ID unchanged")
	require.Equal(t, repoTeams[0].Name, teams[0].Name, "service must map the repository Name unchanged")
	require.Equal(t, repoTeams[0].FlagURL, teams[0].FlagURL, "service must propagate the repository FlagURL unchanged")
	require.Equal(t, repoTeams[1].ID, teams[1].ID, "service must map the repository ID unchanged")
	require.Equal(t, repoTeams[1].Name, teams[1].Name, "service must map the repository Name unchanged")
	require.Equal(t, repoTeams[1].FlagURL, teams[1].FlagURL, "service must propagate the repository FlagURL unchanged")
}

// CT-009: companion negativo de CT-008 — o service é um mapper puro: se o
// repository devolve FlagURL vazia, o service repassa "" e não enriquece o
// campo com nenhum valor derivado.
func TestListNationalTeams_DoesNotEnrichEmptyFlagURL(t *testing.T) {
	ctx := context.Background()

	mock := &nationalTeamRepoMock{
		listFn: func(context.Context) ([]repository.NationalTeam, error) {
			return []repository.NationalTeam{{ID: "id-1", Name: "Brasil", FlagURL: ""}}, nil
		},
	}

	svc := service.NewNationalTeamService(mock)

	teams, err := svc.ListNationalTeams(ctx)

	require.NoError(t, err)
	require.Len(t, teams, 1)
	require.Empty(t, teams[0].FlagURL, "service must not enrich an empty FlagURL")
}

// TestListNationalTeams_RepoError — service propaga erro do repository.
func TestListNationalTeams_RepoError(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("database unavailable")

	mock := &nationalTeamRepoMock{
		listFn: func(context.Context) ([]repository.NationalTeam, error) {
			return nil, repoErr
		},
	}

	svc := service.NewNationalTeamService(mock)

	_, err := svc.ListNationalTeams(ctx)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
}

// TestGetNationalTeamByID_Found — service retorna nome ao encontrar a seleção.
func TestGetNationalTeamByID_Found(t *testing.T) {
	ctx := context.Background()
	const id = "id-1"

	mock := &nationalTeamRepoMock{
		getByIDFn: func(_ context.Context, _ string) (string, error) {
			return "Brasil", nil
		},
	}

	svc := service.NewNationalTeamService(mock)

	name, err := svc.GetNationalTeamByID(ctx, id)

	require.NoError(t, err)
	require.True(t, mock.getByIDCalled, "service must call GetNationalTeamByID on the repository")
	require.Equal(t, id, mock.getByIDArg, "service must forward the id unchanged")
	require.Equal(t, "Brasil", name, "service must return the name from the repository unchanged")
}

// TestGetNationalTeamByID_NotFound — a tradução para service.ErrNationalTeamNotFound
// é feita pelo adapter no wiring (ver module_test.go). A interface que o service
// consome (NationalTeamRepository) já entrega o sentinel do service; o service
// apenas o propaga (preservando errors.Is via wrapping). Este teste planta o
// sentinel do service no mock — fiel ao que o adapter retorna em produção — e
// verifica a propagação.
func TestGetNationalTeamByID_NotFound(t *testing.T) {
	ctx := context.Background()

	mock := &nationalTeamRepoMock{
		getByIDFn: func(context.Context, string) (string, error) {
			return "", service.ErrNationalTeamNotFound
		},
	}

	svc := service.NewNationalTeamService(mock)

	_, err := svc.GetNationalTeamByID(ctx, "nonexistent")

	require.Error(t, err)
	require.ErrorIs(t, err, service.ErrNationalTeamNotFound, "service must propagate ErrNationalTeamNotFound preserving errors.Is")
}
