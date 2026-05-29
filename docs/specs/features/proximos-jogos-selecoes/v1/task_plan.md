# TASK PLAN – MiniSpec

## 1. Identificacao
- **Feature**: proximos-jogos-selecoes (v1)
- **Variante**: backend (Go)
- **Intent**: `docs/specs/features/proximos-jogos-selecoes/v1/intent.md`
- **Scope**: `docs/specs/features/proximos-jogos-selecoes/v1/scope.md`
- **Responsavel**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Status**: Rascunho

---

## 2. Objetivo Tecnico
Entregar o domínio de leitura `match` (handler→service→repository + `fx.Module`) com o RPC gRPC **autenticado** `MatchService.ListUpcomingMatches`, que infere as seleções favoritas do `user_id` do token (anti-IDOR) e devolve os jogos **futuros** dessas seleções em ordem cronológica crescente, com filtro/dedup/ordenação numa única query sqlc cujo corte vem do `clock.Clock` injetado. Inclui a coluna `code` (sigla FIFA) em `national_teams` (migração 000006 + backfill), a tabela `matches` (migração 000007, integridade por schema: FK + NOT NULL + CHECK), o contrato proto `wc2026.match.v1` e a integração no composition root + helper E2E, mantendo o RPC fora da lista de métodos públicos.

---

## 3. Macro-Fases (alto nivel)
- **Fase 1 – Fundamentos de Persistência**
  - Objetivo: coluna `code` + backfill, tabela `matches`, query sqlc tipada.
  - Tasks: T1, T2
- **Fase 2 – Domínio `match`**
  - Objetivo: repository (acesso a dados) e service (corte via clock).
  - Tasks: T3, T4
- **Fase 3 – Superfície gRPC**
  - Objetivo: contrato proto + handler mapper protegido.
  - Tasks: T5
- **Fase 4 – Integração**
  - Objetivo: wiring fx no composition root/E2E + cobertura ponta a ponta.
  - Tasks: T6, T7

---

## 4. Lista de Tasks (visao macro)
| ID | Nome da Task | Arquivo | Fase | Dependencias | Pode Rodar em Paralelo? | Status |
|----|-------------|---------|------|-------------|------------------------|--------|
| T1 | Migração 000006 — coluna `code` + backfill das 16 siglas FIFA | [T1](tasks/T1.md) | Fase 1 | — | Sim (com T5-proto N/A) | A Fazer |
| T2 | Migração 000007 — tabela `matches` + query `ListUpcomingMatchesByUser` + `make sqlc` | [T2](tasks/T2.md) | Fase 1 | T1 | Nao | A Fazer |
| T3 | Repository `MatchRepository` — wrapper sqlc + tipos de domínio | [T3](tasks/T3.md) | Fase 2 | T2 | Nao | A Fazer |
| T4 | Service `MatchService` — orquestrador com corte via `clock.Clock` | [T4](tasks/T4.md) | Fase 2 | T3 | Nao | A Fazer |
| T5 | Proto `MatchService` + `make proto` + Handler `MatchHandler` (protegido) | [T5](tasks/T5.md) | Fase 3 | T4 | Nao | A Fazer |
| T6 | Wiring — `match.Module` (fx) + composition root + helper E2E | [T6](tasks/T6.md) | Fase 4 | T3, T4, T5 | Nao | A Fazer |
| T7 | Testes E2E do `MatchService` (stack autenticado via bufconn) | [T7](tasks/T7.md) | Fase 4 | T6 | Nao | A Fazer |

> **Observação sobre paralelismo**: o contrato proto (gerado dentro de T5) não depende da cadeia de DB; na prática a única paralelização real seria gerar o proto cedo. Como o proto está consolidado em T5 (handler é seu primeiro consumidor) e o caminho crítico é a cadeia de persistência, o pipeline é essencialmente sequencial. Nenhum lote paralelo é declarado (paths sobrepostos / dependências textuais entre camadas).

---

## 5. Ordem de Execucao

```
T1 -> T2 -> T3 -> T4 -> T5 -> T6 -> T7
```

### Grafo de Dependencias
| Task | Depende de | Pode Rodar em Paralelo? | Status |
|------|------------|-------------------------|--------|
| T1 | — | Nao (caminho crítico) | A Fazer |
| T2 | T1 | Nao | A Fazer |
| T3 | T2 | Nao | A Fazer |
| T4 | T3 | Nao | A Fazer |
| T5 | T4 | Nao | A Fazer |
| T6 | T3, T4, T5 | Nao | A Fazer |
| T7 | T6 | Nao | A Fazer |

---

## 6. Arquivos / Areas Impactadas (visao consolidada)

| Area | Arquivos | Acao |
|------|----------|------|
| Proto | `api/proto/wc2026/match/v1/match.proto` | criar |
| Proto (gerado) | `gen/wc2026/match/v1/**` | criar (via `make proto`) |
| Migrations | `internal/infra/db/migrations/000006_add_code_to_national_teams.{up,down}.sql` | criar |
| Migrations | `internal/infra/db/migrations/000007_create_matches.{up,down}.sql` | criar |
| Queries | `internal/infra/db/queries/matches.sql` | criar |
| sqlc (gerado) | `internal/infra/db/sqlc/**` | modificar (via `make sqlc`) |
| Domínio `match` | `internal/domain/match/repository/match_repository.go` (+ `_test.go`) | criar |
| Domínio `match` | `internal/domain/match/service/match_service.go` (+ `_test.go`) | criar |
| Domínio `match` | `internal/domain/match/handler/match_handler.go` (+ `_test.go`) | criar |
| Domínio `match` | `internal/domain/match/module.go` (+ `module_test.go`, `export_test.go` cond.) | criar |
| Auth (teste) | `internal/domain/auth/interceptor/export_test.go` (helper de subject) | criar (condicional) |
| Composition root | `internal/server/server.go` (+ `export_test.go` cond.) | modificar |
| Composition root | `cmd/server/main.go` | modificar |
| E2E helper | `internal/testutil/bufconn.go` | modificar |
| E2E | `test/e2e/match_e2e_test.go` | criar |

> **Legenda de Acoes:** `criar` | `modificar` | `remover`

---

## 7. Criterios de Conclusao Geral
- [ ] Todas as tasks concluidas
- [ ] Objetivo tecnico atingido (RPC autenticado devolve jogos futuros das favoritas; jogos passados não aparecem; partida com duas favoritas aparece uma vez; sem favoritas/sem jogos → vazio)
- [ ] `make sqlc`, `make proto` verdes; código gerado commitado
- [ ] `make test` verde (unit + integração + E2E)
- [ ] `make build-all` (CGO off) verde
- [ ] `ListUpcomingMatches` ausente de `providePublicMethods` (RPC protegido)

---

## 8. Notas para a LLM Executora
- **Molde de referência**: replicar o domínio `nationalteam` (repository/service/handler/module) — naming, interface no consumidor, adapter fino no `fx.Module`, estilo de erro (`fmt.Errorf("...: %w")`, `status.Error(codes.X)`).
- **ADRs invioláveis**: ADR-0001 (driver `modernc`, CGO off), ADR-0002 (fx por domínio + interface no consumidor), ADR-0003 (`sub` do contexto anti-IDOR + corte via `clock.Clock`, nunca `CURRENT_TIMESTAMP`), ADR-0005 (schema/proto em inglês, aliases `home_*`/`away_*` só desambiguam JOIN).
- **Código gerado não se edita à mão** (`gen/**`, `internal/infra/db/sqlc/**`) — regenerar via `make proto`/`make sqlc` e commitar.
- **Determinismo de tempo**: `clk.Now()` injetado; relógio fixo nos testes; sem `time.Sleep`/`time.Now()` em código testável.
- **Boundaries de teste**: integração com SQLite real (`testutil.TestNewDB`), E2E com `testutil.TestNewBufconnServer` (cadeia real). Não mockar o banco; não duplicar a cadeia de interceptors. Owning layer do caminho-feliz com dados = integração-repository (T3), pois o helper E2E não semeia `matches`.
- **`clock.Clock` consumido** do grafo (provido por `auth.Module`) — `match.Module` **não** re-provê (colisão de tipo no fx).
- Disciplina do Executor (Iron Rules) será injetada pelo orquestrador `/agent-spec-minispec-run-tasks` no prompt de cada executor.
