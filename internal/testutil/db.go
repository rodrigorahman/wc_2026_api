// Package testutil provides shared helpers for integration tests. TestNewDB
// gives every test a fully migrated, isolated SQLite database backed by the
// pure-Go modernc driver (no CGO).
package testutil

import (
	"database/sql"
	"path/filepath"
	"testing"

	appdb "github.com/rodrigorahman/wc_2026_api/internal/infra/db"
)

// TestNewDB opens a fresh SQLite database in a per-test temporary directory,
// applies all migrations, and registers cleanup so the connection is closed
// when the test ends. The returned handle has foreign_keys=ON, journal_mode=WAL
// and busy_timeout set (the same PRAGMAs used in production — see db.Open).
//
// A file-backed database (under t.TempDir()) is used rather than ":memory:" so
// that journal_mode=WAL is observable and so the schema survives the single
// connection's lifetime under SetMaxOpenConns(1).
func TestNewDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "test.db")

	database, err := appdb.Open(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	if err := appdb.Migrate(database); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return database
}
