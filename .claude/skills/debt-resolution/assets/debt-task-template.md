# T{{n}} — Resolver D-{{id_debito}}: {{titulo}}

## 1. Identificação

- **ID**: T{{n}}
- **Nome**: Resolver D-{{id_debito}} — {{titulo}}
- **Feature**: {{feature}}
- **Versão**: {{version}}
- **Parent task**: T{{origem_task_parent}} (de {{parent_version}})
- **Débito origem**: D-{{id_debito}} ({{categoria}}, {{severidade}})
- **Source**: debt-resolution
- **Status**: A Fazer

### Frontmatter

- **model**: sonnet
- **risk**: {{risk}}
- **gates**: {{gates}}
- **source**: debt-resolution
- **debito_origem**: D-{{id_debito}}

---

## 2. Objetivo

Resolver o débito técnico D-{{id_debito}} aplicando a correção exata sugerida pelo gate original, **sem expandir escopo** e **sem regressão** na suíte de testes.

---

## 3. Contexto do Débito

**Origem**: task `T{{origem_task_parent}}` da `{{parent_version}}` — registrado em `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md` (linha {{origem_linha}}).

**Descrição original**: {{descricao}}

**Categoria**: `{{categoria}}` ({{severidade}})

**Classificação do especialista**: `{{classificacao_llm}}`
**Justificativa**: {{justificativa_llm}}
**Custo estimado**: ~{{custo_estimado_min}}min
**Risco de regressão**: `{{risco_regressao}}`

---

## 4. Detalhes de Implementação

- [ ] {{correcao_sugerida}}

> Não há subtarefas adicionais. Cleanup é cirúrgico.

---

## 5. Arquivos Impactados

### 5.1 Criar

_Nenhum._

### 5.2 Modificar

- `{{arquivo}}`{{#if linha}} (foco na linha {{linha}}){{/if}}

### 5.3 Referência

- `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md` — débito original.
- `docs/specs/features/{{feature}}/{{parent_version}}/tasks/T{{origem_task_parent}}.md` — task que gerou o débito.

---

## 6. Testes

**N/A — task é cleanup técnico**. Não cria testes novos.

O Gate 1 (QA) DEVE executar a **suíte completa** da feature após a modificação. Comportamento esperado:

- ✅ Todos os testes existentes continuam passando.
- ❌ Qualquer teste regredindo → task rejeitada (sinal de que o débito carregava lógica relevante e não pode ser corrigido isoladamente).

---

## 7. Critérios de Conclusão

- [ ] Correção aplicada exatamente como descrita em §4 — sem expansão de escopo.
- [ ] Diff afeta APENAS o arquivo listado em §5.2.
- [ ] `go test ./...` (ou comando da stack) passa 100% sem regressão.
- [ ] Gate 1 (QA) aprova.

---

## 8. Guardrails de Execução

### DEVE

- Ler `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md` ao redor da linha {{origem_linha}} para entender o contexto original do débito.
- Aplicar a `correcao_sugerida` literalmente.
- Rodar a suíte completa de testes da feature antes de retornar a task como concluída.
- Reportar regressão imediatamente — NÃO tente "consertar a regressão também", isso vira refactor não autorizado.

### NÃO DEVE

- NÃO expandir escopo para "outros débitos parecidos no mesmo arquivo".
- NÃO refatorar funções/módulos não mencionados no débito.
- NÃO adicionar testes novos (suíte existente é o oráculo).
- NÃO alterar comportamento observável — se o cleanup mudar resposta de API/output de função, a correção está errada.

---

## 9. Checklist Final

- [ ] Correção aplicada
- [ ] Suíte completa passa
- [ ] Diff cirúrgico (1 arquivo)
- [ ] Gate 1 aprovou
- [ ] Revisada

---

## 10. Notas / Observações

_Vazio. Tasks de débito não geram débito novo (seria meta-débito sem fim)._

> Se durante execução você identificar OUTROS débitos relacionados não listados, **NÃO os resolva** — registre em `docs/specs/features/{{feature}}/{{parent_version}}/qa-observations.md` como nota e siga adiante. Eles entrarão numa eventual `v{{N+2}}-debits/`.
