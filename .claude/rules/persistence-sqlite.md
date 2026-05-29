---
description: Persistência da WC 2026 API — SQLite pure-Go (modernc, sem CGO), PRAGMAs de conexão, migrations embarcadas, queries parametrizadas e código sqlc gerado. Carregada ao editar a camada de banco, repositórios, queries/migrations, sqlc.yaml ou o build. Materializa a ADR-0001.
paths:
  - "internal/infra/db/**"
  - "internal/**/repository/**"
  - "sqlc.yaml"
  - "go.mod"
  - "Makefile"
---

# WC 2026 API — Persistência (SQLite / sqlc)

> ADR-0001 (driver modernc pure-Go). Detalhe em [`docs/adr/0001-driver-modernc-sqlite-pure-go-portabilidade.md`](../../docs/adr/0001-driver-modernc-sqlite-pure-go-portabilidade.md).

## Driver e portabilidade (sem CGO)

- Driver SQLite é **sempre** `modernc.org/sqlite` (pure-Go). **NUNCA** `github.com/mattn/go-sqlite3` nem o driver `sqlite3` do golang-migrate — use o driver `sqlite` (modernc) também na migração.
- Todo build é `CGO_ENABLED=0`. Não adicione dependência que exija toolchain C.
- `make build-all` (linux/amd64, linux/arm64, darwin/arm64, windows/amd64) é a prova de portabilidade — mantenha verde.

## Conexão (`internal/infra/db/db.go`)

- Aplique, por conexão, os PRAGMAs: `foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`.
- `SetMaxOpenConns(1)` (single-writer do SQLite).
- Migrations e queries reutilizam o **mesmo** `*sql.DB` (não abra conexão paralela).

## Migrations

- Versionadas em `internal/infra/db/migrations/` com par `*.up.sql` / `*.down.sql`.
- Embarcadas via `embed.FS` e aplicadas no `OnStart` do `db.Module`; `OnStop` fecha o banco. **Não** há passo manual de migração em runtime.
- Garantias finais (UNIQUE de e-mail, FK de `national_team_id`) dependem de `foreign_keys=ON` — preservar.

## Queries (sqlc)

- Em `internal/infra/db/queries/*.sql` com anotações sqlc (`-- name: X :one|:many`), **sempre parametrizadas** (`?`). Nunca concatene SQL.
- Gere com `make sqlc` → `internal/infra/db/sqlc/**`. Esse código é **gerado — não editar à mão** (será sobrescrito). Alterou query/schema → regenere e commite o gerado.

> Idioma dos identificadores de schema (inglês, sem bridge de tradução) vive em [`language-naming.md`](language-naming.md).
