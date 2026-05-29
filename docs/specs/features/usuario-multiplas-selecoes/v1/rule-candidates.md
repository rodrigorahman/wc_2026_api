# Rule candidates — usuario-multiplas-selecoes/v1

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-05-29T13:30:00Z | agent-spec-qa-validator | repeated_fixture | Constantes de seed de seleção (seededBrasilID/seededArgentinaID) redeclaradas em `internal/auth/wiring_test.go:28`, `test/e2e/auth_e2e_test.go:19`, `internal/auth/repository/user_repository_test.go:614` | TC-001 / domínio auth — fixtures de national-team seed |
| 2026-05-29T13:45:00Z | staff-review | scope_deviation | Helper de teste `internal/testutil/migrator.go` criado para cobrir AC1 (migration down) sem constar na §8.2 declarada; necessidade de migrator completo (vs TestNewDB só-up) não prevista no planejamento | TC-001 / camada de testes de migration |
