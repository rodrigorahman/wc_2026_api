# QA / Tech Review Observations — usuario-multiplas-selecoes/v1

> Log de decisões de pipeline (resolução de modelo/gates, retry classification, escalações).

## Run 2026-05-29 — TC-001

- [run] executor resolvido: `go-backend-implementer` (origem: descoberta interativa)
- [run] executor_discipline injetado (fonte: agent-spec-executor-discipline.md, versão: d11bfc0)
- [TC-001] executor: opus (declarado no frontmatter) | gates: [qa, tech_review] (declarado)
- [TC-001] diff_touches_critical_path = true (db_migrations + auth + api_contracts) | task_risk = high
- [TC-001] qa_model = opus (escalado: critical_path + risk=high) | tech_model = opus (escalado: critical_path + risk=high)
- [TC-001] base_sha = 9401a80

### TC-001 — Gate 1 (QA) APROVADO (nota 9)
- 0 críticos, 0 altos, 0 médios, 1 baixo.
- BAIXO-001 (data_handling): down-migration recria `selecao_id NOT NULL DEFAULT ''` — débito de reversão lossy N:N→1:1; aceitável em greenfield (§4.2). Registrado para cleanup futuro.
- Falha `internal/auth/token/TestGenerate_ClaimsAndExp`: não-regressão (pacote não tocado, time-bomb wall-clock pré-existente, §4.2 fora de escopo).
- Arquivo extra fora da §8.2: `internal/testutil/migrator.go` (test-only, justificado para CT-053 down) — encaminhado ao Tech Review para avaliação de escopo.

### TC-001 — Gate 2 (Tech Review) APPROVED_WITH_OBSERVATIONS
- 0 críticos, 0 altos. Sem re-loop (débito-controlado: medium/low são observações).
- P1 (medium, scope_deviation): `internal/testutil/migrator.go` fora da §8.2. Sugestão do TR: atualizar §8.2 retroativamente listando-o como entregável de teste. Test-only, pure-Go, sem dependência nova (golang-migrate já é dep de produção). Aceito.
- P2 (low, code_quality): seam `compareFunc` no AuthService com docstring levemente enganoso (variação só em teste, não em produção). Débito estilístico.
- P3 (low, best_practices): down-migration `DEFAULT ''` sentinela FK-inválido — trade-off intrínseco do revert N:N→1:1. Débito documental.
- ADRs consultadas: 0001, 0002, 0003, 0004.

[run] rule_candidates: 2 sinais persistidos em rule-candidates.md (qa=1 repeated_fixture, staff=1 scope_deviation, orquestrador=0)
