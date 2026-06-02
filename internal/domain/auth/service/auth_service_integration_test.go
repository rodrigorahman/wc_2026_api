package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// integrationTeamID is a national-team id seeded by migration 000002 (Brasil),
// guaranteeing the FK constraint is satisfied when creating a user.
const integrationTeamID = "a1f3c5e7-0001-4000-8000-000000000001"

// repoServiceAdapter bridges the concrete *repository.UserRepository to
// service.UserRepository for the integration tests, translating the repository
// not-found sentinel and carrying the temp-password fields. It mirrors the
// production adapter in the auth fx module (unexported there) for the subset of
// methods the recovery flow exercises.
type repoServiceAdapter struct{ repo *repository.UserRepository }

func (a repoServiceAdapter) CreateUser(ctx context.Context, u service.User) error {
	_, err := a.repo.CreateUser(ctx, repository.User{
		ID:              u.ID,
		FullName:        u.FullName,
		Email:           u.Email,
		PasswordHash:    u.PasswordHash,
		NationalTeamIDs: u.NationalTeamIDs,
	})
	return err
}

func (a repoServiceAdapter) GetUserByEmail(ctx context.Context, email string) (service.User, error) {
	u, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return service.User{}, service.ErrUserNotFound
		}
		return service.User{}, err
	}
	return service.User{
		ID:                    u.ID,
		FullName:              u.FullName,
		Email:                 u.Email,
		PasswordHash:          u.PasswordHash,
		NationalTeamIDs:       u.NationalTeamIDs,
		TempPasswordHash:      u.TempPasswordHash,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt,
	}, nil
}

func (a repoServiceAdapter) GetUserByID(ctx context.Context, id string) (service.User, error) {
	u, err := a.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return service.User{}, service.ErrUserNotFound
		}
		return service.User{}, err
	}
	return service.User{
		ID:                    u.ID,
		Email:                 u.Email,
		PasswordHash:          u.PasswordHash,
		TempPasswordHash:      u.TempPasswordHash,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt,
	}, nil
}

// advanceableClock is a clock.Clock whose instant can be moved forward in tests,
// so temporary-password expiration can be reached without any real sleep.
type advanceableClock struct{ now time.Time }

func (c *advanceableClock) Now() time.Time { return c.now }

func (c *advanceableClock) advance(d time.Duration) { c.now = c.now.Add(d) }

func (a repoServiceAdapter) SetTempPassword(ctx context.Context, id, hash string, expiresAt time.Time) error {
	return a.repo.SetTempPassword(ctx, id, hash, expiresAt)
}

func (a repoServiceAdapter) UpdatePassword(ctx context.Context, id, hash string) error {
	return a.repo.UpdatePassword(ctx, id, hash)
}

// CT-026: against the real SQLite stack, processRecovery persists a bcrypt-valid
// temp-password hash with the correct expiration and triggers delivery.
func TestProcessRecovery_StackReal_TempPersistida(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	users := repoServiceAdapter{repo: repo}

	createdID := uuid.NewString()
	_, err := repo.CreateUser(ctx, repository.User{
		ID:              createdID,
		FullName:        "Maria Silva",
		Email:           testEmail,
		PasswordHash:    hashOf(t, testPassword),
		NationalTeamIDs: []string{integrationTeamID},
	})
	require.NoError(t, err)

	email := &emailSenderMock{}
	svc, err := service.NewAuthService(users, teamFound(), &tokenManagerMock{}, email, fixedClock{now: now}, zap.NewNop(), testCost)
	require.NoError(t, err)
	svc.SetGenTempPassword(func() (string, error) { return knownTempPassword, nil })

	require.NoError(t, svc.ProcessRecovery(ctx, testEmail))

	stored, err := repo.GetUserByID(ctx, createdID)
	require.NoError(t, err)
	require.NoError(t,
		bcrypt.CompareHashAndPassword([]byte(stored.TempPasswordHash), []byte(knownTempPassword)),
		"hash persisted in the database must be a valid bcrypt hash of the temp password")
	require.True(t, stored.TempPasswordExpiresAt.Equal(now.Add(15*time.Minute)),
		"persisted expires_at must be clock.Now()+15min")
	require.True(t, email.sendCalled, "delivery must have been triggered")
}

// CT-027: against the real stack, an unknown e-mail mutates nothing and sends no
// e-mail (CA-02).
func TestProcessRecovery_StackReal_EmailInexistente(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	users := repoServiceAdapter{repo: repo}

	existingID := uuid.NewString()
	_, err := repo.CreateUser(ctx, repository.User{
		ID:              existingID,
		FullName:        "Maria Silva",
		Email:           testEmail,
		PasswordHash:    hashOf(t, testPassword),
		NationalTeamIDs: []string{integrationTeamID},
	})
	require.NoError(t, err)

	email := &emailSenderMock{}
	svc, err := service.NewAuthService(users, teamFound(), &tokenManagerMock{}, email, fixedClock{now: now}, zap.NewNop(), testCost)
	require.NoError(t, err)
	svc.SetGenTempPassword(func() (string, error) { return knownTempPassword, nil })

	require.NoError(t, svc.ProcessRecovery(ctx, "ghost@example.com"))

	require.False(t, email.sendCalled, "no e-mail must be sent for an unknown address")
	stored, err := repo.GetUserByID(ctx, existingID)
	require.NoError(t, err)
	require.Empty(t, stored.TempPasswordHash, "the existing user must remain untouched")
	require.True(t, stored.TempPasswordExpiresAt.IsZero(), "no expiration must be set")
}

const integrationTempPassword = "TempXxx-STACK-7!"

// CT-023: against the real SQLite stack, logging in with an active temporary
// password grants access and signals PasswordChangeRequired=true (CA-04).
func TestLogin_TempValida_StackReal_True(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	users := repoServiceAdapter{repo: repo}

	createdID := uuid.NewString()
	_, err := repo.CreateUser(ctx, repository.User{
		ID:              createdID,
		FullName:        "Maria Silva",
		Email:           testEmail,
		PasswordHash:    hashOf(t, testPassword),
		NationalTeamIDs: []string{integrationTeamID},
	})
	require.NoError(t, err)
	require.NoError(t, repo.SetTempPassword(ctx, createdID, hashOf(t, integrationTempPassword), now.Add(15*time.Minute)))

	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) { return testToken, now.Add(time.Hour), nil },
	}
	svc, err := service.NewAuthService(users, teamFound(), tokens, &emailSenderMock{}, fixedClock{now: now}, zap.NewNop(), testCost)
	require.NoError(t, err)

	res, err := svc.Login(ctx, testEmail, integrationTempPassword)

	require.NoError(t, err)
	require.True(t, res.PasswordChangeRequired, "an active temp password from the real stack must require a change")
	require.Equal(t, testToken, res.AccessToken)
	require.Equal(t, createdID, tokens.generatedFor)
}

// CT-024: against the real stack, advancing the injected clock past the
// temporary-password expiration denies access deterministically — no sleep.
func TestLogin_TempExpirada_StackReal_Unauthenticated(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	users := repoServiceAdapter{repo: repo}

	createdID := uuid.NewString()
	_, err := repo.CreateUser(ctx, repository.User{
		ID:              createdID,
		FullName:        "Maria Silva",
		Email:           testEmail,
		PasswordHash:    hashOf(t, testPassword),
		NationalTeamIDs: []string{integrationTeamID},
	})
	require.NoError(t, err)
	require.NoError(t, repo.SetTempPassword(ctx, createdID, hashOf(t, integrationTempPassword), t0.Add(15*time.Minute)))

	clk := &advanceableClock{now: t0}
	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) {
			t.Fatal("Generate must not be called once the temp password is expired")
			return "", time.Time{}, nil
		},
	}
	svc, err := service.NewAuthService(users, teamFound(), tokens, &emailSenderMock{}, clk, zap.NewNop(), testCost)
	require.NoError(t, err)

	clk.advance(16 * time.Minute) // past expires_at = t0+15min, without sleeping.

	_, err = svc.Login(ctx, testEmail, integrationTempPassword)

	require.Equal(t, codes.Unauthenticated, status.Code(err), "an expired temp password must be denied")
	require.Empty(t, tokens.generatedFor)
}

// CT-025: against the real SQLite stack, ChangePassword with a valid temporary
// password persists the new password; afterwards the new password logs in
// normally (PasswordChangeRequired=false) and the temporary password is rejected
// — proving the temp columns were cleared in the same operation (CA-06/CA-07).
func TestChangePassword_StackReal_NovaSenhaAceita(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	const newPassword = "NovaSenha@2026!"

	db := testutil.TestNewDB(t)
	repo := repository.NewUserRepository(db, sqlc.New(db))
	users := repoServiceAdapter{repo: repo}

	createdID := uuid.NewString()
	_, err := repo.CreateUser(ctx, repository.User{
		ID:              createdID,
		FullName:        "Maria Silva",
		Email:           testEmail,
		PasswordHash:    hashOf(t, testPassword),
		NationalTeamIDs: []string{integrationTeamID},
	})
	require.NoError(t, err)
	require.NoError(t, repo.SetTempPassword(ctx, createdID, hashOf(t, integrationTempPassword), now.Add(15*time.Minute)))

	tokens := &tokenManagerMock{
		generateFn: func(string) (string, time.Time, error) { return testToken, now.Add(time.Hour), nil },
	}
	svc, err := service.NewAuthService(users, teamFound(), tokens, &emailSenderMock{}, fixedClock{now: now}, zap.NewNop(), testCost)
	require.NoError(t, err)

	require.NoError(t, svc.ChangePassword(ctx, createdID, integrationTempPassword, newPassword))

	res, err := svc.Login(ctx, testEmail, newPassword)
	require.NoError(t, err, "the new password must authenticate")
	require.False(t, res.PasswordChangeRequired, "the new permanent password must grant normal access")

	_, err = svc.Login(ctx, testEmail, integrationTempPassword)
	require.Equal(t, codes.Unauthenticated, status.Code(err), "the temp password must be rejected after the change")
}
