// Package e2e contains black-box end-to-end tests that dial the gRPC server over
// an in-memory bufconn connection (real interceptor chain, real fx modules).
package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// msgRecoveryRequested is the canonical generic response from
// RequestPasswordRecovery. Declared here to pin the exact string asserted in
// CT-Recovery-028 without importing the service package.
const msgRecoveryRequested = "Se o e-mail estiver cadastrado, enviaremos as instruções de recuperação."

// recoveryTempFromBody extracts the temporary password from the recovery e-mail
// body. The service formats it as "Sua senha temporária é: <temp>\n\n".
func recoveryTempFromBody(t *testing.T, body string) string {
	t.Helper()
	const prefix = "é: "
	idx := strings.Index(body, prefix)
	require.Greater(t, idx, -1, "recovery email body must contain the temp password marker")
	rest := body[idx+len(prefix):]
	end := strings.Index(rest, "\n")
	require.Greater(t, end, -1, "temp password must be followed by a newline")
	return rest[:end]
}

// waitForEmail blocks until the capturing sender delivers an email on its
// channel or the context is cancelled.
func waitForEmail(ctx context.Context, t *testing.T, sender *testutil.CapturingEmailSender) testutil.SentEmail {
	t.Helper()
	select {
	case mail := <-sender.Received:
		return mail
	case <-ctx.Done():
		t.Fatal("timed out waiting for recovery e-mail from background goroutine")
		return testutil.SentEmail{}
	}
}

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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, clk, nil)
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
	conn := testutil.TestNewBufconnServer(t, clk, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, nil, nil)
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
	conn := testutil.TestNewBufconnServer(t, clk, nil)
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

// ── Recovery E2E (CT-028..CT-034) ────────────────────────────────────────────
//
// Synchronisation note: RequestPasswordRecovery launches a goroutine. The
// CapturingEmailSender's buffered channel acts as the rendezvous point — the
// test blocks on waitForEmail until the goroutine completes Send, then reads
// the captured temp password from the body. No time.Sleep is used anywhere.

// CT-Recovery-028 — anti-enumeration: RequestPasswordRecovery returns the same
// generic message and OK status for both a registered e-mail and an unknown
// e-mail. Table-driven with 2 sub-cases.
func TestE2E_Recovery_GenericResponse_AntiEnumeration(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	sender := testutil.NewCapturingEmailSender()
	conn := testutil.TestNewBufconnServer(t, clk, sender)
	client := authv1.NewAuthServiceClient(conn)

	// Register a user so one e-mail is known.
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT-R028",
		Email:           "ct-r028-known@example.com",
		Password:        "senha-valida-123",
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	cases := []struct {
		name  string
		email string
	}{
		{"known_email", "ct-r028-known@example.com"},
		{"unknown_email", "ct-r028-ghost@example.com"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{
				Email: tc.email,
			})
			require.NoError(t, err, "RequestPasswordRecovery must always return OK")
			require.Equal(t, codes.OK, status.Code(err))
			require.Equal(t, msgRecoveryRequested, resp.GetMessage(),
				"message must be identical regardless of e-mail existence")
		})
	}
}

// CT-Recovery-029 — RequestPasswordRecovery is reachable without an
// authorization header (public method configured in T9). The test verifies
// codes.OK — not codes.Unauthenticated — confirming the JWT interceptor does
// not block the RPC.
func TestE2E_Recovery_RequestPasswordRecovery_Public(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := authv1.NewAuthServiceClient(conn)

	// No metadata — unauthenticated caller.
	resp, err := client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{
		Email: "ct-r029-any@example.com",
	})
	require.NoError(t, err)
	require.NotEqual(t, codes.Unauthenticated, status.Code(err),
		"RequestPasswordRecovery must be a public method; JWT interceptor must not block it")
	require.Equal(t, msgRecoveryRequested, resp.GetMessage())
}

// CT-Recovery-030 — ChangePassword without an Authorization header is blocked
// by the JWT interceptor (codes.Unauthenticated), symmetric with GetMe.
//
// The interceptor chain order is: recovery → logging → protovalidate → authJWT.
// Both temp_password (min_len:1) and new_password (min_len:8) must be
// non-empty so protovalidate passes and the request reaches the JWT interceptor,
// which then rejects it with Unauthenticated.
func TestE2E_Recovery_ChangePassword_NoToken_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	conn := testutil.TestNewBufconnServer(t, nil, nil)
	client := authv1.NewAuthServiceClient(conn)

	_, err := client.ChangePassword(ctx, &authv1.ChangePasswordRequest{
		TempPassword: "a-valid-temp-1",
		NewPassword:  "nova-senha-segura-2026!",
	})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

// CT-Recovery-031 — ChangePassword with a valid JWT but the wrong temporary
// password is rejected with codes.InvalidArgument by the service, confirming
// the handler is reached and the service enforces correctness.
func TestE2E_Recovery_ChangePassword_WrongTemp_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	sender := testutil.NewCapturingEmailSender()
	conn := testutil.TestNewBufconnServer(t, clk, sender)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email    = "ct-r031@example.com"
		password = "senha-valida-123"
	)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT-R031",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	// Trigger recovery to obtain an active temp password in the DB.
	_, err = client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{Email: email})
	require.NoError(t, err)

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	mail := waitForEmail(waitCtx, t, sender)
	tempPassword := recoveryTempFromBody(t, mail.Body)
	_ = tempPassword // we intentionally send the wrong one below

	// Login with temp to obtain an authenticated token.
	loginResp, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: tempPassword})
	require.NoError(t, err)
	require.True(t, loginResp.GetPasswordChangeRequired())

	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+loginResp.GetAccessToken())

	// Submit the wrong temp password.
	_, err = client.ChangePassword(authCtx, &authv1.ChangePasswordRequest{
		TempPassword: "wrong-temp-password!!",
		NewPassword:  "nova-senha-segura-2026!",
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-Recovery-032 — FULL RECOVERY FLOW (highest value):
// Register → RequestPasswordRecovery → Login(temp)[PasswordChangeRequired=true]
// → ChangePassword(temp, nova)[OK] → Login(nova)[OK, false]
// → Login(temp antiga)[Unauthenticated].
//
// The CapturingEmailSender synchronizes with the background goroutine so the
// temp password can be read from the email body without sleeping.
func TestE2E_Recovery_FluxoCompleto(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	sender := testutil.NewCapturingEmailSender()
	conn := testutil.TestNewBufconnServer(t, clk, sender)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email       = "ct-r032@example.com"
		password    = "senha-original-123"
		newPassword = "nova-senha-segura-2026!"
	)

	// Step 1: Register.
	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT-R032",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	// Step 2: RequestPasswordRecovery → goroutine sends e-mail.
	_, err = client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{Email: email})
	require.NoError(t, err)

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	mail := waitForEmail(waitCtx, t, sender)
	tempPassword := recoveryTempFromBody(t, mail.Body)
	require.NotEmpty(t, tempPassword)

	// Step 3: Login with temp → PasswordChangeRequired must be true.
	loginTemp, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: tempPassword})
	require.NoError(t, err)
	require.True(t, loginTemp.GetPasswordChangeRequired(),
		"login with active temp password must signal PasswordChangeRequired=true")
	require.NotEmpty(t, loginTemp.GetAccessToken())

	// Step 4: ChangePassword(temp, nova) → OK.
	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+loginTemp.GetAccessToken())
	_, err = client.ChangePassword(authCtx, &authv1.ChangePasswordRequest{
		TempPassword: tempPassword,
		NewPassword:  newPassword,
	})
	require.NoError(t, err, "ChangePassword with correct temp must succeed")

	// Step 5: Login with new password → OK, PasswordChangeRequired=false.
	loginNew, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: newPassword})
	require.NoError(t, err)
	require.False(t, loginNew.GetPasswordChangeRequired(),
		"login with permanent password must not require change")
	require.NotEmpty(t, loginNew.GetAccessToken())

	// Step 6: Login with old temp → Unauthenticated (temp was invalidated by ChangePassword).
	_, err = client.Login(ctx, &authv1.LoginRequest{Email: email, Password: tempPassword})
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err),
		"old temp password must be rejected after ChangePassword")
}

// CT-Recovery-033 — Login with the original (permanent) password while a
// temporary password is active returns PasswordChangeRequired=false. The
// permanent password always grants normal access (RN10/CA-10).
func TestE2E_Recovery_LoginOriginalComTempAtiva_False(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	sender := testutil.NewCapturingEmailSender()
	conn := testutil.TestNewBufconnServer(t, clk, sender)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email    = "ct-r033@example.com"
		password = "senha-original-456"
	)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT-R033",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	// Activate a temp password.
	_, err = client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{Email: email})
	require.NoError(t, err)

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = waitForEmail(waitCtx, t, sender) // drain channel; temp not needed here

	// Login with original password — temp is active but permanent wins.
	loginResp, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: password})
	require.NoError(t, err)
	require.False(t, loginResp.GetPasswordChangeRequired(),
		"login with permanent password must return PasswordChangeRequired=false even when a temp is active")
	require.NotEmpty(t, loginResp.GetAccessToken())
}

// CT-Recovery-034 — Login with an active, non-expired temporary password
// returns PasswordChangeRequired=true (companion positive case of CT-033).
func TestE2E_Recovery_LoginTemp_True(t *testing.T) {
	ctx := context.Background()
	clk := testutil.NewFixedClock(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	sender := testutil.NewCapturingEmailSender()
	conn := testutil.TestNewBufconnServer(t, clk, sender)
	client := authv1.NewAuthServiceClient(conn)

	const (
		email    = "ct-r034@example.com"
		password = "senha-original-789"
	)

	_, err := client.Register(ctx, &authv1.RegisterRequest{
		FullName:        "Cliente CT-R034",
		Email:           email,
		Password:        password,
		NationalTeamIds: []string{seededBrasilID},
	})
	require.NoError(t, err)

	// Activate a temp password.
	_, err = client.RequestPasswordRecovery(ctx, &authv1.RequestPasswordRecoveryRequest{Email: email})
	require.NoError(t, err)

	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	mail := waitForEmail(waitCtx, t, sender)
	tempPassword := recoveryTempFromBody(t, mail.Body)

	// Login with temp — clock not advanced, temp is still active.
	loginResp, err := client.Login(ctx, &authv1.LoginRequest{Email: email, Password: tempPassword})
	require.NoError(t, err)
	require.True(t, loginResp.GetPasswordChangeRequired(),
		"login with active temp password must return PasswordChangeRequired=true")
	require.NotEmpty(t, loginResp.GetAccessToken())
}
