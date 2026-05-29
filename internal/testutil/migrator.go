package testutil

import (
	"database/sql"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"

	// file source: drives migrations up and down from the on-disk directory.
	_ "github.com/golang-migrate/migrate/v4/source/file"

	appdb "github.com/rodrigorahman/wc_2026_api/internal/infra/db"

	_ "modernc.org/sqlite"
)

// migrationsDir resolves the production migrations directory relative to this
// source file, so the helper works regardless of the test's package directory.
func migrationsDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "infra", "db", "migrations")
}

// NewMigratorForTest opens a fresh SQLite database at dsn (pure-Go modernc
// driver, production PRAGMAs) and returns a golang-migrate migrator over the
// production migrations together with the open handle. It drives the same
// migrations both up and down through the same modernc (no CGO) stack used in
// production. TestNewDB only applies them up; this helper exposes a full
// migrator for the down-revert test (CT-053). Cleanup is registered.
func NewMigratorForTest(t *testing.T, dsn string) (*migrate.Migrate, *sql.DB) {
	t.Helper()

	database, err := appdb.Open(dsn)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	driver, err := migratesqlite.WithInstance(database, &migratesqlite.Config{})
	if err != nil {
		t.Fatalf("create migrate driver: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://"+migrationsDir(), "sqlite", driver)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	t.Cleanup(func() { _, _ = migrator.Close() })

	return migrator, database
}
