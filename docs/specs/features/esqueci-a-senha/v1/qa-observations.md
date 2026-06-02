# QA / Tech Review Observations — esqueci-a-senha/v1

## Challenge Session — 2026-05-31 (artifact: tech_spec.md)

- Questões processadas: 4 (interrogadas) + verificações de código (national_team_id, adapter, migrations)
- Conflitos de terminologia resolvidos: 0
- Decisões implícitas explicitadas: 3
  - `password_change_required` é derivado (sem coluna de flag) — reconcilia "flag de troca" do tech-alignment §5
  - "Troca obrigatória" = sinalização ao cliente, sem enforcement de backend (RN9-coerente)
  - Timing leak do 2º branch do Login: equalização total avaliada e rejeitada (custo bcrypt), risco aceito mantido
- Contradições com código corrigidas (inline): 1 (alto impacto)
  - `userRepositoryAdapter` (module.go) copia campo a campo repository.User→service.User; spec não listava propagação dos campos temp nem os campos no struct service.User. Sem isso o 2º branch do Login nunca enxerga a temporária (CT-023/CT-032 quebrariam). Atualizado §3.2, §6.2, §22.2.
- Verificações que confirmaram a spec (sem mudança):
  - §7.2 correta: migration 000004 fez `ALTER TABLE users DROP COLUMN national_team_id`; colunas restantes batem.
  - resend-go ausente do go.mod (esperado — dependência a adicionar).
- Termos canonizados no glossário: 3 (nível FEATURE — arquivo criado)
  - senha temporária, troca obrigatória, recuperação de acesso
- Candidatos a ADR sinalizados: 0 (decisões do challenge são feature-scoped — falham C1)
- ADRs sugeridos para criação: 0

---

## Execution Run — 2026-06-02

- [run] executor resolvido: `go-backend-implementer` (origem: descoberta interativa)
- [run] executor_discipline injetado (fonte: references/executor-discipline.md)
- [run] reorder: T5 antes de T3 (decisão do usuário). Motivo: CT-T3-004/005 referenciam `service.EmailSender`, criada em T5; T3 não compila/passa QA antes de T5. Ordem revisada: [T1∥T2∥T4] → T5 → T3 → T6 → T7 → [T8∥T9] → T10. Conteúdo das tasks inalterado.

### Onda 1 — Gate 1 (QA)
- T2: APROVADO (5/5; add-only confirmado; build verde). → Tech Review.
- T4: APROVADO (7/7; 17 testes config verdes; table-driven legítimo). → Tech Review. RC-001 (repeated_assertion_shape) persistido.
- T1: REJEITADO (1 crítico, categoria=tests). Regressão: migração 000009 quebrou Steps(-N) em match/nationalteam down-tests. attempt=1. Modelo já opus (sem auto-escalate).

### Onda 1 — Gate 2 (Tech Review)
- T2: approved (sem problemas; add-only e ADR-0005 confirmados). → staged. CONCLUÍDA.
- T4: approved_with_observations. Débito anotado (não bloqueante, cleanup futuro):
  - [LOW][code_quality] config.go: predicado `isDevelopment := env=="dev"||env=="development"` em Load() duplica Config.IsDevelopment(). Sugestão: extrair `isDevelopmentEnv(env string) bool` e reusar nos 2 pontos. (Config não está construída no ponto de Load(), inlining defensável.) → staged. CONCLUÍDA.
- T1: correção aplicada (3 down-tests Steps incrementados), suíte verde. Re-QA em andamento (attempt 2).
- T1: re-QA APROVADO (suíte completa verde) + Tech Review approved. → staged. CONCLUÍDA (2 tentativas QA, 0 retry Tech Review). Memória lazy deletada.

### T5 — Gates (após onda 1, reordenado antes de T3)
- QA: APROVADO (8/8). Anti-mock-driven (bcrypt), anti-enumeração, best-effort, determinismo confirmados. Desvios de escopo (module.go stopgap noopEmailSender, +*zap.Logger no construtor, auth_handler_test.go callers) julgados mecanicamente forçados/justificados. RC-001/RC-002 persistidos.
- Tech Review: approved_with_observations. Débito anotado (não bloqueante):
  - [LOW][security] auth_service.go generateTempPassword: viés de módulo (byte%56) — distribuição levemente não-uniforme. Impacto desprezível (~90 bits, bcrypt, TTL 15min). Fix opcional: rejection sampling / crypto/rand.Int.
  - [LOW][code_quality] auth_service.go: texto "15 minutos" no corpo do e-mail duplica recoveryTTL (risco de drift se TTL mudar).
- HANDOFF T9: substituir o stopgap noopEmailSender em module.go pelo bind real (Resend/Noop por config); propagar campos temp no userRepositoryAdapter (GetUserByEmail/ID) — necessário p/ T6 funcionar ponta-a-ponta. → staged. CONCLUÍDA.

### T3 — Gate 1 (QA)
- QA: APROVADO_COM_OBSERVACOES (5/5 CAs; 0 crítico/alto; 4 médios não-bloqueantes). build-all verde (resend-go/v2 pure-Go), CA-09 confirmada (body nunca logado). RC-001/RC-002 persistidos. Débito anotado:
  - [MED][tests] email_test.go:83 CT-T3-002 `_ = s.Send(...)` descarta retorno; add `require.NoError`.
  - [MED][tests] email_test.go:204 CT-T3-009 `assert.True(... || sendErr != nil)` tautológico; asserção específica.
  - [MED][code_quality] resend.go:36 `newResendSenderWithClient` test-only em arquivo de produção (Iron Law #6); mover p/ export_test.go.
  - [MED][tests] AP-26 CT-T3-001 vs CT-T3-002 (spec designou como par negativo autônomo; consolidar table-driven OU diferenciar com require.NoError).

### T3 — Gate 2 (Tech Review) + correção
- Tech Review attempt 1: partial (1 high P1 + 3 med + 1 low). Correção: P1 (asserção falsificável), P2 (test-only → email_test.go), P4 (go mod tidy). P3 (nomes pt-BR) e P5 (CT-002) NÃO corrigidos por decisão do orquestrador (consistência T1/T4/T5; spec). requires_qa_revalidation=true.
- Re-QA attempt 2: APROVADO (9/9). Re-Tech Review attempt 2 (opus, retry escala): approved. → staged. CONCLUÍDA (2 tentativas). Memória lazy deletada.
- RC convention_drift (nomes de teste pt-BR vs ADR-0005) registrado para resolução de doutrina.

### T6 — Gates
- QA: APROVADO (6/6). CT-009 mutation-killing (Before exclui ==now); anti-enumeração (msg genérica + dummy hash) preservada; anti-mock-driven (bcrypt real); determinismo (clock avançável). RC-001/RC-002 persistidos.
- Tech Review: approved_with_observations. Débito anotado (não bloqueante):
  - [LOW][testability] auth_service_test.go CT-008 usa status.FromError(err) descartando ok, enquanto CT-009/CT-024 usam status.Code(err). Inconsistência estilística no mesmo delta. (CT-008 precisa da msg, justifica parcialmente.)
- → staged. CONCLUÍDA.

### T7 — Gates
- QA: APROVADO (6/6). Validação estrita (3 condições + UpdatePassword não chamado em rejeição), ordem CA-12, CT-036 anti-vazamento (dupla NotContains), anti-mock-driven (bcrypt real), anti-IDOR. RC-001 persistido. Débito:
  - [LOW][code_quality] auth_service_test.go: literal "TempXxx-RECOVERY-9!" duplicado em tempPassword (l.664) e changeTempPassword (l.886).
- Tech Review: approved (sem problemas). → staged. CONCLUÍDA.

### Fase 3 — Gates (lote paralelo T8 ∥ T9)
- T9: QA APROVADO (6/6; CT-W3 mutation-killing confirmado empiricamente; fail-open OK — RequestPasswordRecovery público via constante, ChangePassword/GetMe protegidos; sentinela ponto único). Tech Review approved. → staged. CONCLUÍDA.
- T8: QA attempt 1 REJEITADO (1 ALTO + security_flag auth_context_forgeable_via_exported_setter): ContextWithSubject exportado em interceptor/auth_jwt.go (produção auth, test-only, Iron Law #6, fora da §5.2). Correção: revertido o interceptor; testes via caminho real do interceptor + TokenManager fake. last_severity=high → executor escalado para opus. Re-QA em andamento (attempt 2).
- T8: re-QA attempt 2 APROVADO (6/6). Iron Law #6 resolvido (interceptor revertido, diff vazio), sem security_flag, anti-IDOR intacto. gates:[qa] (deviação revertida → mapper puro, sem Tech Review). → staged. CONCLUÍDA (2 tentativas QA). Memória lazy deletada.

### T10 — Gate 1 (QA, único gate)
- QA: APROVADO (6/6; suíte completa verde 23 pacotes). CT-032 mutation-killing (temp extraída do body capturado, não hardcoded); anti-enumeração (CT-028); public/protected (CT-029/030, ordem protovalidate→authJWT entendida); sem time.Sleep (canal CapturingEmailSender). RC-001/RC-002 persistidos. Observação não-bloqueante:
  - [OBS] CT-029 loga "database is closed" — goroutine de background do RequestPasswordRecovery tenta persistir após teardown do server; teste passa (asserta só no retorno gRPC). Risco baixo de flakiness; não depende do resultado da goroutine.
  - [OBS] national_team_e2e_test.go:16 comentário "CT-032" refere-se a CT de outra feature (sobreposição de numeração entre features) — inofensivo, rastreabilidade futura.
- → staged. CONCLUÍDA.

### Fim do run
- [run] rule_candidates: 17 sinais persistidos em rule-candidates.md (qa=10, staff=1, orquestrador/pre_refinement=6).
- Critérios de Conclusão: make test verde (todos os pacotes) + make build-all verde (4 targets CGO off: linux/amd64, linux/arm64, darwin/arm64, windows/amd64). 10/10 tasks, 0 bloqueadas.
- Mudanças staged (não commitadas — usuário commita). Reorder consciente: T3 após T5.
