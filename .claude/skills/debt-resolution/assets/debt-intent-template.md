# INTENT — Cleanup de Débitos Técnicos · {{feature}} · {{version}}

> **Tipo**: Versão de débitos (gerada por `/debt-resolution`).
> **Origem**: `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md`
> **Variante**: {{variant}}
> **Data**: {{data}}

<!-- LLM-ONLY: Esta versão NÃO adiciona funcionalidade. É exclusivamente cleanup
     de débitos `MEDIO`/`BAIXO` anotados pela política débito-controlado dos gates
     durante a execução de `{{parent_version}}`. Não inclua user stories nem
     personas — débito técnico não tem stakeholder externo. -->

## 1. Identificação

- **Feature**: {{feature}}
- **Versão**: {{version}}
- **Versão pai**: {{parent_version}} (feature original)
- **Variante**: {{variant}} (herdada de {{parent_version}})
- **Origem dos débitos**: `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md`
- **Tipo de operação**: cleanup técnico (zero feature nova)

---

## 2. Objetivo

Resolver **{{count_selecionados}} débitos técnicos** acumulados na execução de `{{parent_version}}`, classificados como aceitáveis para passar pelos gates (severidade `MEDIO`/`BAIXO`, categorias `code_review_only`) mas que prejudicam manutenibilidade, legibilidade ou propagam anti-padrões se deixados sem ação.

A versão é gerada via skill `/debt-resolution` que:

1. Coletou {{count_coletado}} débitos elegíveis de `qa-observations.md` da `{{parent_version}}`.
2. Submeteu ao agente especialista da stack (`{{agent_name}}`) para classificação binária (`recomendado_corrigir` / `perfumaria`).
3. Apresentou a classificação ao usuário, que selecionou {{count_selecionados}} para cleanup nesta rodada.
4. Os demais {{count_ignorados}} débitos ficam registrados em `scope.md §2 (Fora do escopo)` para auditoria — podem ser revisitados em uma futura `v{{N+2}}-debits/`.

---

## 3. Resultado esperado

Após execução desta versão via `/minispec-run-tasks`:

- Cada débito selecionado vira **1 task atômica** em `tasks/T{n}.md` com `gates: [qa]`.
- Suíte de testes da feature continua passando (cleanup não muda comportamento).
- `qa-observations.md` da `{{parent_version}}` ganha entrada final marcando débitos resolvidos.
- Diff esperado: pequeno (delete de funções duplicadas, rename de variáveis, remoção de dead code, ajuste de imports/docs).

---

## 4. Critérios de sucesso

- [ ] Todas as {{count_selecionados}} tasks aprovadas pelo Gate 1 (QA).
- [ ] Suíte de testes da feature inteira passa sem regressão.
- [ ] Nenhum arquivo fora do escopo de cada débito modificado.
- [ ] `qa-observations.md` da `{{parent_version}}` registra os débitos resolvidos.

---

## 5. Premissas

- A `{{parent_version}}` está concluída (todas as tasks principais aprovadas pelos gates).
- Os débitos coletados refletem o estado real após a última execução de `/minispec-run-tasks` na `{{parent_version}}`.
- Tasks de débito são independentes entre si (paralelizáveis pelo orquestrador respeitando guards de paths disjuntos).

---

## 6. Fora do escopo

- **Funcionalidade nova**: zero. Esta versão é cleanup puro.
- **Refactor arquitetural**: débitos que exigiriam ADR ou mudança de padrão **não** entram aqui — seriam classificados como `revalidation_required` pelos gates e teriam bloqueado o pipeline da `{{parent_version}}`. Se algum aparece, é bug do gate.
- **Tech Review profundo**: tasks de débito têm `gates: [qa]` por default (categoria `code_review_only`).

---

## 7. Próximo passo

```
/minispec-run-tasks docs/specs/features/{{feature}}/{{version}}/task_plan.md
```

Tempo estimado total: ~{{tempo_estimado_min}} minutos (soma dos custos individuais).
