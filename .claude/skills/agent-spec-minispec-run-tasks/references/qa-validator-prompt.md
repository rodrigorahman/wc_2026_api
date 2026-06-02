# Prompt do Gate 1 — agent-spec-qa-validator (FASE 3.3)

> Referência consumida por `SKILL.md` da skill `agent-spec-minispec-run-tasks`.
> Leia este arquivo **antes de invocar o Gate 1** (FASE 3.3).
> Contém o prompt completo do `agent-spec-qa-validator` e os passos de preparação (3.1, 3.2, 3.3, 3.4, 3.5).
> Use exatamente o texto da seção "Disparar o QA" como `prompt` no `Agent({...})`.
>
> **Pré-requisito de leitura**: [`config.md`](config.md) — necessário para resolver `qa_model` (regras de escalação `agent-spec-qa-validator`) e os campos de `qa_summary_fields`. Leia `config.md` **antes** deste arquivo se ainda não o fez.

---

## FASE 3 — Gate 1 — QA (agent-spec-qa-validator)

> **Único gate que executa testes.**
>
> **Pré-verificação**: se `gates: none` → não invoque QA. Se `gates: [qa]` ou `[qa, tech_review]` → siga.

### 3.1 Preparar arquivos para o QA (lista enxuta)

Inclua:
- **Task implementada** (path via `minispec.tasks.dir` + `minispec.tasks.pattern`)
- **Arquivos criados/modificados** pelo executor (código de produção tocado pela task)
- **Arquivos de teste** criados/modificados (padrão da stack)

> `base_sha` e `executor_summary` viajam **inline em `instrucoes`** (3.2), não em `arquivos[]`.

**NÃO inclua** (evita duplicar contexto e tokens):
- `CLAUDE.md` e `.claude/rules/*.md` (já no contexto do subagente)
- SCOPE e INTENT completos — passe apenas os **paths** em `instrucoes` como referência opcional (o QA consulta sob demanda se precisar)

### 3.2 Preparar `instrucoes` para o QA

1. **ID e nome** da task (contexto)
2. **Contexto da execução** (inline — substitui o execution-summary):
   ```
   - base_sha: <SHA capturado em 2.1>
   - Sumário do executor:
     <output enxuto de 4-6 linhas retornado pelo executor>
   ```
3. **Critérios de conclusão** da task — o QA valida CADA critério
4. **Testes definidos** na task — o QA executa e verifica
5. **Rastreabilidade de testes (BLOQUEANTE)**: lista de IDs (CT-01, CT-02, ...) da seção de Testes. Instrução literal:
   > "Cada CT DEVE ter teste correspondente implementado no código. Testes ausentes/vazios/skip/todo para CTs exigidos = REJEITADO na camada COMPLETUDE."
6. **Comando de teste**: o QA resolve pela precedência de descoberta de stack — (1) rule `.claude/rules/agent-spec-testing-stack.md` se existir; (2) CLAUDE.md/rules; (3) manifest do projeto (`package.json`, `pyproject.toml`, `go.mod`, `Cargo.toml`, `Gemfile`, `pubspec.yaml`, `pom.xml`, `build.gradle`, etc.), scripts e CI — e executa o canônico.
7. **Caminhos de referência opcionais**: `minispec.scope.path` e `minispec.intent.path` — consulta sob demanda.
8. **Economia de Leitura**: "Não leia arquivos desnecessários ao escopo desta task."

### 3.3 Disparar o QA

Resolva `qa_model` (ver `references/config.md` §4 da Lógica de Seleção de Modelo):

```
Agent(
  subagent_type = "agent-spec-qa-validator",
  model         = qa_model,             # sonnet | opus
  description   = "QA validar task TN",
  prompt        = <prompt abaixo>
)
```

Prompt:

```
Você foi invocado com os seguintes parâmetros:

1. **arquivos**: [lista de caminhos preparada em 3.1]
2. **instrucoes**: [conteúdo preparado em 3.2]

OBRIGATÓRIO: Antes de produzir o JSON final:

1. Invoque a skill `agent-spec-testing-best-practices` (Skill(skill="agent-spec-testing-best-practices")) e aplique a Camada 5 (Qualidade dos Testes) usando `references/antipadroes.md` como checklist. Cada antipadrão detectado em arquivos de teste tocados pela task vira um item em `problemas.*` com o campo `smell` preenchido (nome canônico). Severidade do antipadrão determina veredito conforme política débito-controlado (críticos/altos bloqueiam; médios/baixos viram observações). Popule também `testing_smells.red_flags_detectadas[]`, `mock_budget_violado` e `determinismo_observado`.

2. **Aplique a Camada 6 (ADR Compliance Light)** — leia `docs/adr/INDEX.md` (ou liste `docs/adr/*.md`), identifique ADRs ativas grep-detectáveis e cruze com os arquivos tocados pela task. Violações claras viram `problemas.*` com `categoria: "adr_compliance"`. Popule `adr_compliance.violacoes_grep_detectaveis[]`.

3. **Detecte duplicatas semânticas (AP-26)** — para cada par de testes nos arquivos tocados, compare tupla `(test_name_normalizado, alvo_chamado, parametros_chave, resultado_esperado)`. Coincidência em ≥ 3 dos 4 campos sem justificativa → reporte como `MÉDIO` em `problemas.medios[]` com `categoria: "code_quality"`. Não confundir com table-driven (UM teste parametrizado é OK).

4. **Categoria obrigatória** em cada item de `problemas.*` — usar valores canônicos da rule `.claude/rules/agent-spec-workflow-rules.md` (`architecture`, `security`, `tests`, `logic`, `data_handling`, `error_handling`, `performance`, `concurrency`, `code_quality`, `naming`, `style`, `documentation`, `dead_code`, `imports`, `adr_compliance`). Default conservador → `revalidation_required` quando incerto.
```

**IMPORTANTE**: preserve o JSON completo retornado pelo QA. Será usado:
- Sumário mínimo → input do Tech Review (4.2)
- Em rejeição → memória lazy (3.5)

### 3.4 Interpretar o resultado do QA

> **Política débito-controlado**: bloqueia o que é risco real (críticos e altos); anota o que é débito de manutenibilidade (médios e baixos). O loop de correção só dispara quando há crítico ou alto. Médios/baixos viajam adiante registrados em `qa-observations.md` para cleanup futuro.

| Veredito | Críticos+Altos | Médios+Baixos | Ação |
|---|---|---|---|
| `APROVADO` | 0 | 0 | QA aprovado → avançar para Gate 2 |
| `APROVADO_COM_OBSERVACOES` | 0 | ≥ 1 | QA aprovado com débito anotado → avançar para Gate 2; registrar médios/baixos em `qa-observations.md` |
| `REJEITADO` | ≥ 1 | qualquer | Enviar críticos+altos como bloqueantes ao executor (3.5); médios/baixos como observações opcionais |

> **Sinal `stack_discovery.discovery_needed: true`** (não afeta veredito): o QA não resolveu um detalhe **não-derivável do código** (ex.: framework E2E não padronizado, política de cobertura). Recomende ao usuário rodar **`/agent-spec-testing-stack-bootstrap`** — ele descobre a stack do código, pergunta só o não-derivável e gera `.claude/rules/agent-spec-testing-stack.md`. A partir daí o QA resolve a stack automaticamente. Não bloqueie o pipeline por esse sinal.

### 3.5 Loop de correção QA (memória lazy)

Se rejeitado:

1. **Monte/atualize a memória lazy** no path via `shared.temp_memory.dir` + `shared.temp_memory.pattern` (ex.: `docs/specs/features/{feature}/{version}/tasks/.tmp/T1.md`):

   ```markdown
   # Memória temporária — T[N]
   > Criada em [timestamp]. Deletada ao aprovar; expira em 24h.

   ## attempt_count
   [N — incremente a cada retry]

   ## last_severity
   [low|medium|high|critical — do último JSON]

   ## Sumário do executor
   [output enxuto de 4-6 linhas que o executor produziu]

   ## JSON QA Validator
   ```json
   [JSON completo do 3.3]
   ```

   ## Arquivos tocados (`git diff --stat`)
   [saída de `git diff <base_sha> --stat`]

   ## Paths
   - Criados: [lista]
   - Modificados: [lista]
   - Testes: [lista]
   ```

2. **Extraia TODOS os problemas do JSON do QA — sem filtro por severidade**:
   - `problemas.criticos[]` (titulo, descricao, arquivo, linha, correcao_sugerida)
   - `problemas.altos[]`
   - `problemas.medios[]`
   - `problemas.baixos[]`
   - `observacoes[]`
   - `testes_executados.detalhes_falhas[]`
   - `criterios_falhos[]` (CAs com `status` `FALHOU` ou `PARCIAL`)

   > **Zero-débito**: NÃO filtre por severidade. A task não pode ser concluída com qualquer dívida técnica reportada.

3. **Aplique auto-escalonamento de modelo** (ver `references/config.md` §3 da Lógica de Seleção). Logue se escalou.

4. **Monte o prompt de correção** para o executor:

   ```
   A task [ID] foi REJEITADA pelo QA. Leia a memória lazy em [path do arquivo] antes de corrigir.

   ## Problemas Críticos
   [lista de problemas.criticos]

   ## Problemas Altos
   [lista de problemas.altos]

   ## Problemas Médios
   [lista de problemas.medios]

   ## Problemas Baixos
   [lista de problemas.baixos]

   ## Testes que Falharam
   [lista de detalhes_falhas]

   ## Critérios de Aceite não Atendidos
   [lista com status FALHOU ou PARCIAL]

   Corrija APENAS os problemas listados acima. Não expanda escopo.

   Para CADA problema, antes de editar escreva uma linha `CAUSA-RAIZ: <por que o teste ou o código estava errado>`. Correção que apenas faz o gate passar sem atacar a causa — inverter uma flag, enfraquecer a asserção, renomear — será RE-REPROVADA. Se o problema é asserção fraca, mock-driven ou teste oco: reescreva a asserção para validar o comportamento observável real (não ajuste o valor do mock nem inverta booleanos). Se algum problema já havia sido reprovado na tentativa anterior, a correção anterior foi insuficiente — ataque a origem, não o sintoma.

   Após corrigir, execute os testes para garantir que passam.

   Arquivos a corrigir:
   [lista de arquivos dos problemas]
   ```

5. **Dispare o executor** com `effective_model` (escalado se aplicável).
6. **Re-valide com o QA** (volte ao 3.3). Atualize `attempt_count` e `last_severity` na memória lazy.
7. **Limite máximo: 3 tentativas TOTAIS** por task (compartilhado com Tech Review — 4.4).

**Ao aprovar AMBOS os gates**: delete a memória lazy `T{N}.md` (se foi criada por rejeição) — `cleanup_on_approval: true`. **Não há mais execution-summary em disco** (substituído por inline no prompt — ver 2.4 da SKILL.md).
