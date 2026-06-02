package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authv1 "github.com/rodrigorahman/wc_2026_api/gen/wc2026/auth/v1"
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

// T9 — RequestPasswordRecovery is public (the recovery flow has no JWT yet),
// while ChangePassword and GetMe stay protected: ChangePassword authenticates
// the temporary password through the JWT-protected channel, so leaving it out of
// the allowlist is a security requirement (a literal typo here would fail open).
// Asserted via the generated constants, never string literals.
func TestServer_PublicMethods_RecoveryPublic_ChangePasswordProtected(t *testing.T) {
	publicMethods := server.ProvidePublicMethods()

	require.Contains(t, publicMethods, authv1.AuthService_RequestPasswordRecovery_FullMethodName)
	require.NotContains(t, publicMethods, authv1.AuthService_ChangePassword_FullMethodName)
	require.NotContains(t, publicMethods, authv1.AuthService_GetMe_FullMethodName)
}
