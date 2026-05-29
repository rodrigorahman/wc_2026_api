# Rule candidates — national-team-flag-url/v1

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-05-29T16:55:50Z | agent-spec-qa-validator | repeated_assertion_shape | `require.Equal("https://flagcdn.com/w320/{x}.png", flagURL)` e `require.NotEmpty(team.FlagURL/GetFlagUrl())` repetidos em repository_test:114, service_test:64, e2e_test:74 | TC-001 / domínio nationalteam — asserções de FlagURL |
| 2026-05-29T16:55:50Z | agent-spec-qa-validator | repeated_fixture | ID seed Brasil `a1f3c5e7-0001-4000-8000-000000000001` usado como fixture em national_team_repository_test.go:19 e user_repository_test.go:21 | TC-001 / IDs canônicos do seed 000002 reutilizados cross-package |
