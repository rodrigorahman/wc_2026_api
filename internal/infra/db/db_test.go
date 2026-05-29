package db_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	appdb "github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// TestOpenDB_PostMigrateState groups the invariants that must hold for any
// connection returned after Open + Migrate. It reuses TestNewDB, which opens a
// file-backed SQLite database (required for journal_mode=WAL to be observable)
// and applies every migration.
//
// INVARIANT: a migrated connection always has foreign_keys=ON and
// journal_mode=WAL, exposes the users and national_teams tables, and the seed
// migration leaves national_teams non-empty.
// OWNING_LAYER: service-integration (real SQLite, real migrations).
func TestOpenDB_PostMigrateState(t *testing.T) {
	database := testutil.TestNewDB(t)

	// CT-F06: foreign_keys must be ON, otherwise the national_team_id FK is not
	// enforced and CT-019 (T8) would pass silently.
	t.Run("ForeignKeysEnabled", func(t *testing.T) {
		var foreignKeys int
		err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
		require.NoError(t, err)
		require.Equal(t, 1, foreignKeys, "PRAGMA foreign_keys must be ON")
	})

	// CT-F07: journal_mode must be WAL. This requires a file-backed database;
	// :memory: would report "memory" instead.
	t.Run("JournalModeWAL", func(t *testing.T) {
		var journalMode string
		err := database.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
		require.NoError(t, err)
		require.Equal(t, "wal", strings.ToLower(journalMode))
	})

	// CT-F08: both domain tables must exist after migrations.
	t.Run("TablesExist", func(t *testing.T) {
		for _, table := range []string{"users", "national_teams"} {
			var name string
			err := database.QueryRow(
				"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
				table,
			).Scan(&name)
			require.NoErrorf(t, err, "table %q must exist in sqlite_master", table)
			require.Equal(t, table, name)
		}
	})

	// CT-F09: the seed migration (000002) must populate national_teams.
	t.Run("SeedApplied", func(t *testing.T) {
		var count int
		err := database.QueryRow("SELECT COUNT(*) FROM national_teams").Scan(&count)
		require.NoError(t, err)
		require.Greater(t, count, 0, "seed migration must populate national_teams")
	})
}

// TestQuery_NoMigrations_Error is the negative companion to the post-migrate
// suite: an un-migrated database must fail explicitly when queried, never
// return an empty result silently (CT-021 / CA-10).
//
// INVARIANT: querying national_teams before migrations run fails with a "no such
// table" error rather than succeeding with zero rows.
// OWNING_LAYER: service-integration (real SQLite, no migrations applied).
func TestQuery_NoMigrations_Error(t *testing.T) {
	dsn := t.TempDir() + "/no_migrations.db"

	database, err := appdb.Open(dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM national_teams").Scan(&count)

	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "no such table")
}
