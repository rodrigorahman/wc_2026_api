package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/auth/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// Selection ids seeded by migration 000002. Using real ids guarantees the FK
// constraint is satisfied on the happy path.
const (
	validNationalTeamID  = "a1f3c5e7-0001-4000-8000-000000000001" // Brasil
	validNationalTeamID2 = "a1f3c5e7-0002-4000-8000-000000000002" // Argentina
	validNationalTeamID3 = "a1f3c5e7-0003-4000-8000-000000000003" // França
)

func newRepoDB(t *testing.T) (*repository.UserRepository, *sql.DB) {
	t.Helper()
	db := testutil.TestNewDB(t)
	return repository.NewUserRepository(db, sqlc.New(db)), db
}

func newRepo(t *testing.T) *repository.UserRepository {
	t.Helper()
	repo, _ := newRepoDB(t)
	return repo
}

func baseUser() repository.User {
	return repository.User{
		ID:              uuid.NewString(),
		FullName:        "João Silva",
		Email:           "joao@example.com",
		PasswordHash:    "$2a$12$hashdummy",
		NationalTeamIDs: []string{validNationalTeamID},
	}
}

// CT-018 — UNIQUE constraint on e-mail prevents duplicate insertion.
func TestUserRepository_CreateUser_UniqueEmail(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	first := baseUser()
	_, err := repo.CreateUser(ctx, first)
	require.NoError(t, err, "first insertion should succeed")

	second := baseUser()
	second.ID = uuid.NewString() // different PK
	// same e-mail as first
	_, err = repo.CreateUser(ctx, second)
	require.Error(t, err, "second insertion with duplicate e-mail must fail")
	// Verify it is indeed a UNIQUE violation (SQLite error message contains "UNIQUE")
	require.Contains(t, err.Error(), "UNIQUE", "error should reference UNIQUE constraint")
}

// CT-019 — FK constraint rejects a non-existent national_team_id (foreign_keys=ON).
func TestUserRepository_CreateUser_FK_NonexistentNationalTeam(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	u := baseUser()
	u.NationalTeamIDs = []string{"00000000-0000-0000-0000-000000000000"} // does not exist in national_teams

	_, err := repo.CreateUser(ctx, u)
	require.Error(t, err, "inserting with a non-existent national_team_id must fail")
	// SQLite reports FK violations with "FOREIGN KEY constraint failed"
	require.Contains(t, err.Error(), "FOREIGN KEY", "error should reference FK constraint")
}

// CT-047 — CreateUser persists the user and its selection, retrievable through
// ListUserNationalTeams (real SQLite boundary).
func TestUserRepository_CreateUser_PersistsSelections(t *testing.T) {
	ctx := context.Background()
	repo, db := newRepoDB(t)

	u := baseUser()
	created, err := repo.CreateUser(ctx, u)
	require.NoError(t, err)
	require.Equal(t, u.NationalTeamIDs, created.NationalTeamIDs)

	// Read back independently of the repository to confirm real persistence.
	stored, err := sqlc.New(db).ListUserNationalTeams(ctx, u.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, u.NationalTeamIDs, stored)
}

// CT-048 — CreateUser with three valid selections persists all three rows.
func TestUserRepository_CreateUser_ThreeSelections(t *testing.T) {
	ctx := context.Background()
	repo, db := newRepoDB(t)

	u := baseUser()
	u.NationalTeamIDs = []string{validNationalTeamID, validNationalTeamID2, validNationalTeamID3}
	_, err := repo.CreateUser(ctx, u)
	require.NoError(t, err)

	stored, err := sqlc.New(db).ListUserNationalTeams(ctx, u.ID)
	require.NoError(t, err)
	require.ElementsMatch(t, u.NationalTeamIDs, stored)
}

// CT-049 — GetUserByID populates NationalTeamIDs from user_national_teams.
func TestUserRepository_GetUserByID_PopulatesSelections(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	u := baseUser()
	u.NationalTeamIDs = []string{validNationalTeamID, validNationalTeamID2}
	_, err := repo.CreateUser(ctx, u)
	require.NoError(t, err)

	got, err := repo.GetUserByID(ctx, u.ID)
	require.NoError(t, err)
	require.Equal(t, u.ID, got.ID)
	require.ElementsMatch(t, u.NationalTeamIDs, got.NationalTeamIDs)
}

// CT-050 — CreateUser with a non-existent selection rolls back: neither the user
// row nor any selection row is persisted.
func TestUserRepository_CreateUser_InvalidSelection_Rollback(t *testing.T) {
	ctx := context.Background()
	repo, db := newRepoDB(t)

	u := baseUser()
	u.NationalTeamIDs = []string{"00000000-0000-0000-0000-000000000000"}

	_, err := repo.CreateUser(ctx, u)
	require.Error(t, err)

	// The user row must not exist.
	_, err = repo.GetUserByID(ctx, u.ID)
	require.True(t, errors.Is(err, repository.ErrUserNotFound), "user must not be persisted on rollback")

	// No selection rows for this user.
	stored, err := sqlc.New(db).ListUserNationalTeams(ctx, u.ID)
	require.NoError(t, err)
	require.Empty(t, stored, "no selection rows must survive rollback")
}

// CT-051 — CreateUser with three selections where the third has an invalid FK
// rolls back all three plus the user.
func TestUserRepository_CreateUser_ThirdSelectionInvalid_Rollback(t *testing.T) {
	ctx := context.Background()
	repo, db := newRepoDB(t)

	u := baseUser()
	u.NationalTeamIDs = []string{
		validNationalTeamID,
		validNationalTeamID2,
		"00000000-0000-0000-0000-000000000000", // invalid FK
	}

	_, err := repo.CreateUser(ctx, u)
	require.Error(t, err)

	_, err = repo.GetUserByID(ctx, u.ID)
	require.True(t, errors.Is(err, repository.ErrUserNotFound), "user must not survive rollback")

	stored, err := sqlc.New(db).ListUserNationalTeams(ctx, u.ID)
	require.NoError(t, err)
	require.Empty(t, stored, "no selection rows must survive rollback")
}

// CT-052 — after migration 000004 (applied by TestNewDB), user_national_teams
// exists with a composite primary key and users no longer has national_team_id.
func TestMigration000004_Up_Schema(t *testing.T) {
	db := testutil.TestNewDB(t)
	ctx := context.Background()

	// national_team_id was dropped from users.
	var hasNationalTeamID int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'national_team_id'").
		Scan(&hasNationalTeamID)
	require.NoError(t, err)
	require.Equal(t, 0, hasNationalTeamID, "national_team_id must no longer exist in users")

	// user_national_teams exists.
	var hasTable int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'user_national_teams'").
		Scan(&hasTable)
	require.NoError(t, err)
	require.Equal(t, 1, hasTable, "user_national_teams table must exist")

	// Composite primary key over (user_id, national_team_id): both columns have pk > 0.
	rows, err := db.QueryContext(ctx,
		"SELECT name, pk FROM pragma_table_info('user_national_teams') WHERE pk > 0")
	require.NoError(t, err)
	defer rows.Close()

	pkCols := map[string]int{}
	for rows.Next() {
		var name string
		var pk int
		require.NoError(t, rows.Scan(&name, &pk))
		pkCols[name] = pk
	}
	require.NoError(t, rows.Err())
	require.Len(t, pkCols, 2, "primary key must span exactly two columns")
	require.Contains(t, pkCols, "user_id")
	require.Contains(t, pkCols, "national_team_id")
}

// CT-053 — migration 000004 down reverts to the 1:1 schema: national_team_id returns
// to users and user_national_teams is removed. The migration is exercised
// through the same golang-migrate driver used in production.
func TestMigration000004_Down_RevertsTo1to1(t *testing.T) {
	ctx := context.Background()

	dsn := t.TempDir() + "/down.db"
	migrator, db := testutil.NewMigratorForTest(t, dsn)

	// Migrate fully up, then step down past 000005 (add flag_url) and 000004
	// to assert 000004's down behaviour (the 1:1 schema revert).
	require.NoError(t, migrator.Up())
	require.NoError(t, migrator.Steps(-2))

	// national_team_id is back in users.
	var hasNationalTeamID int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'national_team_id'").
		Scan(&hasNationalTeamID)
	require.NoError(t, err)
	require.Equal(t, 1, hasNationalTeamID, "national_team_id must be restored in users after down")

	// user_national_teams is gone.
	var hasTable int
	err = db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'user_national_teams'").
		Scan(&hasTable)
	require.NoError(t, err)
	require.Equal(t, 0, hasTable, "user_national_teams must be dropped after down")
}

// GetUserByEmail: happy path — returns the user that was just created.
func TestUserRepository_GetUserByEmail_Found(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	u := baseUser()
	created, err := repo.CreateUser(ctx, u)
	require.NoError(t, err)

	got, err := repo.GetUserByEmail(ctx, u.Email)
	require.NoError(t, err)

	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Email, got.Email)
	require.Equal(t, created.FullName, got.FullName)
	require.ElementsMatch(t, created.NationalTeamIDs, got.NationalTeamIDs)
}

// GetUserByEmail: not-found path — returns ErrUserNotFound sentinel.
func TestUserRepository_GetUserByEmail_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	_, err := repo.GetUserByEmail(ctx, "nobody@example.com")
	require.Error(t, err)
	require.True(t, errors.Is(err, repository.ErrUserNotFound), "error must unwrap to ErrUserNotFound")
}
