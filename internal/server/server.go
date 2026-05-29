// Package server is the composition glue shared by the production binary
// (cmd/server) and the E2E bufconn helper (internal/testutil). It assembles the
// gRPC server with the security-critical interceptor chain in the mandated order
// (recovery → logging → protovalidate → auth JWT — Tech Spec §3.1) and binds the
// cross-domain inputs the per-domain fx modules declare but do not own.
//
// It exists as its own package — rather than living inside cmd/server (package
// main, which is not importable) — so that the E2E tests exercise exactly the
// same chain and wiring as production. Duplicating the chain into the test
// helper would let the tests pass against a different chain than the one shipped.
package server

import (
	"context"
	"errors"

	"buf.build/go/protovalidate"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"

	authhandler "github.com/rodrigorahman/wc_2026_api/internal/domain/auth/handler"
	authservice "github.com/rodrigorahman/wc_2026_api/internal/domain/auth/service"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	applogger "github.com/rodrigorahman/wc_2026_api/internal/infra/logger"
	nthandler "github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/handler"
	ntrepo "github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
	nationalteamv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/nationalteam/v1"
)

// Providers is the set of cross-domain fx providers owned by the composition
// root: logger, protovalidate validator, the public-methods list, the
// NationalTeam repository bridge consumed by the AuthService, and the assembled
// *grpc.Server. It is combined with db.Module, auth.Module, nationalteam.Module
// and a config.Config provider by both the binary and the bufconn helper.
//
// config.Config is intentionally NOT provided here: the binary loads it via
// ConfigProvider (fail-fast), while the bufconn helper supplies a fixed config.
var Providers = fx.Options(
	fx.Provide(
		provideLogger,
		provideValidator,
		providePublicMethods,
		provideNationalTeamBridge,
		fx.Annotate(
			newGRPCServer,
			fx.ParamTags(`name:"protoValidateInterceptor"`, `name:"authInterceptor"`),
		),
	),
)

// ConfigProvider provides the application configuration loaded via viper. It
// propagates the fail-fast JWT_SECRET error (Tech Spec §11.6): an invalid secret
// aborts the fx graph construction before any lifecycle hook runs. It is a
// standalone provider (not part of Providers) so tests can substitute a fixed
// config without invoking viper.
func ConfigProvider() (config.Config, error) {
	return config.Load()
}

// provideLogger builds the shared structured logger at info level.
func provideLogger() (*zap.Logger, error) {
	return applogger.New(zapcore.InfoLevel)
}

// provideValidator constructs the protovalidate validator consumed by the
// protovalidate interceptor declared in the auth module.
func provideValidator() (protovalidate.Validator, error) {
	return protovalidate.New()
}

// providePublicMethods returns the RPCs that bypass JWT authentication. The
// values are the generated FullMethodName constants (never string literals) so a
// typo cannot silently fail open. GetMe is intentionally absent — it is the
// protected reference RPC.
func providePublicMethods() []string {
	return []string{
		authv1.AuthService_Register_FullMethodName,
		authv1.AuthService_Login_FullMethodName,
		nationalteamv1.NationalTeamService_ListNationalTeams_FullMethodName,
	}
}

// nationalTeamBridge adapts the nationalteam domain's concrete repository to the
// auth service's NationalTeamRepository interface. The two layers use distinct
// not-found sentinels; this bridge translates ntrepo.ErrNationalTeamNotFound
// into authservice.ErrNationalTeamNotFound so that an unknown selection on
// Register surfaces as codes.InvalidArgument rather than codes.Internal.
type nationalTeamBridge struct {
	repo *ntrepo.NationalTeamRepository
}

func (b nationalTeamBridge) GetNationalTeamByID(ctx context.Context, id string) (string, error) {
	name, err := b.repo.GetNationalTeamByID(ctx, id)
	if err != nil {
		if errors.Is(err, ntrepo.ErrNationalTeamNotFound) {
			return "", authservice.ErrNationalTeamNotFound
		}
		return "", err
	}
	return name, nil
}

// provideNationalTeamBridge binds the nationalteam concrete repository to the
// auth service's interface.
func provideNationalTeamBridge(repo *ntrepo.NationalTeamRepository) authservice.NationalTeamRepository {
	return nationalTeamBridge{repo: repo}
}

// newGRPCServer assembles the gRPC server with the unary interceptor chain in
// the mandated order — recovery → logging → protovalidate → auth JWT — and
// registers both domain servers. The protovalidate and auth interceptors are
// injected by name from the auth module; the logging interceptor is built here
// from the shared logger; recovery converts a panic into codes.Internal. Server
// reflection is registered only in development (cfg.IsDevelopment): generic gRPC
// tooling can introspect the API without the .proto files, while production
// keeps the API surface unexposed.
func newGRPCServer(
	protoValidate grpc.UnaryServerInterceptor,
	authJWT grpc.UnaryServerInterceptor,
	logger *zap.Logger,
	authSrv *authhandler.AuthHandler,
	ntSrv *nthandler.NationalTeamHandler,
	cfg config.Config,
) *grpc.Server {
	recoveryInterceptor := recovery.UnaryServerInterceptor(
		recovery.WithRecoveryHandler(func(any) error {
			return status.Error(codes.Internal, "erro interno")
		}),
	)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor,
			applogger.UnaryServerInterceptor(logger),
			protoValidate,
			authJWT,
		),
	)

	authv1.RegisterAuthServiceServer(srv, authSrv)
	nationalteamv1.RegisterNationalTeamServiceServer(srv, ntSrv)

	if cfg.IsDevelopment() {
		reflection.Register(srv)
		logger.Info("grpc server reflection enabled", zap.String("app_env", cfg.Env))
	}

	return srv
}
