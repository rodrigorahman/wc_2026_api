package db

import (
	"context"
	"database/sql"

	"go.uber.org/fx"

	"github.com/rodrigorahman/wc_2026_api/internal/infra/config"
	"github.com/rodrigorahman/wc_2026_api/internal/infra/db/sqlc"
)

// Module wires the persistence layer into the fx application graph. It provides
// the shared *sql.DB and *sqlc.Queries, applies the embedded migrations on
// OnStart, and closes the connection on OnStop (§10.2).
var Module = fx.Module("db",
	fx.Provide(
		provideDB,
		provideQueries,
	),
)

// provideDB opens the SQLite connection from the configured DB path and
// registers the lifecycle hooks: migrations run on start, the handle closes on
// stop.
func provideDB(lc fx.Lifecycle, cfg config.Config) (*sql.DB, error) {
	database, err := Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return Migrate(database)
		},
		OnStop: func(context.Context) error {
			return database.Close()
		},
	})

	return database, nil
}

// provideQueries builds the sqlc Queries over the shared connection.
func provideQueries(database *sql.DB) *sqlc.Queries {
	return sqlc.New(database)
}
