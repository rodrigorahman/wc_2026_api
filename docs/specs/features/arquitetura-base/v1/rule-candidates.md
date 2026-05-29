# Rule candidates — arquitetura-base/v1

> Append-only. Emitido por agentes do framework durante o run. Consumido por `mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-05-29T00:00:00Z | qa-validator | repeated_fixture | helper teamFound() em 6 testes | T7 / AuthService bcrypt+anti-timing |
| 2026-05-29T00:00:00Z | qa-validator | repeated_fixture | builder baseUser() + newRepo(t) em 4 testes | T8 / UserRepository (auth) integração |
| 2026-05-29T00:00:00Z | qa-validator | repeated_fixture | userRepositoryAdapter duplicado em auth_handler_test.go:45 e module.go:113 | T11 / handler+module auth |
| 2026-05-29T00:00:00Z | qa-validator | repeated_fixture | testutil.TestNewDB(t) em 3 testes (module_test + repository_test) | T9 / módulo NationalTeam |
| 2026-05-29T00:00:00Z | qa-validator | repeated_assertion_shape | require.True(errors.Is(err, X)) em 3 lugares | T9 / módulo NationalTeam |
| 2026-05-29T00:00:00Z | qa-validator | repeated_assertion_shape | require.Equal(codes.X, status.Code(err)) em E2E+wiring | T12 / E2E auth + wiring |
| 2026-05-29T00:00:00Z | qa-validator | repeated_fixture | seededBrasilID const redeclarado em e2e e wiring | T12 / fixtures de seed |
