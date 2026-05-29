package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/repository"
	"github.com/rodrigorahman/wc_2026_api/internal/domain/match/service"
	appdb "github.com/rodrigorahman/wc_2026_api/internal/infra/db"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
	"github.com/rodrigorahman/wc_2026_api/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helpers locais: mock manual + clock fixo
// ---------------------------------------------------------------------------

// matchRepoMock is a manual mock of service.MatchRepository. It captures the
// arguments passed by the service and delegates to listFn to control output.
type matchRepoMock struct {
	listFn          func(ctx context.Context, userID string, cutoff time.Time) ([]repository.Match, error)
	listCalled      bool
	capturedUserID  string
	capturedCutoff  time.Time
}

func (m *matchRepoMock) ListUpcomingMatchesByUser(ctx context.Context, userID string, cutoff time.Time) ([]repository.Match, error) {
	m.listCalled = true
	m.capturedUserID = userID
	m.capturedCutoff = cutoff
	return m.listFn(ctx, userID, cutoff)
}

// fixedClock is a deterministic clock for unit tests.
type fixedClock struct {
	now time.Time
}

func (c *fixedClock) Now() time.Time { return c.now }

// ---------------------------------------------------------------------------
// T4-CT-001: TestListUpcoming_CutoffEqualsClockNow
// ---------------------------------------------------------------------------

// T4-CT-001: o service passa exatamente clk.Now() como cutoff ao repository.
func TestListUpcoming_CutoffEqualsClockNow(t *testing.T) {
	t0 := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	clk := &fixedClock{now: t0}

	mock := &matchRepoMock{
		listFn: func(_ context.Context, _ string, _ time.Time) ([]repository.Match, error) {
			return []repository.Match{}, nil
		},
	}

	svc := service.NewMatchService(mock, clk)

	_, err := svc.ListUpcomingMatches(context.Background(), "user-test")

	require.NoError(t, err)
	require.True(t, mock.listCalled, "service must call ListUpcomingMatchesByUser on the repository")
	require.Equal(t, "user-test", mock.capturedUserID, "service must forward the userID unchanged")
	require.Equal(t, t0, mock.capturedCutoff, "service must pass clk.Now() as cutoff")
}

// ---------------------------------------------------------------------------
// T4-CT-002: TestListUpcoming_PassesThroughMatches
// ---------------------------------------------------------------------------

// T4-CT-002: service mapeia todos os campos do repository para os tipos do service.
func TestListUpcoming_PassesThroughMatches(t *testing.T) {
	t0 := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	clk := &fixedClock{now: t0}

	repoMatches := []repository.Match{
		{
			ID:      "m-0001",
			Kickoff: t0.Add(2 * time.Hour),
			HomeTeam: repository.MatchTeam{
				ID:      "team-br",
				Name:    "Brasil",
				FlagURL: "https://flagcdn.com/w320/br.png",
				Code:    "BRA",
			},
			AwayTeam: repository.MatchTeam{
				ID:      "team-ar",
				Name:    "Argentina",
				FlagURL: "https://flagcdn.com/w320/ar.png",
				Code:    "ARG",
			},
			Stadium: "Estádio Azteca",
			City:    "Cidade do México",
			Stage:   "Grupo A",
		},
		{
			ID:      "m-0002",
			Kickoff: t0.Add(24 * time.Hour),
			HomeTeam: repository.MatchTeam{
				ID:      "team-fr",
				Name:    "França",
				FlagURL: "https://flagcdn.com/w320/fr.png",
				Code:    "FRA",
			},
			AwayTeam: repository.MatchTeam{
				ID:      "team-de",
				Name:    "Alemanha",
				FlagURL: "https://flagcdn.com/w320/de.png",
				Code:    "GER",
			},
			Stadium: "MetLife Stadium",
			City:    "Nova York",
			Stage:   "Final",
		},
	}

	mock := &matchRepoMock{
		listFn: func(_ context.Context, _ string, _ time.Time) ([]repository.Match, error) {
			return repoMatches, nil
		},
	}

	svc := service.NewMatchService(mock, clk)

	matches, err := svc.ListUpcomingMatches(context.Background(), "user-test")

	require.NoError(t, err)
	require.Len(t, matches, 2)

	require.Equal(t, repoMatches[0].ID, matches[0].ID)
	require.Equal(t, repoMatches[0].Kickoff, matches[0].Kickoff)
	require.Equal(t, repoMatches[0].HomeTeam.ID, matches[0].HomeTeam.ID)
	require.Equal(t, repoMatches[0].HomeTeam.Name, matches[0].HomeTeam.Name)
	require.Equal(t, repoMatches[0].HomeTeam.FlagURL, matches[0].HomeTeam.FlagURL)
	require.Equal(t, repoMatches[0].HomeTeam.Code, matches[0].HomeTeam.Code)
	require.Equal(t, repoMatches[0].AwayTeam.ID, matches[0].AwayTeam.ID)
	require.Equal(t, repoMatches[0].AwayTeam.Name, matches[0].AwayTeam.Name)
	require.Equal(t, repoMatches[0].AwayTeam.FlagURL, matches[0].AwayTeam.FlagURL)
	require.Equal(t, repoMatches[0].AwayTeam.Code, matches[0].AwayTeam.Code)
	require.Equal(t, repoMatches[0].Stadium, matches[0].Stadium)
	require.Equal(t, repoMatches[0].City, matches[0].City)
	require.Equal(t, repoMatches[0].Stage, matches[0].Stage)

	require.Equal(t, repoMatches[1].ID, matches[1].ID)
	require.Equal(t, repoMatches[1].HomeTeam.Code, matches[1].HomeTeam.Code)
	require.Equal(t, repoMatches[1].AwayTeam.Code, matches[1].AwayTeam.Code)
}

// ---------------------------------------------------------------------------
// T4-CT-003: TestListUpcoming_PropagatesRepoError
// ---------------------------------------------------------------------------

// T4-CT-003: service propaga erro do repository via errors.Is.
func TestListUpcoming_PropagatesRepoError(t *testing.T) {
	clk := &fixedClock{now: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)}
	repoErr := errors.New("database unavailable")

	mock := &matchRepoMock{
		listFn: func(_ context.Context, _ string, _ time.Time) ([]repository.Match, error) {
			return nil, repoErr
		},
	}

	svc := service.NewMatchService(mock, clk)

	_, err := svc.ListUpcomingMatches(context.Background(), "user-test")

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
}

// ---------------------------------------------------------------------------
// T4-CT-004: TestListUpcoming_EmptyRepoResult_EmptySlice
// ---------------------------------------------------------------------------

// T4-CT-004: repo vazio → NoError, slice não nil e len==0.
func TestListUpcoming_EmptyRepoResult_EmptySlice(t *testing.T) {
	clk := &fixedClock{now: time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)}

	mock := &matchRepoMock{
		listFn: func(_ context.Context, _ string, _ time.Time) ([]repository.Match, error) {
			return []repository.Match{}, nil
		},
	}

	svc := service.NewMatchService(mock, clk)

	matches, err := svc.ListUpcomingMatches(context.Background(), "user-test")

	require.NoError(t, err)
	require.NotNil(t, matches)
	require.Len(t, matches, 0)
}

// ---------------------------------------------------------------------------
// T4-CT-005: integração — service + repository real + clock avançável
// ---------------------------------------------------------------------------

// Seed IDs from migration 000002_seed_national_teams.up.sql.
const (
	intBrasilID    = "a1f3c5e7-0001-4000-8000-000000000001"
	intArgentinaID = "a1f3c5e7-0002-4000-8000-000000000002"
	intTestUserID  = "b0000000-0000-4000-8000-000000000099"
)

// insertIntegrationUser inserts a minimal user row to satisfy the FK from
// user_national_teams. Must be called before inserting favorites.
func insertIntegrationUser(t *testing.T, ctx context.Context, db *sql.DB, ref time.Time) {
	t.Helper()
	_, err := db.ExecContext(ctx,
		"INSERT INTO users (id, full_name, email, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
		intTestUserID, "Integration User", "integration@example.com", "hash", ref)
	if err != nil {
		t.Fatalf("insert integration user: %v", err)
	}
}

// advancableClock is a clock that can be advanced for integration tests.
type advancableClock struct {
	now time.Time
}

func (c *advancableClock) Now() time.Time       { return c.now }
func (c *advancableClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// TestIntegration_ListUpcoming_ClockDrivesCut verifica que o clock injetado
// controla o corte: em T0 retorna a partida futura; avançado para T0+2h não
// retorna nada (a partida ficou no passado do novo "agora").
func TestIntegration_ListUpcoming_ClockDrivesCut(t *testing.T) {
	ctx := context.Background()

	dbPath := t.TempDir() + "/int.db"
	database, err := appdb.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	require.NoError(t, appdb.Migrate(database))
	// Isola do seed da Copa 2026 (000008): este teste é dono das suas fixtures de jogo.
	if _, derr := database.ExecContext(ctx, "DELETE FROM matches"); derr != nil {
		t.Fatalf("clear matches: %v", derr)
	}

	t0 := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	insertIntegrationUser(t, ctx, database, t0)

	// Favorita: Brasil
	_, err = database.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		intTestUserID, intBrasilID)
	require.NoError(t, err)

	// Partida às T0+1h (futuro a partir de T0, passado a partir de T0+2h)
	kickoff := t0.Add(time.Hour)
	_, err = database.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-int-001", kickoff, intBrasilID, intArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo A")
	require.NoError(t, err)

	clk := &advancableClock{now: t0}
	repo := repository.NewMatchRepository(sqlc.New(database))
	svc := service.NewMatchService(repo, clk)

	// clock em T0: partida deve aparecer (kickoff T0+1h > T0)
	matches, err := svc.ListUpcomingMatches(ctx, intTestUserID)
	require.NoError(t, err)
	require.Len(t, matches, 1, "clock at T0: match at T0+1h must be returned")

	// avança clock para T0+2h: partida em T0+1h ficou no passado
	clk.Advance(2 * time.Hour)

	matches, err = svc.ListUpcomingMatches(ctx, intTestUserID)
	require.NoError(t, err)
	require.Len(t, matches, 0, "clock at T0+2h: match at T0+1h must no longer appear")
}

// ---------------------------------------------------------------------------
// Verify testutil.TestNewDB can also be used (mock budget companion)
// ---------------------------------------------------------------------------

// TestIntegration_ListUpcoming_ViaTestNewDB verifica o mesmo comportamento
// usando testutil.TestNewDB para satisfazer o mock budget: todo teste com mock
// tem companion real sem mock cobrindo o mesmo caminho.
func TestIntegration_ListUpcoming_ViaTestNewDB(t *testing.T) {
	ctx := context.Background()
	database := testutil.TestNewDB(t)
	// Isola do seed da Copa 2026 (000008): este teste é dono das suas fixtures de jogo.
	if _, derr := database.ExecContext(ctx, "DELETE FROM matches"); derr != nil {
		t.Fatalf("clear matches: %v", derr)
	}

	t0 := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)

	insertIntegrationUser(t, ctx, database, t0)

	_, err := database.ExecContext(ctx,
		"INSERT INTO user_national_teams (user_id, national_team_id) VALUES (?, ?)",
		intTestUserID, intBrasilID)
	require.NoError(t, err)

	kickoff := t0.Add(time.Hour)
	_, err = database.ExecContext(ctx,
		"INSERT INTO matches (id, kickoff, home_team_id, away_team_id, stadium, city, stage) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"m-budget-001", kickoff, intBrasilID, intArgentinaID,
		"Estádio Azteca", "Cidade do México", "Grupo A")
	require.NoError(t, err)

	clk := &fixedClock{now: t0}
	repo := repository.NewMatchRepository(sqlc.New(database))
	svc := service.NewMatchService(repo, clk)

	matches, err := svc.ListUpcomingMatches(ctx, intTestUserID)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "m-budget-001", matches[0].ID)
}
