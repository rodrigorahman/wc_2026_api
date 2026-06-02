// Package auth wires the authentication domain into the fx application graph.
//
// The module provides the auth handler, service, repository, token manager and
// the two cross-cutting interceptors, and binds the concrete repository and
// token-manager types to the interfaces declared in their consumer packages.
//
// The composition root (T12) supplies the cross-domain inputs this module
// depends on but does not own: the NationalTeamRepository (bound by the
// nationalteam module), the protovalidate validator, and the list of public
// gRPC methods. The interceptor chain order itself is assembled in the
// composition root (see T10 §7).
package auth

import (
	"context"
	"errors"
	"time"

	"buf.build/go/protovalidate"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/interceptor"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/token"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/email"
)

// Module wires the auth domain. Constructors are provided as concrete types and
// bound to the interfaces their consumers declare:
//   - userRepositoryAdapter binds *repository.UserRepository to
//     service.UserRepository, translating repository.ErrUserNotFound into
//     service.ErrUserNotFound (the sentinel the service branches on).
//   - *token.TokenManager satisfies both service.TokenManager (Generate) and
//     interceptor.TokenManager (Validate).
//   - *handler.AuthHandler satisfies handler.AuthService via *service.AuthService.
var Module = fx.Module("auth",
	fx.Provide(
		clock.New,
		provideTokenManager,
		provideServiceTokenManager,
		repository.NewUserRepository,
		provideServiceUserRepository,
		provideEmailSender,
		provideAuthService,
		provideAuthHandler,
		fx.Annotate(
			provideAuthInterceptor,
			fx.ResultTags(`name:"authInterceptor"`),
		),
		fx.Annotate(
			provideProtoValidateInterceptor,
			fx.ResultTags(`name:"protoValidateInterceptor"`),
		),
	),
)

// provideTokenManager builds the JWT token manager from configuration and the
// shared clock.
func provideTokenManager(cfg config.Config, clk clock.Clock) *token.TokenManager {
	return token.NewTokenManager([]byte(cfg.JWTSecret), cfg.JWTTTL, clk)
}

// provideServiceTokenManager binds the concrete *token.TokenManager to the
// service.TokenManager interface its consumer (AuthService) declares. The
// interceptor consumes the concrete type directly, so both shapes are needed.
func provideServiceTokenManager(tm *token.TokenManager) service.TokenManager {
	return tm
}

// provideAuthService constructs the AuthService from its declared interface
// dependencies. The NationalTeamRepository is supplied by the nationalteam
// module at the composition root (T12). bcryptCost 0 selects the production
// default (cost 12) inside NewAuthService.
func provideAuthService(
	users service.UserRepository,
	nationalTeams service.NationalTeamRepository,
	tokens service.TokenManager,
	emailSender service.EmailSender,
	clk clock.Clock,
	logger *zap.Logger,
) (*service.AuthService, error) {
	return service.NewAuthService(users, nationalTeams, tokens, emailSender, clk, logger, 0)
}

// provideEmailSender selects the EmailSender by configuration: an empty
// ResendAPIKey (development — config's fail-fast guarantees production has the
// key) binds the NoopSender that logs the destination and drops the body; a
// present key binds the production ResendSender.
func provideEmailSender(cfg config.Config, logger *zap.Logger) service.EmailSender {
	if cfg.ResendAPIKey == "" {
		return email.NewNoopSender(logger)
	}
	return email.NewResendSender(cfg.ResendAPIKey, cfg.ResendFromEmail, logger)
}

// provideAuthHandler adapts the concrete *service.AuthService to the narrow
// handler.AuthService interface.
func provideAuthHandler(svc *service.AuthService) *handler.AuthHandler {
	return handler.NewAuthHandler(svc)
}

// provideAuthInterceptor builds the JWT auth interceptor. The TokenManager is
// the concrete *token.TokenManager (satisfies interceptor.TokenManager); the
// public-methods list is supplied by the composition root.
func provideAuthInterceptor(tm *token.TokenManager, publicMethods []string) grpc.UnaryServerInterceptor {
	return interceptor.NewAuthJWTUnaryInterceptor(tm, publicMethods)
}

// provideProtoValidateInterceptor builds the protovalidate interceptor from the
// validator supplied by the composition root.
func provideProtoValidateInterceptor(v protovalidate.Validator) grpc.UnaryServerInterceptor {
	return interceptor.NewProtoValidateUnaryInterceptor(v)
}

// provideServiceUserRepository binds the concrete repository to the service's
// interface via the translating adapter.
func provideServiceUserRepository(repo *repository.UserRepository) service.UserRepository {
	return userRepositoryAdapter{repo: repo}
}

// userRepositoryAdapter bridges the concrete repository.UserRepository to the
// service.UserRepository interface. It exists because the two layers differ in
// two ways the service cannot tolerate directly:
//
//  1. Shapes: the repository uses repository.User and CreateUser returns the
//     persisted row; the service uses service.User and CreateUser returns only
//     an error.
//  2. Sentinels: the repository signals "not found" with repository.ErrUserNotFound,
//     but the service branches on service.ErrUserNotFound via errors.Is. Without
//     translation, duplicate-email (Register) and unknown-email (Login) would
//     fall through to codes.Internal instead of AlreadyExists / Unauthenticated.
type userRepositoryAdapter struct {
	repo *repository.UserRepository
}

// CreateUser maps the service.User to repository.User and discards the returned
// row, matching the service.UserRepository signature.
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

// GetUserByEmail maps the repository.User back to service.User and translates
// the repository's not-found sentinel into the service's.
func (a userRepositoryAdapter) GetUserByEmail(ctx context.Context, email string) (service.User, error) {
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
		TempPasswordHash:      u.TempPasswordHash,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt,
		NationalTeamIDs:       u.NationalTeamIDs,
	}, nil
}

// GetUserByID maps the repository.User back to service.User and translates the
// repository's not-found sentinel into the service's.
func (a userRepositoryAdapter) GetUserByID(ctx context.Context, id string) (service.User, error) {
	u, err := a.repo.GetUserByID(ctx, id)
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
		TempPasswordHash:      u.TempPasswordHash,
		TempPasswordExpiresAt: u.TempPasswordExpiresAt,
		NationalTeamIDs:       u.NationalTeamIDs,
	}, nil
}

// SetTempPassword forwards to the concrete repository and translates the
// repository's not-found sentinel into the service's at the single translation
// point.
func (a userRepositoryAdapter) SetTempPassword(ctx context.Context, id, hash string, expiresAt time.Time) error {
	err := a.repo.SetTempPassword(ctx, id, hash, expiresAt)
	if errors.Is(err, repository.ErrUserNotFound) {
		return service.ErrUserNotFound
	}
	return err
}

// UpdatePassword forwards to the concrete repository and translates the
// repository's not-found sentinel into the service's at the single translation
// point.
func (a userRepositoryAdapter) UpdatePassword(ctx context.Context, id, hash string) error {
	err := a.repo.UpdatePassword(ctx, id, hash)
	if errors.Is(err, repository.ErrUserNotFound) {
		return service.ErrUserNotFound
	}
	return err
}

// Compile-time assertions that the wiring binds satisfy their interfaces.
var (
	_ service.UserRepository = userRepositoryAdapter{}
	_ handler.AuthService    = (*service.AuthService)(nil)
)
