# Guardrails Invioláveis + Checklist Final

> Referência consumida por `SKILL.md` da skill `agent-spec-minispec-run-tasks`.
> Leia este arquivo:
> - **No início da execução** (após FASE 0) para internalizar os DEVE / NÃO DEVE.
> - **Antes de encerrar** (após FASE 5) para validar o checklist final.

---

## Guardrails Invioláveis

### DEVE

1. **SEMPRE delegar** ao subagente `agent_name` — coordenador NUNCA implementa diretamente.
2. **Executar SEQUENCIALMENTE** — uma task por vez, na ordem das dependências.
3. **SEMPRE validar com QA** após cada task (exceto `gates: none`) — nenhuma task avança sem aprovação do QA.
4. **SEMPRE validar com Tech Review** após QA (exceto `gates: none` ou `[qa]`) — nenhuma task concluída sem aprovação do Tech Review.
5. **Resolver `model`/`risk`/`gates`** do frontmatter da task antes de invocar executor.
6. **Aplicar auto-escalonamento** em retry (sonnet→opus[xhigh] após 2 tentativas ou severity=high).
7. **Capturar `base_sha`** por task antes do executor (2.1).
8. **Passar `base_sha` + sumário do executor INLINE** no prompt do QA e do Tech Review (2.4 — sem arquivo intermediário).
9. **Preservar JSON completo do QA** para retry e sumário do Tech Review.
10. **Stage real (`git add`)** apenas após Tech Review aprovar (4.5).
11. **Cleanup de memória** ao aprovar AMBOS os gates.
12. **Cleanup idempotente** (>24h) no início da execução.
13. **Logar resolução de modelo/gates** no terminal antes de invocar executor/gates.
14. **Injetar o bloco "Disciplina do Executor (Iron Rules)"** verbatim no prompt de TODO executor invocado — fonte: `references/executor-discipline.md` (entre os marcadores `<<<EXECUTOR_DISCIPLINE` … `EXECUTOR_DISCIPLINE>>>`). O sub-agente NÃO herda essa referência via system-prompt (ela vive sob demanda em `references/`, não em `.claude/rules/`); sem o bloco no prompt, as 4 Iron Rules (Pense antes de codar / Simplicidade primeiro / Cirúrgico / Goal-driven) não chegam ao executor.

### NÃO DEVE

1. **NUNCA implementar** uma task diretamente — sempre delegue.
2. **Tasks em paralelo são permitidas APENAS** quando passam nos guards da rule `agent-spec-workflow-rules.md` → "Execução Paralela de Tasks" (independência no DAG + disjunção de símbolo + paths disjuntos + sem arquivo de alta contenção compartilhado + lote ≤ MAX_PARALLEL=4). Qualquer guard sem prova de independência → fallback determinístico para sequencial. O flag derivado é **re-verificado** — nunca confie cego na coluna.
3. **NUNCA lance QA e Tech Review da MESMA task em paralelo**. Entre tasks de um lote paralelo, pipelines isolados PODEM rodar em paralelo (cada um QA→TR sequencial internamente).
4. **NUNCA usar Haiku no executor** — rejeite com erro claro se frontmatter declarar.
5. **Política débito-controlado em retry**: envie ao executor APENAS problemas com `severity` `critical` ou `high` como bloqueantes; problemas `medium`/`low` vão como "Observações" opcionais no mesmo prompt (não exigem correção no ciclo). Esses médios/baixos ficam registrados em `qa-observations.md` para cleanup futuro.
6. **NUNCA usar paths hardcoded** — sempre resolva via templates do CLAUDE.md.
7. **NUNCA alterar INTENT, SCOPE ou criar novas tasks** sem o usuário pedir.
8. **NUNCA continuar após 3 tentativas falhas** — escale ao usuário.
9. **NUNCA commitar** ao final do Tech Review aprovar — apenas `git add`. O usuário commita.
10. **NUNCA enviar JSON completo do QA ao Tech Review** — apenas o sumário mínimo (`qa_summary_fields`).

---

## Checklist Final (orquestrador, antes de encerrar)

- [ ] Repositório git verificado no início
- [ ] Cleanup idempotente de memória stale executado
- [ ] `minispec_state.yaml` atualizado para `execution: in_progress` no início
- [ ] Cada task processada respeitando o algoritmo de "Execução Paralela de Tasks" (lote paralelo com guards OU sequencial); gates dentro de cada task continuam SEQUENCIAIS (QA → TR)
- [ ] Bloco "Disciplina do Executor (Iron Rules)" carregado de `references/executor-discipline.md` no início e injetado no prompt de cada executor
- [ ] `model`/`risk`/`gates` resolvidos por task com logs no terminal
- [ ] `base_sha` capturado por task
- [ ] Execution summary criado após cada executor concluir
- [ ] Sumário mínimo do QA enviado ao Tech Review (não JSON completo)
- [ ] Memória lazy criada apenas em rejeição
- [ ] Stage (`git add`) feito apenas após Tech Review aprovar
- [ ] Memória lazy `T{N}.md` deletada ao aprovar (se foi criada por rejeição)
- [ ] Tasks bloqueadas escaladas ao usuário (após 3 tentativas)
- [ ] `task_plan.md` (tabela + grafo + critérios gerais) atualizado ao final
- [ ] `minispec_state.yaml` atualizado para `execution: completed` ao final
- [ ] Relatório Final apresentado ao usuário
