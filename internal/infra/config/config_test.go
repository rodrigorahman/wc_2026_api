package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/stretchr/testify/require"
)

// secret32 is the minimum-length valid secret used across tests.
const secret32 = "12345678901234567890123456789012" // exactly 32 bytes

// TestLoad_RejectsMissingSecret verifies that Load returns an error when
// JWT_SECRET is not present in the environment (CT-F01).
func TestLoad_RejectsMissingSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DB_PATH", ":memory:")

	_, err := config.Load()

	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "JWT_SECRET"), "error message should reference JWT_SECRET")
}

// TestLoad_RejectsShortSecret_TableDriven covers CT-F01, CT-F02, and CT-F03
// using a table-driven approach for secrets that are too short (< 32 bytes).
func TestLoad_RejectsShortSecret_TableDriven(t *testing.T) {
	cases := []struct {
		name   string
		secret string
	}{
		{"empty (CT-F02)", ""},
		{"31 bytes (CT-F03)", strings.Repeat("x", 31)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tc.secret)
			t.Setenv("DB_PATH", ":memory:")

			_, err := config.Load()

			require.Error(t, err, "expected error for secret %q", tc.secret)
		})
	}
}

// TestLoad_RejectsShortSecret_Cites32Bytes verifies that the error message
// for a 31-byte secret explicitly cites the 32-byte minimum requirement (CT-F03).
func TestLoad_RejectsShortSecret_Cites32Bytes(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("x", 31))
	t.Setenv("DB_PATH", ":memory:")

	_, err := config.Load()

	require.Error(t, err)
	require.Contains(t, err.Error(), "32", "error message should cite the 32-byte minimum")
}

// TestLoad_AcceptsMinSecret_AppliesDefaults verifies that a 32-byte secret is
// accepted and that JWT_TTL / GRPC_PORT defaults are applied when not set (CT-F04).
func TestLoad_AcceptsMinSecret_AppliesDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "development") // dev: no Resend secret required
	// Deliberately NOT setting JWT_TTL or GRPC_PORT — defaults must kick in.

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Greater(t, cfg.JWTTTL.Seconds(), float64(0), "JWTTTL default must be > 0")
	require.NotEmpty(t, cfg.GRPCPort, "GRPCPort default must not be empty")
}

// TestLoad_PreservesLongSecret verifies that a 64-byte secret is accepted and
// is not truncated silently (CT-F05).
func TestLoad_PreservesLongSecret(t *testing.T) {
	secret64 := strings.Repeat("a", 64)
	t.Setenv("JWT_SECRET", secret64)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "development") // dev: no Resend secret required

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, 64, len(cfg.JWTSecret), "64-byte secret must not be truncated")
}

// TestLoad_DefaultsEnvToProduction verifies that APP_ENV defaults to production
// (reflection-off by default) when not set, so IsDevelopment is false.
func TestLoad_DefaultsEnvToProduction(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("RESEND_API_KEY", "re_test_key") // required in production default
	t.Setenv("RESEND_FROM_EMAIL", "noreply@example.com")
	// Deliberately NOT setting APP_ENV — default must be production.

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "production", cfg.Env)
	require.False(t, cfg.IsDevelopment(), "default env must not be development")
}

// TestLoad_ReadsDevelopmentEnv verifies that APP_ENV=development is read and
// flips IsDevelopment to true (gates gRPC reflection).
func TestLoad_ReadsDevelopmentEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "development")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.True(t, cfg.IsDevelopment(), "APP_ENV=development must enable development mode")
}

// TestLoad_DefaultsDBPathNextToExecutable verifies that when DB_PATH is unset,
// the database path defaults to a file in the running executable's directory,
// so a natively launched binary persists its database next to itself.
func TestLoad_DefaultsDBPathNextToExecutable(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", "")            // force the default to kick in regardless of ambient env
	t.Setenv("APP_ENV", "development") // dev: no Resend secret required

	cfg, err := config.Load()

	require.NoError(t, err)

	exe, err := os.Executable()
	require.NoError(t, err)
	require.Equal(t, filepath.Dir(exe), filepath.Dir(cfg.DBPath), "DB must default to the executable's directory")
	require.Equal(t, "wc2026.db", filepath.Base(cfg.DBPath))
}

// TestLoad_RespectsExplicitDBPath verifies that an explicit DB_PATH overrides
// the executable-directory default.
func TestLoad_RespectsExplicitDBPath(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", "/tmp/custom.db")
	t.Setenv("APP_ENV", "development") // dev: no Resend secret required

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "/tmp/custom.db", cfg.DBPath)
}

// TestIsDevelopment_TableDriven covers the accepted dev aliases and the
// production/empty cases that must keep reflection off.
func TestIsDevelopment_TableDriven(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{"development", true},
		{"dev", true},
		{"production", false},
		{"prod", false},
		{"", false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.env, func(t *testing.T) {
			require.Equal(t, tc.want, config.Config{Env: tc.env}.IsDevelopment())
		})
	}
}

// TestLoad_ResendPolicy_TableDriven consolidates CT-T4-001..005 + CT-T4-007:
// the APP_ENV × RESEND_API_KEY × RESEND_FROM_EMAIL matrix. Mirrors the
// JWT_SECRET fail-fast style: outside development the key is mandatory, and the
// from address is mandatory whenever the key is present (any environment).
func TestLoad_ResendPolicy_TableDriven(t *testing.T) {
	cases := []struct {
		name        string
		env         string
		apiKey      string
		fromEmail   string
		wantErr     bool
		errContains string
	}{
		{"prod sem key (CT-T4-001)", "production", "", "", true, "RESEND_API_KEY"},
		{"dev sem key (CT-T4-002)", "development", "", "", false, ""},
		{"prod key sem from (CT-T4-004)", "production", "re_key", "", true, "RESEND_FROM_EMAIL"},
		{"dev key sem from (CT-T4-005)", "development", "re_key", "", true, "RESEND_FROM_EMAIL"},
		{"prod key + from", "production", "re_key", "noreply@example.com", false, ""},
		{"dev key + from", "development", "re_key", "noreply@example.com", false, ""},
		{"prod default (sem APP_ENV explícito) com key+from", "production", "re_key", "noreply@example.com", false, ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", secret32)
			t.Setenv("DB_PATH", ":memory:")
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("RESEND_API_KEY", tc.apiKey)
			t.Setenv("RESEND_FROM_EMAIL", tc.fromEmail)

			_, err := config.Load()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestLoad_ProdComKey_CarregaValores verifies that both Resend fields are
// loaded verbatim in production when present (CT-T4-003).
func TestLoad_ProdComKey_CarregaValores(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "production")
	t.Setenv("RESEND_API_KEY", "re_prod_key_001")
	t.Setenv("RESEND_FROM_EMAIL", "prod@example.com")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "re_prod_key_001", cfg.ResendAPIKey)
	require.Equal(t, "prod@example.com", cfg.ResendFromEmail)
}

// TestLoad_DevComKey_CarregaValores verifies that both Resend fields are loaded
// with the dev alias APP_ENV=dev (CT-T4-009).
func TestLoad_DevComKey_CarregaValores(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("RESEND_API_KEY", "re_dev_key_009")
	t.Setenv("RESEND_FROM_EMAIL", "dev@example.com")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "re_dev_key_009", cfg.ResendAPIKey)
	require.Equal(t, "dev@example.com", cfg.ResendFromEmail)
}

// TestLoad_CamposResendPresentesNaStruct verifies the structural contract: in
// dev without a key, both new fields are accessible strings and empty (CT-T4-008).
func TestLoad_CamposResendPresentesNaStruct(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "development")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, "", cfg.ResendAPIKey)
	require.Equal(t, "", cfg.ResendFromEmail)
}

// TestLoad_SemResend_SemRegressao verifies that the existing fields are
// unchanged and the new Resend fields come empty in a minimal dev config
// (CT-T4-010).
func TestLoad_SemResend_SemRegressao(t *testing.T) {
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "development")

	cfg, err := config.Load()

	require.NoError(t, err)
	require.Equal(t, secret32, cfg.JWTSecret)
	require.Greater(t, cfg.JWTTTL.Seconds(), float64(0))
	require.NotEmpty(t, cfg.GRPCPort)
	require.Equal(t, "development", cfg.Env)
	require.Equal(t, "", cfg.ResendAPIKey)
	require.Equal(t, "", cfg.ResendFromEmail)
}

// TestLoad_SecretNuncaNoErro verifies that the RESEND_API_KEY value never
// leaks into the error message — the missing-from error must reference only the
// variable name, never the secret value (CT-T4-006).
func TestLoad_SecretNuncaNoErro(t *testing.T) {
	const traceableKey = "re_super_secret_traceable_value_006"
	t.Setenv("JWT_SECRET", secret32)
	t.Setenv("DB_PATH", ":memory:")
	t.Setenv("APP_ENV", "production")
	t.Setenv("RESEND_API_KEY", traceableKey)
	t.Setenv("RESEND_FROM_EMAIL", "") // triggers the from-missing error

	_, err := config.Load()

	require.Error(t, err)
	require.NotContains(t, err.Error(), traceableKey, "secret value must never appear in error output")
}
