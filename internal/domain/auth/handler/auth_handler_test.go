package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// seededNationalTeamID is one of the rows seeded by migration 000002 (Brasil).
// A real id satisfies the FK constraint when pre-creating the test user.
const seededNationalTeamID = "a1f3c5e7-0001-4000-8000-000000000001"

const (
	correctPassword = "senha-correta-123"
	wrongPassword   = "senha-errada-999"
	knownEmail      = "login@example.com"
)

// fixedClock is a deterministic Clock so token issuance is reproducible.
type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// userRepositoryAdapter mirrors the production bind in internal/auth/module.go:
// it adapts the concrete repository to the service.UserRepository interface and
// translates repository.ErrUserNotFound into service.ErrUserNotFound. The same
// translation is exercised end-to-end here (unknown-email Login path) so the
// carry-over from T7/T8 is protected by an integration test rather than a
// reimplementation.
type userRepositoryAdapter struct {
	repo *repository.UserRepository
}

func (a userRepositoryAdapter) CreateUser(ctx context.Context, u service.User) error {
	_, err := a.repo.CreateUser(ctx, repository.User{
		ID:              u.ID,
		FullName:        u.FullName,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		NationalTeamIDs: u.NationalTeamIDs,
	})
	return err
}

func (a userRepositoryAdapter) GetUserByEmail(ctx context.Context, email string) (service.User, error) {
	u, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return service.User{}, service.ErrUserNotFound
		}
		return service.User{}, err
	}
	return service.User{
		ID:              u.ID,
		FullName:        u.FullName,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		NationalTeamIDs: u.NationalTeamIDs,
	}, nil
}

func (a userRepositoryAdapter) GetUserByID(ctx context.Context, id string) (service.User, error) {
	u, err := a.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return service.User{}, service.ErrUserNotFound
		}
		return service.User{}, err
	}
	return service.User{
		ID:              u.ID,
		FullName:        u.FullName,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		NationalTeamIDs: u.NationalTeamIDs,
	}, nil
}

// noopNationalTeamRepo satisfies service.NationalTeamRepository. Login never
// touches it (only Register validates the national team), so its methods are
// never invoked on the paths under test.
type noopNationalTeamRepo struct{}

func (noopNationalTeamRepo) GetNationalTeamByID(context.Context, string) (string, error) {
	return "", service.ErrNationalTeamNotFound
}

// testSecret is a 32+ byte HMAC secret for the test token manager.
const testSecret = "test-secret-at-least-32-bytes-long!!"

// CT-023 — TestIntegration_Login_WrongPassword
//
// INVARIANT: Login through the real handler→service→repository→DB stack with a
// correct e-mail but wrong password returns codes.Unauthenticated with the
// generic credential message (no field-specific leak).
// OWNING_LAYER: service-integration. real_execution_boundary: db.
func TestIntegration_Login_WrongPassword(t *testing.T) {
	ctx := context.Background()

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	clk := fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tm := token.NewTokenManager([]byte(testSecret), time.Hour, clk)

	svc, err := service.NewAuthService(
		userRepositoryAdapter{repo: repo},
		noopNationalTeamRepo{},
		tm,
		clk,
		bcrypt.MinCost,
	)
	require.NoError(t, err)
	h := handler.NewAuthHandler(svc)

	// Pre-create the user with a known bcrypt hash of correctPassword.
	hash, err := bcrypt.GenerateFromPassword([]byte(correctPassword), bcrypt.MinCost)
	require.NoError(t, err)
	_, err = repo.CreateUser(ctx, repository.User{
		ID:              uuid.NewString(),
		FullName:        "Login User",
		Email:           knownEmail,
		PasswordHash:    string(hash),
		NationalTeamIDs: []string{seededNationalTeamID},
	})
	require.NoError(t, err)

	// Wrong password for an existing e-mail.
	resp, err := h.Login(ctx, &authv1.LoginRequest{
		Email:    knownEmail,
		Password: wrongPassword,
	})

	require.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error must be a gRPC status")
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Equal(t, "e-mail ou senha inválidos", st.Message(), "message must be generic (no field-specific leak)")
}

// TestIntegration_Login_UnknownEmail_TranslatesNotFound exercises the adapter's
// repository.ErrUserNotFound → service.ErrUserNotFound translation through the
// real stack. Without that translation the service would not branch on
// errors.Is(err, service.ErrUserNotFound) and would return codes.Internal
// instead of the generic Unauthenticated — this protects the T7/T8 carry-over.
//
// INVARIANT: Login with an e-mail that has no row returns codes.Unauthenticated
// with the same generic message as the wrong-password path (anti-enumeration).
func TestIntegration_Login_UnknownEmail_TranslatesNotFound(t *testing.T) {
	ctx := context.Background()

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	clk := fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tm := token.NewTokenManager([]byte(testSecret), time.Hour, clk)

	svc, err := service.NewAuthService(
		userRepositoryAdapter{repo: repo},
		noopNationalTeamRepo{},
		tm,
		clk,
		bcrypt.MinCost,
	)
	require.NoError(t, err)
	h := handler.NewAuthHandler(svc)

	// No user created: the e-mail is unknown.
	resp, err := h.Login(ctx, &authv1.LoginRequest{
		Email:    "nobody@example.com",
		Password: "irrelevant",
	})

	require.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok, "error must be a gRPC status")
	require.Equal(t, codes.Unauthenticated, st.Code(),
		"unknown e-mail must map to Unauthenticated, not Internal — proves ErrUserNotFound was translated")
	require.Equal(t, "e-mail ou senha inválidos", st.Message())
}
