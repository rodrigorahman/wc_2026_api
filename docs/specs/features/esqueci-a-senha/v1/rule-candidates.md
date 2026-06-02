# Rule candidates — esqueci-a-senha/v1

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Usar Resend (resend.com) como serviço de envio de e-mail" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Recuperação via senha temporária enviada por e-mail (não token/OTP)" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Troca obrigatória de senha no primeiro login após o reset" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Resposta sempre genérica no pedido de reset (anti-enumeração)" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Expiração da senha temporária: 15 minutos" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-sdd-run-tasks | pre_refinement_decision | "Rate limit fica para v2" | agent-spec-pre-refinement / esqueci-a-senha |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "Padrão t.Setenv(JWT_SECRET/DB_PATH/APP_ENV[/RESEND_*]) + config.Load() + require.NoError/Error em 6+ testes" | T4 / config.Load fail-fast secrets Resend |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_fixture | "adapter repository->service reimplementado em auth_service_integration_test.go:29 e auth_handler_test.go:46, espelhando module.go:137" | T5 / adapter de teste repository->service |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "bcrypt.CompareHashAndPassword(hash, knownTempPassword) repetido em CT-001/CT-026/CT-003" | T5 / verificação de hash bcrypt da temp |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_fixture | "helper newObservedLogger(t) usado em 6 funções de teste" | T3 / infra/email NoopSender e ResendSender |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "loop range logs.All() + assert.NotContains(body) em 3 testes de segurança de log" | T3 / infra/email varredura de não-vazamento de body |
| 2026-06-02T01:02:42Z | staff-review | convention_drift | "Nomes de função de teste em pt-BR (TestNoopSender_Send_RetornaNilENaoLogaBody) — divergência: T1/T4/T5 usaram pt-BR e passaram; ADR-0005 ambíguo quanto a nomes de teste" | T3 / infra/email — idioma de nomes de teste |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_fixture | "clock injetável reimplementado: fixedClock (unit) + advanceableClock (integração) no pacote service" | T6 / AuthService.Login temp password |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "require.Equal(codes.Unauthenticated, status.Code(err)) + require.Empty(tokens.generatedFor) em CT-008/009/024" | T6 / Login caminhos de negação |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "require.NoError(bcrypt.CompareHashAndPassword(hashArg, plain)) p/ verificar hash persistido em CT-001/003/011/016/025" | T7 / verificação de hash bcrypt real |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_fixture | "waitForEmail + recoveryTempFromBody como setup de 3-4 linhas em CT-031/032/033/034" | T10 / e2e recuperação |
| 2026-06-02T01:02:42Z | agent-spec-qa-validator | repeated_assertion_shape | "require.NoError + require.True/False(resp.GetPasswordChangeRequired()) em CT-031/032/033/034" | T10 / e2e asserção PasswordChangeRequired |
