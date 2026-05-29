# Rule candidates — proximos-jogos-selecoes/v1

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-05-29T20:04:13Z | agent-spec-qa-validator | repeated_fixture | Helper insertTestUser + testutil.TestNewDB + INSERT INTO user_national_teams em 5 testes T3 | T3 / repository MatchRepository |
| 2026-05-29T20:04:13Z | agent-spec-qa-validator | repeated_assertion_shape | require.NoError + require.NotNil + require.Len(...,0) idêntico em CT-004 e CT-005 | T3 / cenários lista vazia |
| 2026-05-29T20:13:09Z | agent-spec-qa-validator | repeated_fixture | fixedClock local (struct now+Now()) em 4 funções de teste | T4 / service MatchService |
| 2026-05-29T20:13:09Z | agent-spec-qa-validator | repeated_assertion_shape | require.Equal(repoMatches[N].Field, matches[N].Field) em 13+ asserts de mapeamento | T4 / TestListUpcoming_PassesThroughMatches |
| 2026-05-29T20:37:29Z | agent-spec-minispec-run-tasks | executor_askquestion | "Como injetar subject autenticado nos testes do handler mantendo subKey encapsulado? (export_test.go cross-pacote não compila)" | T5 / seam de teste de auth (RPC protegido) |
| 2026-05-29T20:53:05Z | agent-spec-qa-validator | repeated_assertion_shape | fx.ValidateApp(...módulos..., server.Providers, fx.Supply(cfg), fx.Invoke) + require.NoError em 3 smoke tests de wiring | T6 / smoke tests wiring fx |
| 2026-05-29T20:55:05Z | staff-review | scope_deviation | smoke de wiring fx compartilhado (auth/wiring_test.go CT-064) precisa enrolar novo módulo ao adicionar domínio (server.Providers exige o handler) | T6 / wiring match.Module |
| 2026-05-29T21:00:04Z | agent-spec-qa-validator | repeated_fixture | testutil.TestNewBufconnServer(t, nil) em 2 testes E2E (padrão repetido em auth/national_team e2e) | T7 / E2E MatchService |
