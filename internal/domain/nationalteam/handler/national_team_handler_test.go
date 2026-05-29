package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/service"
	nationalteamv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/nationalteam/v1"
)

// --- Manual mock -------------------------------------------------------------

type nationalTeamServiceMock struct {
	listFn     func(ctx context.Context) ([]service.NationalTeam, error)
	listCalled bool
}

func (m *nationalTeamServiceMock) ListNationalTeams(ctx context.Context) ([]service.NationalTeam, error) {
	m.listCalled = true
	return m.listFn(ctx)
}

// CT-010: o handler mapeia service.FlagURL → proto flag_url ao montar a
// resposta. Valida que o handler invoca o service e propaga o campo (incluindo
// id/name) sem alterá-lo.
func TestListNationalTeams_MapsFlagURLToProto(t *testing.T) {
	ctx := context.Background()

	svcTeams := []service.NationalTeam{
		{ID: "id-1", Name: "Brasil", FlagURL: "https://flagcdn.com/w320/br.png"},
		{ID: "id-2", Name: "Coreia do Sul", FlagURL: "https://flagcdn.com/w320/kr.png"},
	}

	mock := &nationalTeamServiceMock{
		listFn: func(context.Context) ([]service.NationalTeam, error) {
			return svcTeams, nil
		},
	}

	h := handler.NewNationalTeamHandler(mock)

	resp, err := h.ListNationalTeams(ctx, &nationalteamv1.ListNationalTeamsRequest{})

	require.NoError(t, err)
	require.True(t, mock.listCalled, "handler must call ListNationalTeams on the service")

	got := resp.GetNationalTeams()
	require.Len(t, got, len(svcTeams))
	require.Equal(t, svcTeams[0].ID, got[0].GetId())
	require.Equal(t, svcTeams[0].Name, got[0].GetName())
	require.Equal(t, svcTeams[0].FlagURL, got[0].GetFlagUrl(), "handler must map FlagURL to proto flag_url")
	require.Equal(t, svcTeams[1].FlagURL, got[1].GetFlagUrl(), "handler must map FlagURL to proto flag_url")
}
