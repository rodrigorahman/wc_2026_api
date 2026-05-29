// bufconn.go provides TestNewBufconnServer: an in-process gRPC server wired with
// the *real* production interceptor chain (recovery → logging → protovalidate →
// auth JWT) and the real db/auth/nationalteam fx modules, reachable over an
// in-memory bufconn listener. E2E tests (CT-027..032) dial the returned
// connection and exercise the full stack black-box.
package testutil

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/clock"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam"
	"github.com/rodrigorahman/wc_2026_api/internal/server"
)

// bufconnBufSize is the in-memory listener buffer size.
const bufconnBufSize = 1024 * 1024

// testJWTSecret is a 32+ byte HMAC secret satisfying config's fail-fast rule.
const testJWTSecret = "test-secret-at-least-32-bytes-long!!"

// FixedClock is a deterministic Clock for bufconn E2E tests. Advance moves time
// forward, e.g. past a token's expiration (CT-031).
type FixedClock struct {
	now time.Time
}

// NewFixedClock returns a FixedClock anchored at now.
func NewFixedClock(now time.Time) *FixedClock {
	return &FixedClock{now: now}
}

// Now returns the current (frozen) time.
func (c *FixedClock) Now() time.Time { return c.now }

// Advance moves the clock forward by d.
func (c *FixedClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// TestNewBufconnServer starts the full gRPC server (real interceptor chain, real
// modules) over an in-memory bufconn listener and returns a connected
// *grpc.ClientConn. The optional clk overrides the system clock so tests can
// control token issuance/expiration; pass nil to use the system clock.
//
// The server is built around a fresh migrated SQLite database (seed applied) so
// every test is isolated. Cleanup (stop server, close conn) is registered via
// t.Cleanup.
func TestNewBufconnServer(t *testing.T, clk clock.Clock) *grpc.ClientConn {
	t.Helper()

	lis := bufconn.Listen(bufconnBufSize)
	dbPath := filepathTemp(t)

	testCfg := config.Config{
		DBPath:    dbPath,
		JWTSecret: testJWTSecret,
		JWTTTL:    time.Hour,
		GRPCPort:  "0",
	}

	opts := []fx.Option{
		db.Module,
		auth.Module,
		nationalteam.Module,
		server.Providers,
		// Supply a fixed config (no env / viper needed) and the bufconn listener.
		fx.Supply(testCfg),
		fx.Supply(fx.Annotate(lis, fx.As(new(net.Listener)))),
		fx.Invoke(serveBufconn),
		fx.NopLogger,
	}
	if clk != nil {
		opts = append(opts, fx.Decorate(func(clock.Clock) clock.Clock { return clk }))
	}

	app := fx.New(opts...)

	startCtx, cancelStart := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStart()
	if err := app.Start(startCtx); err != nil {
		t.Fatalf("start bufconn server: %v", err)
	}

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		stopCtx, cancelStop := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelStop()
		_ = app.Stop(stopCtx)
	})

	return conn
}

// serveBufconn registers the lifecycle that serves the gRPC server on the
// bufconn listener and stops it gracefully. The DB lifecycle (migrations,
// close) is owned by db.Module.
func serveBufconn(lc fx.Lifecycle, srv *grpc.Server, lis net.Listener) {
	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() { _ = srv.Serve(lis) }()
			return nil
		},
		OnStop: func(context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})
}

// filepathTemp returns a fresh SQLite file path under the test's temp dir.
func filepathTemp(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/e2e.db"
}
