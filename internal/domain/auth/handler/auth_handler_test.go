package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
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

// callChangePasswordAuthenticated invokes h.ChangePassword through the real JWT
// interceptor: it mints a token for sub via the production TokenManager, wires
// the production interceptor, and lets the interceptor inject the sub into the
// context exactly as it does in production. This exercises the legitimate
// authentication boundary (no exported test-only setter on production code).
func callChangePasswordAuthenticated(t *testing.T, h *handler.AuthHandler, sub string, req *authv1.ChangePasswordRequest) (*authv1.ChangePasswordResponse, error) {
	t.Helper()

	clk := fixedClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tm := token.NewTokenManager([]byte(testSecret), time.Hour, clk)

	tokenStr, _, err := tm.Generate(sub)
	require.NoError(t, err)

	md := metadata.Pairs("authorization", "Bearer "+tokenStr)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	intercept := interceptor.NewAuthJWTUnaryInterceptor(tm, nil)
	info := &grpc.UnaryServerInfo{FullMethod: authv1.AuthService_ChangePassword_FullMethodName}

	resp, err := intercept(ctx, req, info, func(ctx context.Context, _ any) (any, error) {
		return h.ChangePassword(ctx, req)
	})
	if resp == nil {
		return nil, err
	}
	return resp.(*authv1.ChangePasswordResponse), err
}

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

func (a userRepositoryAdapter) SetTempPassword(ctx context.Context, id, hash string, expiresAt time.Time) error {
	return a.repo.SetTempPassword(ctx, id, hash, expiresAt)
}

func (a userRepositoryAdapter) UpdatePassword(ctx context.Context, id, hash string) error {
	return a.repo.UpdatePassword(ctx, id, hash)
}

// noopNationalTeamRepo satisfies service.NationalTeamRepository. Login never
// touches it (only Register validates the national team), so its methods are
// never invoked on the paths under test.
type noopNationalTeamRepo struct{}

func (noopNationalTeamRepo) GetNationalTeamByID(context.Context, string) (string, error) {
	return "", service.ErrNationalTeamNotFound
}

// noopEmailSender satisfies service.EmailSender. The login paths under test
// never trigger recovery e-mails, so Send is never invoked.
type noopEmailSender struct{}

func (noopEmailSender) Send(context.Context, string, string, string) error { return nil }

// testSecret is a 32+ byte HMAC secret for the test token manager.
const testSecret = "test-secret-at-least-32-bytes-long!!"

// authServiceMock is a configurable mock of handler.AuthService for unit tests.
type authServiceMock struct {
	registerFn                func(ctx context.Context, params service.RegisterParams) (string, error)
	loginFn                   func(ctx context.Context, email, password string) (service.LoginResult, error)
	getUserFn                 func(ctx context.Context, id string) (service.User, error)
	requestPasswordRecoveryFn func(ctx context.Context, email string) (string, error)
	changePasswordFn          func(ctx context.Context, userID, tempPassword, newPassword string) error
}

func (m *authServiceMock) Register(ctx context.Context, params service.RegisterParams) (string, error) {
	return m.registerFn(ctx, params)
}

func (m *authServiceMock) Login(ctx context.Context, email, password string) (service.LoginResult, error) {
	return m.loginFn(ctx, email, password)
}

func (m *authServiceMock) GetUser(ctx context.Context, id string) (service.User, error) {
	return m.getUserFn(ctx, id)
}

func (m *authServiceMock) RequestPasswordRecovery(ctx context.Context, email string) (string, error) {
	return m.requestPasswordRecoveryFn(ctx, email)
}

func (m *authServiceMock) ChangePassword(ctx context.Context, userID, tempPassword, newPassword string) error {
	return m.changePasswordFn(ctx, userID, tempPassword, newPassword)
}

// CT-004 — TestRequestPasswordRecovery_RespondeGenericaImediata
//
// INVARIANT: RequestPasswordRecovery returns the generic pt-BR message with
// status OK when the service returns (msg, nil) — e-mail exists path.
func TestRequestPasswordRecovery_RespondeGenericaImediata(t *testing.T) {
	const genericMsg = "Se o e-mail estiver cadastrado, enviaremos as instruções de recuperação."

	svc := &authServiceMock{
		requestPasswordRecoveryFn: func(_ context.Context, _ string) (string, error) {
			return genericMsg, nil
		},
	}
	h := handler.NewAuthHandler(svc)

	resp, err := h.RequestPasswordRecovery(context.Background(), &authv1.RequestPasswordRecoveryRequest{
		Email: "user@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, genericMsg, resp.GetMessage())
}

// CT-005 — TestRequestPasswordRecovery_MesmaMsgEmailDesconhecido
//
// INVARIANT: RequestPasswordRecovery returns the same generic message for an
// unknown e-mail (anti-enumeration in the presentation layer). The handler only
// propagates what the service returns — which is the same string regardless of
// e-mail existence.
func TestRequestPasswordRecovery_MesmaMsgEmailDesconhecido(t *testing.T) {
	const genericMsg = "Se o e-mail estiver cadastrado, enviaremos as instruções de recuperação."

	svc := &authServiceMock{
		requestPasswordRecoveryFn: func(_ context.Context, _ string) (string, error) {
			return genericMsg, nil
		},
	}
	h := handler.NewAuthHandler(svc)

	resp, err := h.RequestPasswordRecovery(context.Background(), &authv1.RequestPasswordRecoveryRequest{
		Email: "nobody@example.com",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, genericMsg, resp.GetMessage())
}

// TestChangePassword_SubAusente_Unauthenticated verifies that ChangePassword
// returns codes.Unauthenticated when the sub claim is absent from the context
// (wiring fault — interceptor did not run or was bypassed).
func TestChangePassword_SubAusente_Unauthenticated(t *testing.T) {
	svc := &authServiceMock{}
	h := handler.NewAuthHandler(svc)

	resp, err := h.ChangePassword(context.Background(), &authv1.ChangePasswordRequest{
		TempPassword: "tmp",
		NewPassword:  "new",
	})

	require.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.Unauthenticated, st.Code())
	require.Equal(t, "não autenticado", st.Message())
}

// TestChangePassword_RepassaSubAoService verifies that the sub read from the
// context is the exact userID forwarded to the service — prevents IDOR.
func TestChangePassword_RepassaSubAoService(t *testing.T) {
	const expectedSub = "user-id-abc123"

	var capturedUserID string
	svc := &authServiceMock{
		changePasswordFn: func(_ context.Context, userID, _, _ string) error {
			capturedUserID = userID
			return nil
		},
	}
	h := handler.NewAuthHandler(svc)

	resp, err := callChangePasswordAuthenticated(t, h, expectedSub, &authv1.ChangePasswordRequest{
		TempPassword: "tmp123",
		NewPassword:  "newPass456",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, expectedSub, capturedUserID)
}

// TestLogin_PropagaPasswordChangeRequired verifies that the Login handler
// propagates the PasswordChangeRequired field from the service result.
func TestLogin_PropagaPasswordChangeRequired(t *testing.T) {
	fixedTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	svc := &authServiceMock{
		loginFn: func(_ context.Context, _, _ string) (service.LoginResult, error) {
			return service.LoginResult{
				AccessToken:            "tok",
				ExpiresAt:              fixedTime,
				PasswordChangeRequired: true,
			}, nil
		},
	}
	h := handler.NewAuthHandler(svc)

	resp, err := h.Login(context.Background(), &authv1.LoginRequest{
		Email:    "user@example.com",
		Password: "pass",
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.True(t, resp.GetPasswordChangeRequired())
}

// TestChangePassword_ErroPropagadoVerbatim verifies that the handler propagates
// service errors verbatim (codes.InvalidArgument) without re-translating.
func TestChangePassword_ErroPropagadoVerbatim(t *testing.T) {
	serviceErr := status.Error(codes.InvalidArgument, "senha temporária inválida ou expirada")

	svc := &authServiceMock{
		changePasswordFn: func(_ context.Context, _, _, _ string) error {
			return serviceErr
		},
	}
	h := handler.NewAuthHandler(svc)

	resp, err := callChangePasswordAuthenticated(t, h, "some-user-id", &authv1.ChangePasswordRequest{
		TempPassword: "wrong-tmp",
		NewPassword:  "new",
	})

	require.Nil(t, resp)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "senha temporária inválida ou expirada", st.Message())
}

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
		noopEmailSender{},
		clk,
		zap.NewNop(),
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
		noopEmailSender{},
		clk,
		zap.NewNop(),
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
