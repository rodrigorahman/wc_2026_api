// Command server is the composition root of the WC 2026 API. It builds the
// uber-fx application graph from the per-domain modules (db, auth, nationalteam)
// plus the shared server providers, opens a TCP listener on the configured port
// and serves gRPC. Config is validated fail-fast (an invalid JWT_SECRET aborts
// before serving); shutdown is graceful and closes the database.
package main

import (
	"context"
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam"
	"github.com/rodrigorahman/wc_2026_api/internal/server"
)

func main() {
	fx.New(
		db.Module,
		auth.Module,
		nationalteam.Module,
		server.Providers,
		fx.Provide(server.ConfigProvider),
		fx.Provide(provideListener),
		fx.Invoke(runGRPCServer),
	).Run()
}

// provideListener opens the TCP listener on the configured gRPC port. Opening it
// here (rather than inside the OnStart hook) lets fx report a bind failure as a
// graph error before any background goroutine starts.
func provideListener(cfg config.Config) (net.Listener, error) {
	return net.Listen("tcp", ":"+cfg.GRPCPort)
}

// runGRPCServer registers the serve/shutdown lifecycle. OnStart serves in a
// background goroutine (Serve blocks); OnStop stops gracefully. Migrations and
// DB close are owned by db.Module's own hooks.
func runGRPCServer(lc fx.Lifecycle, srv *grpc.Server, lis net.Listener, logger *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				logger.Info("grpc server listening", zap.String("addr", lis.Addr().String()))
				if err := srv.Serve(lis); err != nil {
					logger.Error("grpc server stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})
}
