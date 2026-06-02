package repository_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// Seed IDs from migration 000002_seed_national_teams.up.sql (both valid FKs).
const (
	seedBrasilID    = "a1f3c5e7-0001-4000-8000-000000000001"
	seedArgentinaID = "a1f3c5e7-0002-4000-8000-000000000002"

	missingTeamID = "00000000-0000-0000-0000-000000000000"
)

// CT-T2-001: a tabela matches existe com as 7 colunas, todas NOT NULL.
func TestIntegration_MatchesTable_ColumnsNotNull(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)

	columns := []string{"id", "kickoff", "home_team_id", "away_team_id", "stadium", "city", "stage"}
	for _, col := range columns {
		var name string
		var notNull int
		err := db.QueryRowContext(ctx,
			"SELECT name, \"notnull\" FROM pragma_table_info('matches') WHERE name = ?", col).
			Scan(&name, &notNull)

		require.NoError(t, err, "column %s must exist in matches", col)
		require.Equal(t, col, name)
		require.Equal(t, 1, notNull, "column %s must be NOT NULL", col)
	}
}

// CT-T2-004: INSERT válido (Brasil×Argentina do seed, todos os campos) aceito.
func TestIntegration_Matches_ValidInsert(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)

	_, err := db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-0001", time.Date(2026, 6, 11, 20, 0, 0, 0, time.UTC), seedBrasilID, seedArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo A")
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM matches").Scan(&count))
	require.Equal(t, 1, count)
}

// CT-T2-002: foreign_keys=ON rejeita home/away inexistente (table-driven).
func TestIntegration_MatchFK_RejectsInvalidTeam(t *testing.T) {
	tests := []struct {
		name   string
		homeID string
		awayID string
	}{
		{name: "home inexistente", homeID: missingTeamID, awayID: seedArgentinaID},
		{name: "away inexistente", homeID: seedBrasilID, awayID: missingTeamID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			db := testutil.TestNewDB(t)

			_, err := db.ExecContext(ctx,
				"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"m-fk", time.Date(2026, 6, 11, 20, 0, 0, 0, time.UTC), tc.homeID, tc.awayID,
				"Estádio Azteca", "Cidade do México", "Grupo A")

			require.Error(t, err, "FK constraint must reject nonexistent team")
		})
	}
}

// CT-T2-003: o CHECK impede mandante == visitante (ambos válidos isola o CHECK da FK).
func TestIntegration_Matches_RejectsSameTeam(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)

	_, err := db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-same", time.Date(2026, 6, 11, 20, 0, 0, 0, time.UTC), seedBrasilID, seedBrasilID,
		"Estádio Azteca", "Cidade do México", "Grupo A")

	require.Error(t, err, "CHECK must reject home_team_id == away_team_id")
}

// CT-T2-005: a down migration 000007 remove a tabela matches, exercitada pelo
// mesmo migrator de produção.
func TestIntegration_MatchesTable_DownDrops(t *testing.T) {
	ctx := context.Background()

	dsn := filepath.Join(t.TempDir(), "down007.db")
	migrator, db := testutil.NewMigratorForTest(t, dsn)

	require.NoError(t, migrator.Up())

	var beforeDown int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'matches'").
		Scan(&beforeDown))
	require.Equal(t, 1, beforeDown, "matches must exist before down")

	require.NoError(t, migrator.Steps(-3)) // -3: desfaz 000009 (temp password), 000008 (seed Copa) e 000007 (cria matches)

	var afterDown int
	require.NoError(t, db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'matches'").
		Scan(&afterDown))
	require.Equal(t, 0, afterDown, "matches must be dropped after down")
}

// CT-T2-006: o pacote sqlc regenerado expõe ListUpcomingMatchesByUserRow com
// todos os campos esperados, e NationalTeam ganhou Code. A declaração só compila
// se cada campo existir com o tipo correto.
func TestSqlc_MatchRow_HasExpectedFields(t *testing.T) {
	var row sqlc.ListUpcomingMatchesByUserRow
	row.ID = ""
	row.Kickoff = time.Time{}
	row.Stadium = ""
	row.City = ""
	row.Stage = ""
	row.HomeID = ""
	row.HomeName = ""
	row.HomeFlagUrl = ""
	row.HomeCode = ""
	row.AwayID = ""
	row.AwayName = ""
	row.AwayFlagUrl = ""
	row.AwayCode = ""

	var team sqlc.NationalTeam
	team.Code = ""

	var match sqlc.Match
	match.HomeTeamID = ""
	match.AwayTeamID = ""

	// CT-T2-006 é um gate de compilação: as atribuições acima só compilam se
	// cada campo existir com o tipo esperado no pacote sqlc regenerado. Sem
	// assertion runtime — o valor real do caso é a presença dos campos.
	_ = match
}

// ---------------------------------------------------------------------------
// Constantes e fixtures para T3
// ---------------------------------------------------------------------------

const (
	seedCoreiaDoSulID = "a1f3c5e7-0016-4000-8000-000000000016"

	testUserID = "a0000000-0000-4000-8000-000000000099"
)

// t0 é o instante de corte fixo usado nos testes T3 para determinismo.
var t0 = time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

// insertTestUser insere um usuário mínimo para satisfazer a FK de
// user_national_teams. Deve ser chamado antes de inserir favoritas.
func insertTestUser(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (id, full_name, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		testUserID, "Test User", "test@example.com", "hash", t0)
	if err != nil {
		t.Fatalf("insert test user: %v", err)
	}
}

// clearMatches remove os jogos semeados pela migração 000008 (Copa 2026) para
// que cada teste seja dono das fixtures de partida que insere e asserta contagens
// em isolamento do seed.
func clearMatches(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	if _, err := db.ExecContext(ctx, "DELETE FROM matches"); err != nil {
		t.Fatalf("clear matches: %v", err)
	}
}

// ---------------------------------------------------------------------------
// T3-CT-001: ordenação por kickoff ASC
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_OrdersByKickoffAsc verifica que a query
// retorna partidas ordenadas por kickoff ascendente (CA2).
func TestIntegration_ListUpcoming_OrdersByKickoffAsc(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)
	repo := repository.NewMatchRepository(sqlc.New(db))

	insertTestUser(t, ctx, db)

	// Favorita: Brasil
	_, err := db.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		testUserID, seedBrasilID)
	require.NoError(t, err)

	// Jogo T0+48h (inserido primeiro)
	_, err = db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-ord-late", t0.Add(48*time.Hour), seedBrasilID, seedArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo A")
	require.NoError(t, err)

	// Jogo T0+2h (inserido segundo — deve aparecer primeiro na ordenação)
	_, err = db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-ord-early", t0.Add(2*time.Hour), seedBrasilID, seedArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo B")
	require.NoError(t, err)

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.Len(t, matches, 2)
	require.True(t, matches[0].Kickoff.Before(matches[1].Kickoff),
		"first match must have earlier kickoff than second")
	require.Equal(t, t0.Add(2*time.Hour).UTC(), matches[0].Kickoff.UTC())
	require.Equal(t, t0.Add(48*time.Hour).UTC(), matches[1].Kickoff.UTC())
}

// ---------------------------------------------------------------------------
// T3-CT-002: corte estritamente futuro (boundary == cutoff excluído)
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_StrictlyFutureCut verifica que partidas em
// exatamente T0 (igual ao cutoff) são excluídas; apenas T0+1s retorna (CA3).
func TestIntegration_ListUpcoming_StrictlyFutureCut(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)
	repo := repository.NewMatchRepository(sqlc.New(db))

	insertTestUser(t, ctx, db)

	_, err := db.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		testUserID, seedBrasilID)
	require.NoError(t, err)

	for _, tc := range []struct{ id string; kickoff time.Time }{
		{"m-past", t0.Add(-time.Second)},
		{"m-equal", t0},
		{"m-future", t0.Add(time.Second)},
	} {
		_, err = db.ExecContext(ctx,
			"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
			tc.id, tc.kickoff, seedBrasilID, seedArgentinaID,
			"Estádio Azteca", "Cidade do México", "Grupo A")
		require.NoError(t, err)
	}

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.Len(t, matches, 1, "only the strictly-future match must be returned")
	require.Equal(t, t0.Add(time.Second).UTC(), matches[0].Kickoff.UTC())
}

// ---------------------------------------------------------------------------
// T3-CT-003: dedup — partida entre duas favoritas aparece uma só vez
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_DedupTwoFavorites verifica que uma partida
// envolvendo dois times favoritos não é duplicada no resultado (CA4).
func TestIntegration_ListUpcoming_DedupTwoFavorites(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)
	repo := repository.NewMatchRepository(sqlc.New(db))

	insertTestUser(t, ctx, db)

	// Favoritas: Brasil e Argentina
	for _, teamID := range []string{seedBrasilID, seedArgentinaID} {
		_, err := db.ExecContext(ctx,
			"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
			testUserID, teamID)
		require.NoError(t, err)
	}

	_, err := db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-dedup", t0.Add(time.Hour), seedBrasilID, seedArgentinaID,
		"Estádio Azteca", "Cidade do México", "Final")
	require.NoError(t, err)

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.Len(t, matches, 1, "match between two favorites must not be duplicated")
}

// ---------------------------------------------------------------------------
// T3-CT-004: usuário sem favoritas retorna slice vazio (não nil)
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_NoFavorites_Empty verifica que um usuário sem
// favoritas recebe slice vazio e não nil (CA5).
func TestIntegration_ListUpcoming_NoFavorites_Empty(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	repo := repository.NewMatchRepository(sqlc.New(db))

	// Nenhuma favorita inserida para testUserID

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.NotNil(t, matches, "result must not be nil")
	require.Len(t, matches, 0)
}

// ---------------------------------------------------------------------------
// T3-CT-005: favorita sem jogos futuros retorna slice vazio
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_FavoritesNoFutureGames_Empty verifica que
// quando o único jogo do time favorito é passado, retorna vazio (CA5).
func TestIntegration_ListUpcoming_FavoritesNoFutureGames_Empty(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)
	repo := repository.NewMatchRepository(sqlc.New(db))

	insertTestUser(t, ctx, db)

	_, err := db.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		testUserID, seedBrasilID)
	require.NoError(t, err)

	// Único jogo: 1 hora antes do cutoff
	_, err = db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-past-only", t0.Add(-time.Hour), seedBrasilID, seedArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo A")
	require.NoError(t, err)

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.NotNil(t, matches, "result must not be nil")
	require.Len(t, matches, 0)
}

// ---------------------------------------------------------------------------
// T3-CT-006: mapeamento completo de todos os campos
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_MapsAllFields verifica que todos os campos do
// domínio são mapeados corretamente a partir da linha sqlc (CA6).
func TestIntegration_ListUpcoming_MapsAllFields(t *testing.T) {
	ctx := context.Background()
	db := testutil.TestNewDB(t)
	clearMatches(t, ctx, db)
	repo := repository.NewMatchRepository(sqlc.New(db))

	insertTestUser(t, ctx, db)

	_, err := db.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		testUserID, seedBrasilID)
	require.NoError(t, err)

	kickoff := time.Date(2026, 7, 19, 21, 0, 0, 0, time.UTC)
	_, err = db.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-fields", kickoff, seedBrasilID, seedCoreiaDoSulID,
		"MetLife Stadium", "Nova York", "Oitavas de Final")
	require.NoError(t, err)

	matches, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.NoError(t, err)
	require.Len(t, matches, 1)

	m := matches[0]
	require.Equal(t, "m-fields", m.ID)
	require.Equal(t, kickoff.UTC(), m.Kickoff.UTC())
	require.Equal(t, "MetLife Stadium", m.Stadium)
	require.Equal(t, "Nova York", m.City)
	require.Equal(t, "Oitavas de Final", m.Stage)

	require.Equal(t, seedBrasilID, m.HomeTeam.ID)
	require.Equal(t, "BRA", m.HomeTeam.Code, "HomeTeam.Code must be 'BRA'")
	require.Equal(t, "Brasil", m.HomeTeam.Name)
	require.Equal(t, "https://flagcdn.com/w320/br.png", m.HomeTeam.FlagURL)

	require.Equal(t, seedCoreiaDoSulID, m.AwayTeam.ID)
	require.Equal(t, "KOR", m.AwayTeam.Code, "AwayTeam.Code must be 'KOR'")
	require.Equal(t, "Coreia do Sul", m.AwayTeam.Name)
	require.Equal(t, "https://flagcdn.com/w320/kr.png", m.AwayTeam.FlagURL)
}

// ---------------------------------------------------------------------------
// T3-CT-007/008: schema constraints (cobertos por CT-T2-003 e CT-T2-002)
// ---------------------------------------------------------------------------

// Pendências: T3-CT-007 (mandante == visitante) e T3-CT-008 (FK inexistente)
// são semanticamente idênticos a CT-T2-003 e CT-T2-002, que já residem neste
// arquivo e exercitam os mesmos paths de schema com o mesmo driver. Não
// duplicados para evitar AP-26 (redundância semântica).

// ---------------------------------------------------------------------------
// T3-CT-009: propagação de erro do banco (ctx cancelado)
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_PropagatesDBError verifica que erros do banco
// são propagados com wrap e sem panic; usa ctx cancelado para forçar falha.
func TestIntegration_ListUpcoming_PropagatesDBError(t *testing.T) {
	db := testutil.TestNewDB(t)
	repo := repository.NewMatchRepository(sqlc.New(db))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancela antes da chamada

	_, err := repo.ListUpcomingMatchesByUser(ctx, testUserID, t0)
	require.ErrorIs(t, err, context.Canceled)
}
