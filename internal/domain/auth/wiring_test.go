package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam"
	"github.com/rodrigorahman/wc_2026_api/internal/server"
)

// seededBrasilID and seededArgentinaID are national-team ids present in the seed
// (migration 000002).
const (
	seededBrasilID    = "a1f3c5e7-0001-4000-8000-000000000001"
	seededArgentinaID = "a1f3c5e7-0002-4000-8000-000000000002"
)

// wiringTestSecret is a 32+ byte HMAC secret satisfying config's fail-fast rule.
const wiringTestSecret = "wiring-secret-at-least-32-bytes-ok!!"

// wired holds the production-wired dependencies pulled out of a started fx app
// for cross-module integration assertions.
type wired struct {
	fx.In

	Service *service.AuthService
	Tokens  *token.TokenManager
	Queries *sqlc.Queries
}

// startWiredApp builds and starts the real fx graph (db + auth + nationalteam +
// server providers) over a migrated DB, extracts the production-wired
// AuthService, TokenManager and Queries, and registers cleanup. The clock is the
// decorated test clock so token timing is deterministic. Building through fx (not
// by hand) is the point: it exercises the real userRepositoryAdapter and the
// real NationalTeam bridge as production wires them.
func startWiredApp(t *testing.T, clk clock.Clock) wired {
	t.Helper()

	cfg := config.Config{
		DBPath:    t.TempDir() + "/wiring.db",
		JWTSecret: wiringTestSecret,
		JWTTTL:    time.Hour,
		GRPCPort:  "0",
	}

	var out wired
	app := fx.New(
		db.Module,
		auth.Module,
		nationalteam.Module,
		server.Providers,
		fx.Supply(cfg),
		fx.Decorate(func(clock.Clock) clock.Clock { return clk }),
		fx.Populate(&out),
		fx.NopLogger,
	)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStart()
	require.NoError(t, app.Start(startCtx))
	t.Cleanup(func() {
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelStop()
		_ = app.Stop(stopCtx)
	})

	return out
}

// CT-022 — Register(seed selection) → Login; the issued JWT is verifiable and
// the password is stored as a hash (not plaintext) in the DB.
func TestIntegration_Register_Login_JWT(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	w := startWiredApp(t, clk)

	email := "ct022@example.com"
	password := "senha-forte-123"

	userID, err := w.Service.Register(ctx, service.RegisterParams{
		FullName:        "Cliente CT022",
		Email:           email,
		Password:        password,
		NationalTeamIDs: []string{seededBrasilID},
	})
	require.NoError(t, err)
	require.NotEmpty(t, userID)

	login, err := w.Service.Login(ctx, email, password)
	require.NoError(t, err)
	require.NotEmpty(t, login.AccessToken)

	// The token must validate to the registered user's id.
	sub, err := w.Tokens.Validate(login.AccessToken)
	require.NoError(t, err)
	require.Equal(t, userID, sub)

	// The stored hash must verify the original password but never equal it.
	row, err := w.Queries.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotEqual(t, password, row.PasswordHash)
}

// CT-054 — Register with a multi-selection list works through the real fx graph
// (userRepositoryAdapter + transactional repository), and GetUser returns the
// exact selections persisted.
func TestIntegration_Register_MultiSelection_RoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	w := startWiredApp(t, clk)

	teamIDs := []string{seededBrasilID, seededArgentinaID}
	userID, err := w.Service.Register(ctx, service.RegisterParams{
		FullName:        "Cliente CT054",
		Email:           "ct054@example.com",
		Password:        "senha-forte-123",
		NationalTeamIDs: teamIDs,
	})
	require.NoError(t, err)
	require.NotEmpty(t, userID)

	got, err := w.Service.GetUser(ctx, userID)
	require.NoError(t, err)
	require.ElementsMatch(t, teamIDs, got.NationalTeamIDs)
}

// CT-026 — the persisted password_hash differs from the original password and is a
// valid bcrypt hash that verifies it.
func TestIntegration_PasswordHash_RealDB(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	w := startWiredApp(t, clk)

	email := "ct026@example.com"
	password := "outra-senha-456"

	_, err := w.Service.Register(ctx, service.RegisterParams{
		FullName:        "Cliente CT026",
		Email:           email,
		Password:        password,
		NationalTeamIDs: []string{seededBrasilID},
	})
	require.NoError(t, err)

	row, err := w.Queries.GetUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotEqual(t, password, row.PasswordHash, "password_hash must not be the plaintext password")
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)),
		"stored hash must verify the original password",
	)
}

// CT-034 — two registrations yield distinct, valid UUID v4 ids.
func TestIntegration_UUIDv4_Unique(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	w := startWiredApp(t, clk)

	id1, err := w.Service.Register(ctx, service.RegisterParams{
		FullName: "U1", Email: "u1@example.com", Password: "senha-1-123", NationalTeamIDs: []string{seededBrasilID},
	})
	require.NoError(t, err)
	id2, err := w.Service.Register(ctx, service.RegisterParams{
		FullName: "U2", Email: "u2@example.com", Password: "senha-2-123", NationalTeamIDs: []string{seededBrasilID},
	})
	require.NoError(t, err)

	require.NotEqual(t, id1, id2)

	parsed1, err := uuid.Parse(id1)
	require.NoError(t, err)
	require.Equal(t, uuid.Version(4), parsed1.Version())

	parsed2, err := uuid.Parse(id2)
	require.NoError(t, err)
	require.Equal(t, uuid.Version(4), parsed2.Version())
}

// CT-055 — Register with an unknown national team is rejected with
// codes.InvalidArgument, proving the server's NationalTeam bridge translates the
// repository sentinel into the auth service sentinel through the real wiring.
func TestIntegration_Register_UnknownNationalTeam_InvalidArgument(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC))
	w := startWiredApp(t, clk)

	_, err := w.Service.Register(ctx, service.RegisterParams{
		FullName:        "Sem Seleção",
		Email:           "noteam@example.com",
		Password:        "senha-x-123",
		NationalTeamIDs: []string{"00000000-0000-4000-8000-000000000000"},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

// CT-064 — the full fx graph (db + auth + nationalteam + server providers)
// builds without error with the new N:N UserRepository shape: every dependency
// the auth module declares (validator, public methods, NationalTeam bridge,
// config) is satisfied — including the *sql.DB now required by the repository —
// and the *grpc.Server is constructible. fx.ValidateApp resolves the graph
// without running lifecycle.
func TestIntegration_FxWiring_Smoke(t *testing.T) {
	cfg := config.Config{
		DBPath:    t.TempDir() + "/smoke.db",
		JWTSecret: wiringTestSecret,
		JWTTTL:    time.Hour,
		GRPCPort:  "0",
	}

	err := fx.ValidateApp(
		db.Module,
		auth.Module,
		nationalteam.Module,
		server.Providers,
		fx.Supply(cfg),
		fx.Invoke(func(*grpc.Server) {}),
	)
	require.NoError(t, err)
}

// fixedClock is a deterministic Clock for the wiring tests.
type fixedClock struct{ now time.Time }

func newFixedClock(now time.Time) *fixedClock { return &fixedClock{now: now} }

func (c *fixedClock) Now() time.Time { return c.now }
