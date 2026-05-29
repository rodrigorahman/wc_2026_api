package repository_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/nationalteam/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// Seed IDs from migration 000002_seed_national_teams.up.sql; flag URL backfilled by
// migration 000005_add_flag_url_to_national_teams.up.sql.
const (
	seedBrasilID      = "a1f3c5e7-0001-4000-8000-000000000001"
	seedBrasilName    = "Brasil"
	seedBrasilFlagURL = "https://flagcdn.com/w320/br.png"

	seedNationalTeamCount = 16
)

func newRepo(t *testing.T) *repository.NationalTeamRepository {
	t.Helper()
	db := testutil.TestNewDB(t)
	return repository.NewNationalTeamRepository(sqlc.New(db))
}

// CT-020: ListNationalTeams retorna seleções do seed — lista não vazia com
// seleções do seed.
func TestIntegration_SeedNationalTeams_List(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	teams, err := repo.ListNationalTeams(ctx)

	require.NoError(t, err)
	require.NotEmpty(t, teams, "seed must provide at least one national team")

	// Verify at least Brasil is present (from seed data).
	found := false
	for _, tm := range teams {
		if tm.ID == seedBrasilID {
			require.Equal(t, seedBrasilName, tm.Name)
			require.Equal(t, seedBrasilFlagURL, tm.FlagURL)
			found = true
			break
		}
	}
	require.True(t, found, "seed national team Brasil must be present in the list")
}

// CT-001: a coluna flag_url existe em national_teams com NOT NULL após as
// migrations (PRAGMA table_info).
func TestIntegration_FlagURLColumn_ExistsNotNull(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)

	var name string
	var notNull int
	err := db.QueryRowContext(ctx,
		"SELECT name, \"notnull\" FROM pragma_table_info('national_teams') WHERE name = 'flag_url'").
		Scan(&name, &notNull)

	require.NoError(t, err, "flag_url column must exist in national_teams")
	require.Equal(t, "flag_url", name)
	require.Equal(t, 1, notNull, "flag_url must be NOT NULL")
}

// CT-002: a down migration 000005 remove a coluna flag_url (DROP COLUMN
// suportado pelo driver modernc), exercitada pelo mesmo migrator de produção.
func TestIntegration_FlagURLColumn_DownRemovesColumn(t *testing.T) {
	ctx := context.Background()

	dsn := filepath.Join(t.TempDir(), "down.db")
	migrator, db := testutil.NewMigratorForTest(t, dsn)

	require.NoError(t, migrator.Up())

	var beforeDown int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('national_teams') WHERE name = 'flag_url'").
		Scan(&beforeDown))
	require.Equal(t, 1, beforeDown, "flag_url must exist before down")

	require.NoError(t, migrator.Steps(-1))

	var afterDown int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info('national_teams') WHERE name = 'flag_url'").
		Scan(&afterDown))
	require.Equal(t, 0, afterDown, "flag_url must be dropped after down")
}

// CT-003: as 16 seleções do seed têm flag_url no formato
// https://flagcdn.com/w320/{codigo}.png, conferindo os códigos sensíveis
// (Brasil=br, Inglaterra=gb-eng, Coreia do Sul=kr).
func TestIntegration_Backfill_FlagURLsBySeedID(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	teams, err := repo.ListNationalTeams(ctx)
	require.NoError(t, err)
	require.Len(t, teams, seedNationalTeamCount)

	byID := make(map[string]string, len(teams))
	for _, tm := range teams {
		byID[tm.ID] = tm.FlagURL
	}

	require.Equal(t, "https://flagcdn.com/w320/br.png", byID["a1f3c5e7-0001-4000-8000-000000000001"], "Brasil")
	require.Equal(t, "https://flagcdn.com/w320/gb-eng.png", byID["a1f3c5e7-0006-4000-8000-000000000006"], "Inglaterra")
	require.Equal(t, "https://flagcdn.com/w320/kr.png", byID["a1f3c5e7-0016-4000-8000-000000000016"], "Coreia do Sul")
}

// CT-004: companion negativo de CT-003 — nenhuma seleção fica com flag_url
// vazia após o backfill.
func TestIntegration_Backfill_NoEmptyFlagURL(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)

	var emptyCount int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM national_teams WHERE flag_url = ''").
		Scan(&emptyCount))
	require.Equal(t, 0, emptyCount, "no selection may have an empty flag_url after backfill")
}

// CT-005: ListNationalTeams retorna FlagURL preenchida para cada seleção.
func TestIntegration_ListNationalTeams_FlagURLPopulated(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	teams, err := repo.ListNationalTeams(ctx)
	require.NoError(t, err)
	require.Len(t, teams, seedNationalTeamCount)

	for _, tm := range teams {
		require.NotEmpty(t, tm.FlagURL, "FlagURL must be populated for %s (%s)", tm.Name, tm.ID)
	}
}

// CT-007: GetNationalTeamByID mantém a assinatura (string, error), retornando
// apenas o nome — não vaza FlagURL para quem não precisa.
func TestIntegration_GetNationalTeamByID_DoesNotLeakFlagURL(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	// The return type is string (the name) — the type system already guarantees
	// no FlagURL is exposed. This assertion locks the contract behaviourally.
	name, err := repo.GetNationalTeamByID(ctx, seedBrasilID)

	require.NoError(t, err)
	require.Equal(t, seedBrasilName, name)
}

// CT-014: a coluna flag_url do schema (inglês) é exposta no tipo gerado pelo
// sqlc como FlagUrl, sem alias SQL nem conversão manual. O scan abaixo só
// compila/preenche se o campo gerado for FlagUrl.
func TestIntegration_SqlcLanguageBridge_FlagURLField(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)

	row, err := sqlc.New(db).GetNationalTeamByID(ctx, seedBrasilID)

	require.NoError(t, err)
	require.Equal(t, seedBrasilFlagURL, row.FlagUrl)
}

// CT-024: busca por id existente do seed — seleção correta.
func TestIntegration_GetNationalTeamByID_Seed(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	name, err := repo.GetNationalTeamByID(ctx, seedBrasilID)

	require.NoError(t, err)
	require.Equal(t, seedBrasilName, name)
}

// CT-025: id inexistente — not-found tratável via errors.Is.
func TestIntegration_GetNationalTeamByID_NotFound(t *testing.T) {
	ctx := context.Background()
	repo := newRepo(t)

	_, err := repo.GetNationalTeamByID(ctx, "00000000-0000-0000-0000-000000000000")

	require.Error(t, err)
	require.True(t, errors.Is(err, repository.ErrNationalTeamNotFound), "error must unwrap to ErrNationalTeamNotFound")
}
