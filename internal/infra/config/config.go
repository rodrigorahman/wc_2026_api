// Package config implements application configuration loading via viper.
// It enforces a fail-fast security policy: JWT_SECRET must be present and
// contain at least 32 bytes before the server is allowed to start.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// dbFileName is the SQLite file created next to the executable when DB_PATH is
// not provided, so a natively launched binary persists its database in its own
// directory regardless of the current working directory.
const dbFileName = "wc2026.db"

// Config holds all runtime configuration values for the application.
type Config struct {
	// DBPath is the file-system path to the SQLite database file.
	DBPath string

	// JWTSecret is the HMAC-SHA256 signing key for JWT tokens.
	// Must be at least 32 bytes.
	JWTSecret string

	// JWTTTL is how long a newly issued JWT token remains valid.
	// Defaults to 1h when not set.
	JWTTTL time.Duration

	// GRPCPort is the TCP port the gRPC server listens on.
	// Defaults to "50051" when not set.
	GRPCPort string

	// Env is the deployment environment ("development" / "production").
	// Defaults to "production" so that introspection surfaces (e.g. gRPC
	// reflection) stay off unless a dev environment is explicitly declared.
	Env string

	// ResendAPIKey is the secret API key used to send transactional e-mails
	// via Resend. Required outside development; empty in development selects
	// the NoopSender at wiring time.
	ResendAPIKey string

	// ResendFromEmail is the sender address used for Resend e-mails.
	// Required whenever ResendAPIKey is set — without it there is no sender.
	ResendFromEmail string
}

// IsDevelopment reports whether the application is running in a development
// environment. Development-only surfaces (gRPC reflection) gate on this.
func (c Config) IsDevelopment() bool {
	return c.Env == "dev" || c.Env == "development"
}

// Load reads configuration from environment variables (and an optional config
// file) via viper, applies sensible defaults, and enforces the fail-fast
// security rule for JWT_SECRET.
//
// It returns an error — never calls os.Exit or log.Fatal — so that the caller
// (composition root, T12) decides how to abort the fx OnStart lifecycle.
func Load() (Config, error) {
	v := viper.New()

	v.SetDefault("JWT_TTL", "1h")
	v.SetDefault("GRPC_PORT", "50051")
	v.SetDefault("APP_ENV", "production")

	v.AutomaticEnv()

	secret := v.GetString("JWT_SECRET")
	if len(secret) < 32 {
		return Config{}, fmt.Errorf(
			"JWT_SECRET inválido: deve ter no mínimo 32 bytes (atual: %d bytes) — configure a variável de ambiente JWT_SECRET",
			len(secret),
		)
	}

	ttlRaw := v.GetString("JWT_TTL")
	ttl, err := time.ParseDuration(ttlRaw)
	if err != nil {
		return Config{}, fmt.Errorf("JWT_TTL inválido %q: %w", ttlRaw, err)
	}

	dbPath := v.GetString("DB_PATH")
	if dbPath == "" {
		dbPath, err = defaultDBPath()
		if err != nil {
			return Config{}, err
		}
	}

	env := v.GetString("APP_ENV")
	resendAPIKey := v.GetString("RESEND_API_KEY")
	resendFromEmail := v.GetString("RESEND_FROM_EMAIL")

	isDevelopment := env == "dev" || env == "development"
	if !isDevelopment && resendAPIKey == "" {
		return Config{}, fmt.Errorf(
			"RESEND_API_KEY ausente: obrigatório fora de desenvolvimento — configure a variável de ambiente RESEND_API_KEY",
		)
	}
	if resendAPIKey != "" && resendFromEmail == "" {
		return Config{}, fmt.Errorf(
			"RESEND_FROM_EMAIL ausente: obrigatório quando RESEND_API_KEY está presente — configure a variável de ambiente RESEND_FROM_EMAIL",
		)
	}

	return Config{
		DBPath:          dbPath,
		JWTSecret:       secret,
		JWTTTL:          ttl,
		GRPCPort:        v.GetString("GRPC_PORT"),
		Env:             env,
		ResendAPIKey:    resendAPIKey,
		ResendFromEmail: resendFromEmail,
	}, nil
}

// defaultDBPath resolves the SQLite file location next to the running
// executable. The executable's directory always exists, so SQLite can create
// the file there on first run without any directory setup.
func defaultDBPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolver caminho do executável para DB_PATH padrão: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), dbFileName), nil
}
