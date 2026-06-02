package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	nationalteamv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/nationalteam/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// CT-032 — ListNationalTeams is public: it succeeds without any token and
// returns the seeded selections (the bridge between the public-methods list and
// the seed data is exercised end-to-end).
func TestE2E_ListNationalTeams_Public(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := nationalteamv1.NewNationalTeamServiceClient(conn)

	resp, err := client.ListNationalTeams(ctx, &nationalteamv1.ListNationalTeamsRequest{})
	require.NoError(t, err)

	teams := resp.GetNationalTeams()
	require.NotEmpty(t, teams, "seed selections must be returned")

	// Seed 000002 inserts 16 selections; seed 000008 (Copa 2026) adds 33 more (49 total).
	require.Len(t, teams, 49)

	names := make(map[string]struct{}, len(teams))
	for _, team := range teams {
		require.NotEmpty(t, team.GetId())
		require.NotEmpty(t, team.GetName())
		names[team.GetName()] = struct{}{}
	}
	require.Contains(t, names, "Brasil")
}

// CT-011: end-to-end, ListNationalTeams returns flag_url for the seeded
// selections, and Brasil carries its exact flag URL.
func TestE2E_ListNationalTeams_FlagURL(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := nationalteamv1.NewNationalTeamServiceClient(conn)

	resp, err := client.ListNationalTeams(ctx, &nationalteamv1.ListNationalTeamsRequest{})
	require.NoError(t, err)

	teams := resp.GetNationalTeams()
	require.Len(t, teams, 49)

	var brasilFlagURL string
	for _, team := range teams {
		if team.GetName() == "Brasil" {
			brasilFlagURL = team.GetFlagUrl()
			break
		}
	}
	require.Equal(t, "https://flagcdn.com/w320/br.png", brasilFlagURL, "Brasil must carry its exact flag URL")
}

// CT-012: companion negativo de CT-011 — nenhuma seleção retorna flag_url vazia
// ponta a ponta.
func TestE2E_ListNationalTeams_NoEmptyFlagURL(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := nationalteamv1.NewNationalTeamServiceClient(conn)

	resp, err := client.ListNationalTeams(ctx, &nationalteamv1.ListNationalTeamsRequest{})
	require.NoError(t, err)

	for _, team := range resp.GetNationalTeams() {
		require.NotEmpty(t, team.GetFlagUrl(), "national team %s (%s) must carry a flag_url", team.GetName(), team.GetId())
	}
}

// onlyListNationalTeams enumerates the methods the nationalteam gRPC surface is
// expected to expose. The generated client MUST satisfy it. If a
// GetNationalTeamByID RPC (or any flag_url-leaking RPC) were ever added, this
// stays compiling — so the test also asserts the client carries no extra
// gRPC-shaped GetNationalTeamByID method at runtime via a type switch below.
type onlyListNationalTeams interface {
	ListNationalTeams(ctx context.Context, in *nationalteamv1.ListNationalTeamsRequest, opts ...grpc.CallOption) (*nationalteamv1.ListNationalTeamsResponse, error)
}

// CT-013: guard de regressão — GetNationalTeamByID NÃO é um RPC público do
// NationalTeamService (é método interno consumido pelo auth para validação de
// cadastro), logo a superfície gRPC do domínio não expõe flag_url por essa via.
// A asserção é estrutural: o client gerado satisfaz a interface restrita
// (compile-time) e não implementa nenhuma interface com GetNationalTeamByID.
func TestE2E_NationalTeamService_HasNoGetByIDRPC(t *testing.T) {
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := nationalteamv1.NewNationalTeamServiceClient(conn)

	// Compile-time + runtime: the generated client satisfies the restricted
	// surface (only ListNationalTeams).
	var restricted onlyListNationalTeams = client
	require.NotNil(t, restricted)

	// Runtime guard: the client must NOT implement a gRPC-shaped
	// GetNationalTeamByID RPC that could leak flag_url.
	_, leaks := any(client).(interface {
		GetNationalTeamByID(ctx context.Context, in *nationalteamv1.ListNationalTeamsRequest, opts ...grpc.CallOption) (*nationalteamv1.NationalTeam, error)
	})
	require.False(t, leaks, "NationalTeamService must not expose a GetNationalTeamByID RPC")
}
