# TASK PLAN — Cleanup de Débitos · {{feature}} · {{version}}

## 1. Identificação

- **Feature**: {{feature}}
- **Versão**: {{version}}
- **Versão pai**: {{parent_version}}
- **Variante**: {{variant}}
- **Intent**: `docs/specs/features/{{feature}}/{{version}}/intent.md`
- **Scope**: `docs/specs/features/{{feature}}/{{version}}/scope.md`
- **Origem**: gerado por `/debt-resolution` em {{data}}
- **Agente especialista (classificação)**: `{{agent_name}}`
- **Status**: A Fazer

---

## 2. Objetivo Técnico

Resolver {{count_selecionados}} débitos técnicos atômicos via tasks `gates: [qa]` (cleanup é `code_review_only`). Cada task toca exatamente 1 débito; suíte existente é o oráculo de regressão.

---

## 3. Macro-Fases

- **Fase 1 — Cleanup**
  - Objetivo: aplicar correção pontual de cada débito.
  - Tasks: T1 .. T{{count_selecionados}} (todas paralelizáveis — guards do orquestrador decidem se vão em lote ou sequencial).

> Por que 1 fase só: débitos são independentes. Não há ordem técnica obrigatória.

---

## 4. Lista de Tasks

| ID  | Nome | Arquivo da task | Débito original | Custo (min) | model | risk | gates | Paralelo? | Status |
|-----|------|-----------------|-----------------|-------------|-------|------|-------|-----------|--------|
{{#each tasks}}
| T{{n}} | {{nome_curto}} | [T{{n}}](tasks/T{{n}}.md) | D-{{id_debito}} ({{categoria}}) | ~{{custo_min}} | sonnet | {{risk}} | {{gates}} | Sim | A Fazer |
{{/each}}

---

## 5. Ordem de Execução

```
Fase 1 (paralelo, respeitando guards do orquestrador):
  T1 ─┐
  T2 ─┤
  T3 ─┤  → orquestrador detecta lote paralelizável até MAX_PARALLEL=4
  T4 ─┤    com guards de paths disjuntos e dep transitiva textual.
  ... ┘    Falha em guard → fallback sequencial automático.
```

### Grafo de Dependências

Nenhuma dependência entre tasks (são independentes por construção).

| Task | Depende de | Pode Rodar em Paralelo? |
|------|------------|-------------------------|
{{#each tasks}}
| T{{n}} | — | Sim |
{{/each}}

---

## 6. Arquivos / Áreas Impactadas (consolidado)

| Arquivo | Tasks que tocam | Categorias |
|---------|-----------------|------------|
{{#each arquivos_consolidados}}
| `{{arquivo}}` | {{lista_tasks}} | {{categorias}} |
{{/each}}

> **Atenção do orquestrador**: se 2+ tasks na mesma onda paralela tocam o mesmo arquivo, o guard "paths disjuntos" da rule `Execução Paralela de Tasks` força fallback para sequencial para evitar colisão de `git add`.

---

## 7. Critérios de Conclusão Geral

- [ ] Todas as {{count_selecionados}} tasks com Status `Concluído`.
- [ ] Suíte de testes da feature passa sem regressão (Gate 1 valida em cada task).
- [ ] Nenhum diff em arquivos fora da seção 6 acima.
- [ ] `qa-observations.md` da `{{parent_version}}` registra cleanup concluído.
- [ ] `minispec_state.yaml` desta versão marca `execution: completed`.

---

## 8. Notas para a LLM Executora

### Convenções desta versão

- **NÃO** criar testes novos. Tasks são cleanup.
- **NÃO** refatorar fora do escopo do débito específico.
- **NÃO** "aproveitar a oportunidade" para corrigir débitos não listados.
- **SIM**: aplicar exatamente a `correcao_sugerida` da task — escopo cirúrgico.
- **SIM**: executar `go test ./...` (ou equivalente da stack) após cada modificação para confirmar zero regressão antes de retornar a task.

### Frontmatter de cada task

```markdown
- model: sonnet
- risk: low
- gates: [qa]
- source: debt-resolution
- debito_origem: D-XXX
- task_origem_parent: T{{N}}
```

### Saída esperada do executor

Formato padrão de output enxuto:

```
✅ T{N} — Resolver D-XXX: <título curto> /
  Arquivos: 1 modificado /
  Testes: <regrediu? 0 sim, todos passando> /
  Pendências: nenhuma
```
