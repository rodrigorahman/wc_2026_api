package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	matchv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/match/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// CT-001 — ListUpcomingMatches sem token é barrado pelo interceptor JWT com
// codes.Unauthenticated, provando que o RPC não é público.
func TestE2E_ListUpcomingMatches_NoToken_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := matchv1.NewMatchServiceClient(conn)

	_, err := client.ListUpcomingMatches(ctx, &matchv1.ListUpcomingMatchesRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// CT-002 — ListUpcomingMatches com usuário autenticado cuja única favorita não
// tem jogos futuros (Itália, cadastrada mas fora da Copa 2026) retorna lista vazia
// (não erro), provando que vazio != erro ponta a ponta mesmo com o seed de jogos.
func TestE2E_ListUpcomingMatches_AuthenticatedNoFavorites_Empty(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)

	authClient := authv1.NewAuthServiceClient(conn)
	matchClient := matchv1.NewMatchServiceClient(conn)

	const (
		email    = "ct002-match@example.com"
		password = "senha-valida-123"
	)

	_, err := authClient.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT002",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededItaliaID},
	})
	require.NoError(t, err)

	login, err := authClient.Login(ctx, &authv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)
	require.NotEmpty(t, login.GetAccessToken())

	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+login.GetAccessToken())
	resp, err := matchClient.ListUpcomingMatches(authCtx, &matchv1.ListUpcomingMatchesRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.GetMatches())
}
