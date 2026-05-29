package match_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"google.golang.org/grpc"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match"
	matchhandler "github.com/rodrigorahman/wc_2026_api/internal/domain/match/handler"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/server"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

const moduleTestJWTSecret = "module-test-secret-at-least-32-bytes!!"

// T6-CT-001 — TestModule_ComposesInGraph: match.Module resolve no grafo fx
// (sem dependência ausente ou colisão de tipo) com DB real e clock.
// fx.ValidateApp resolve o grafo sem executar o lifecycle.
func TestModule_ComposesInGraph(t *testing.T) {
	cfg := config.Config{
		DBPath:    t.TempDir() + "/smoke.db",
		JWTSecret: moduleTestJWTSecret,
		JWTTTL:    time.Hour,
		GRPCPort:  "0",
	}

	err := fx.ValidateApp(
		db.Module,
		auth.Module,
		nationalteam.Module,
		match.Module,
		server.Providers,
		fx.Supply(cfg),
		fx.Invoke(func(*matchhandler.MatchHandler) {}),
	)
	require.NoError(t, err)
}

// T6-CT-002 — TestModule_AdapterSatisfiesServiceRepository: o adapter
// provideServiceRepository satisfaz service.MatchRepository e delega
// corretamente ao repositório concreto (retorna lista vazia sem erro para DB
// limpo).
func TestModule_AdapterSatisfiesServiceRepository(t *testing.T) {
	ctx := context.Background()

	rawDB := testutil.TestNewDB(t)
	repo := repository.NewMatchRepository(sqlc.New(rawDB))
	adapter := match.ProvideServiceRepository(repo)

	cutoff := time.Now()
	got, err := adapter.ListUpcomingMatchesByUser(ctx, "00000000-0000-0000-0000-000000000000", cutoff)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got, 0)
}

// T6-CT-004 — TestModule_ConsumesClockNoCollision: match.Module consome
// clock.Clock do grafo compartilhado (provido por auth.Module) sem re-prover
// clock.New, evitando colisão de tipo no fx.
func TestModule_ConsumesClockNoCollision(t *testing.T) {
	cfg := config.Config{
		DBPath:    t.TempDir() + "/smoke2.db",
		JWTSecret: moduleTestJWTSecret,
		JWTTTL:    time.Hour,
		GRPCPort:  "0",
	}

	err := fx.ValidateApp(
		db.Module,
		auth.Module,
		nationalteam.Module,
		match.Module,
		server.Providers,
		fx.Supply(cfg),
		fx.Invoke(func(*grpc.Server) {}),
		fx.Invoke(func(*matchhandler.MatchHandler) {}),
	)
	require.NoError(t, err)
}
