// Package db provides the SQLite persistence layer: it opens a pure-Go
// (modernc.org/sqlite) connection with the mandatory PRAGMAs, applies the
// embedded golang-migrate migrations on boot, and exposes the sqlc Queries
// through an fx.Module. No CGO driver is used (see ADR-0001).
package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	// modernc.org/sqlite registers the pure-Go "sqlite" driver via its init.
	_ "modernc.org/sqlite"
)

// migrationsFS embeds the versioned migrations so the binary is fully portable
// (no external SQL files needed at runtime).
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens a SQLite database at dsn using the pure-Go modernc driver and
// applies the mandatory connection settings (§7.1):
//   - PRAGMA foreign_keys = ON   (enforces the national_team_id FK)
//   - PRAGMA journal_mode = WAL  (concurrent reads alongside serialized writes)
//   - PRAGMA busy_timeout = 5000 (wait on locks instead of failing immediately)
//   - SetMaxOpenConns(1)         (single-writer model — §7.4)
//
// It does not apply migrations; call Migrate for that on the returned handle.
func Open(dsn string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	database.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if _, err := database.Exec(pragma); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("apply %q: %w", pragma, err)
		}
	}

	return database, nil
}

// Migrate applies all embedded migrations over the already-open *sql.DB,
// reusing the modernc connection and its PRAGMAs (§7.3). It uses the pure-Go
// golang-migrate "sqlite" driver — never the CGO "sqlite3" driver (ADR-0001).
func Migrate(database *sql.DB) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	driver, err := migratesqlite.WithInstance(database, &migratesqlite.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", source, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
