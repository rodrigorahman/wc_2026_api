package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	matchv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/match/v1"
	"github.com/rodrigorahman/wc_2026_api/internal/server"
)

// T6-CT-003 — TestServer_ListUpcomingMatches_NotPublic: ListUpcomingMatches
// NÃO deve constar na lista de métodos públicos (allowlist do interceptor JWT).
// Usa a constante gerada (nunca literal) para evitar typos que abrissem o RPC
// silenciosamente (fail-open).
func TestServer_ListUpcomingMatches_NotPublic(t *testing.T) {
	publicMethods := server.ProvidePublicMethods()

	require.NotContains(t, publicMethods, matchv1.MatchService_ListUpcomingMatches_FullMethodName)
}
