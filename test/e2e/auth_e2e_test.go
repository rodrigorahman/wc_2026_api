// Package e2e contains black-box end-to-end tests that dial the gRPC server over
// an in-memory bufconn connection (real interceptor chain, real fx modules).
package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

const (
	seededBrasilID    = "a1f3c5e7-0001-4000-8000-000000000001"
	seededArgentinaID = "a1f3c5e7-0002-4000-8000-000000000002"
	seededFrancaID    = "a1f3c5e7-0003-4000-8000-000000000003"
	// Itália está no seed 000002 mas não se classificou para a Copa 2026 (seed 000008),
	// portanto não tem jogos futuros — útil para exercitar o caminho "lista vazia".
	seededItaliaID = "a1f3c5e7-0009-4000-8000-000000000009"
)

// CT-056 — a valid Register with one selection over the full stack returns a
// non-empty user_id.
func TestE2E_Register_Valid(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	resp, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente E2E",
		Email:           "ct027@example.com",
		Password:        "senha-valida-123",
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetUserId())
}

// CT-028 — Register with a missing field is rejected by the protovalidate
// interceptor (codes.InvalidArgument) before reaching the handler.
func TestE2E_Register_MissingField_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	// full_name omitted → protovalidate rejects.
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		Email:           "ct028@example.com",
		Password:        "senha-valida-123",
		NationalTeamIds: []string{seededBrasilID},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-057 — Register → Login → GetMe(Bearer): the protected RPC returns the
// authenticated user's data (full_name, email, national_team_ids) populated with
// the selection chosen at registration.
func TestE2E_Register_Login_GetMe(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	conn := testutil.TestNewBufconnServer(t, clk)
	client := authv1.NewAuthServiceClient(conn)

	const (
		fullName = "Cliente CT029"
		email    = "ct029@example.com"
		password = "senha-valida-123"
	)

	reg, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        fullName,
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	login, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)
	require.NotEmpty(t, login.GetAccessToken())

	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+login.GetAccessToken())
	me, err := client.GetMe(authCtx, &authv1.GetMeRequest{})
	require.NoError(t, err)

	require.Equal(t, reg.GetUserId(), me.GetUserId())
	require.Equal(t, fullName, me.GetFullName())
	require.Equal(t, email, me.GetEmail())
	require.Equal(t, []string{seededBrasilID}, me.GetNationalTeamIds())
}

// CT-058 — Register with three distinct valid selections; GetMe returns all
// three.
func TestE2E_Register_ThreeSelections_GetMe(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	conn := testutil.TestNewBufconnServer(t, clk)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email    = "ct058@example.com"
		password = "senha-valida-123"
	)
	teamIDs := []string{seededBrasilID, seededArgentinaID, seededFrancaID}

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT058",
		Email:           email,
		Password:        password,
		NationalTeamIds: teamIDs,
	})
	require.NoError(t, err)

	login, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)

	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+login.GetAccessToken())
	me, err := client.GetMe(authCtx, &authv1.GetMeRequest{})
	require.NoError(t, err)
	require.ElementsMatch(t, teamIDs, me.GetNationalTeamIds())
}

// CT-059 — Register with an empty national_team_ids list is rejected by
// protovalidate (min_items: 1) before reaching the handler.
func TestE2E_Register_EmptySelections_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT059",
		Email:           "ct059@example.com",
		Password:        "senha-valida-123",
		NationalTeamIds: nil,
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-060 — Register with four selections is rejected by protovalidate
// (max_items: 3).
func TestE2E_Register_FourSelections_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName: "Cliente CT060",
		Email:    "ct060@example.com",
		Password: "senha-valida-123",
		NationalTeamIds: []string{
			seededBrasilID,
			seededArgentinaID,
			seededFrancaID,
			"a1f3c5e7-0004-4000-8000-000000000004",
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-061 — Register with a non-existent (but syntactically valid) selection is
// rejected with InvalidArgument by the service and no user is created.
func TestE2E_Register_NonexistentSelection_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	const email = "ct061@example.com"
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT061",
		Email:           email,
		Password:        "senha-valida-123",
		NationalTeamIds: []string{"00000000-0000-4000-8000-000000000000"},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	// The user must not have been created: a fresh registration with a valid
	// selection on the same e-mail must succeed (proving no prior row exists).
	resp, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT061",
		Email:           email,
		Password:        "senha-valida-123",
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetUserId())
}

// CT-062 — Register with a syntactically invalid UUID in national_team_ids is
// rejected by protovalidate (items.string.uuid).
func TestE2E_Register_InvalidUUIDInSelections_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT062",
		Email:           "ct062@example.com",
		Password:        "senha-valida-123",
		NationalTeamIds: []string{"not-a-uuid"},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-030 — GetMe without an authorization token is rejected with
// codes.Unauthenticated by the auth interceptor.
func TestE2E_GetMe_NoToken_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil)
	client := authv1.NewAuthServiceClient(conn)

	_, err := client.GetMe(ctx, &authv1.GetMeRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// CT-063 — GetMe with an expired token is rejected with codes.Unauthenticated,
// confirming the auth/JWT flow is unaffected by the N:N migration. The injected
// clock is advanced past the token TTL after issuance, so the previously valid
// token is now expired at validation time.
func TestE2E_GetMe_ExpiredToken_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	conn := testutil.TestNewBufconnServer(t, clk)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email    = "ct031@example.com"
		password = "senha-valida-123"
	)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT031",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	login, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)

	// Move time past the 1h TTL: the token issued above is now expired.
	clk.Advance(2 * time.Hour)

	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+login.GetAccessToken())
	_, err = client.GetMe(authCtx, &authv1.GetMeRequest{})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}
