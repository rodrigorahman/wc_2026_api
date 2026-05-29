# Prompt do Gate 2 — agent-spec-staff-architecture-review (FASE 4)

> Referência consumida por `SKILL.md` da skill `agent-spec-minispec-run-tasks`.
> Leia este arquivo **antes de invocar o Gate 2** (FASE 4.2).
> Contém o prompt completo do `agent-spec-staff-architecture-review` e os passos de preparação (4.1, 4.2, 4.3, 4.4, 4.5, 4.6).
> Use exatamente o texto da seção "Disparar o Tech Review" como `prompt` no `Agent({...})`.
>
> **Pré-requisito de leitura**: [`config.md`](config.md) — necessário para resolver `tech_model` (regras de escalação `agent-spec-staff-architecture-review`) e os campos de `qa_summary_fields` (sumário mínimo do QA). Leia `config.md` **antes** deste arquivo se ainda não o fez.

---

## FASE 4 — Gate 2 — Tech Review (agent-spec-staff-architecture-review)

> **Pré-verificação**: se `gates: [qa]` → PULE este gate; marque concluída após QA aprovar.
>
> O Tech Review **NÃO re-executa testes** salvo se: `tocou_area_critica: true` E `escopo_testes != "SUITE_COMPLETA"`, OU se detectar violação `critical` em `architecture`/`security`.

### 4.1 Preparar contexto para o Tech Review

O agente staff **gera os diffs por conta própria** via Bash (`git diff <base_sha> -- <path>` por arquivo). O orquestrador NÃO mais executa `git diff` para captura — apenas prepara setup de estado.

#### 4.1.1 Visibilidade git dos paths NOVOS

1. Use `base_sha` da variável em memória (capturado em 2.1).
2. Colete `task_paths`: arquivos criados + modificados + arquivos de teste da task.
3. **Intent-to-add para untracked**: `git add -N -- <task_paths>` (torna NOVOS visíveis no `git diff` sem staged real). Ignore erros de já-adicionados. **Esta é a única operação git do orquestrador no Gate 2.**

#### 4.1.2 Sumário mínimo do QA

Extraia do JSON completo do QA (preservado em 3.3) **APENAS os campos** de `qa_summary_fields`:

```json
{
  "veredito": "APROVADO|APROVADO_COM_OBSERVACOES",
  "nota_qualidade": N,
  "security_flags": [...],
  "executou_testes": true|false,
  "escopo_testes": "SUITE_COMPLETA|PARCIAL|NAO_EXECUTADO",
  "tocou_area_critica": true|false,
  "escopo_declarado": {
    "fonte": "task_secao_arquivos|ausente",
    "arquivos_a_criar_faltantes": [],
    "arquivos_a_modificar_faltantes": [],
    "subtasks_sem_evidencia": []
  }
}
```

> O campo `escopo_declarado` vem da Camada 0 do QA (Completude de Escopo Declarado). Tech Review usa para confirmar que entregáveis estruturais não ficaram faltando. Se QA aprovou mas `escopo_declarado.fonte == "ausente"`, Tech Review precisa fazer a checagem de presença ele mesmo (ver agente).

> NÃO envie `problemas[]`, `files_reviewed[]`, `criterios_aceitacao[]` no prompt do staff. O agente gera o diff por conta própria; o sumário cobre a metadata.

#### 4.1.3 Categorizar paths (NOVOS vs MODIFICADOS)

Use a estrutura da task como fonte autoritativa:
- **NOVOS** = arquivos declarados como criados na task + arquivos de teste novos.
- **MODIFICADOS** = arquivos declarados como alterados + arquivos de teste pré-existentes alterados.

Identifique adicionalmente **paths em área crítica**: cruze `task_paths` com os globs de `critical_paths` (ver `references/config.md`) e liste à parte para sinalizar releitura recomendada ao staff.

NÃO execute `git diff` para categorizar — a categorização vem da task.

### 4.2 Disparar o Tech Review

Resolva `tech_model` (ver `references/config.md` §4 da Lógica de Seleção de Modelo).

```
Agent(
  subagent_type = "agent-spec-staff-architecture-review",
  model         = tech_model,            # sonnet | opus
  description   = "Tech Review task TN",
  prompt        = <prompt abaixo>
)
```

Prompt:

```
Realize a revisão técnica da task [ID] - [Nome da Task].

## Sumário do QA Validator (input metadata)
```json
[colar sumário mínimo extraído em 4.1.2 — APENAS os campos de qa_summary_fields]
```

## base_sha
[SHA capturado pelo orquestrador em 2.1]

## Sumário do executor (intenção)
[output enxuto de 4-6 linhas retornado pelo executor em 2.3]

## Como gerar os diffs (você mesmo executa via Bash)
Para cada path em "Arquivos NOVOS" + "Arquivos MODIFICADOS", rode em paralelo:
```bash
git diff <base_sha> -- <path>
```
- NOVOS: o diff retorna o conteúdo completo do arquivo — NÃO releia via Read.
- MODIFICADOS: o diff retorna apenas hunks alterados — Read sob demanda se contexto adjacente não bastar.
- NÃO use `--stat`, `..HEAD`, ou pipes para `head/tail`. Veja a seção FLUXO DE DIFF no seu contrato.

## Contexto da Task
- **Objetivo**: [conteúdo da task]
- **Critérios de Conclusão**: [critérios]

## Arquivos NOVOS (criados nesta task — `git diff` retorna conteúdo completo, NÃO releia via Read)
[lista de paths criados]

## Arquivos MODIFICADOS (alterados nesta task — diff retorna hunks parciais, Read sob demanda)
[lista de paths alterados]

## Arquivos em área crítica (releitura recomendada pelo staff)
[lista de paths em `critical_paths` que aparecem na task — pode estar vazia]

## Arquivos de Referência (para comparação de padrões — leia sob demanda)
[lista de arquivos de referência, se aplicável]

## Documentos de Referência (consultar sob demanda)
- Task completa: [path resolvido via minispec.tasks.dir + minispec.tasks.pattern]
- SCOPE: [path resolvido via minispec.scope.path]
- INTENT: [path resolvido via minispec.intent.path]

## ADRs
Consulte [path resolvido via adr.index_file] e leia ADRs específicas relacionadas aos paths tocados.

Valide (sobre o que mudou nos diffs que você gerar):
1. Conformidade arquitetural (camadas, fluxo de dependência, separação de responsabilidades)
2. Boas práticas de desenvolvimento (clean code, coesão, acoplamento, complexidade)
3. Qualidade de código (nomenclatura, legibilidade, duplicação, gambiarras)
4. Aderência aos padrões do projeto (convenções, nomenclatura, estrutura, `.claude/rules/*`)
5. Conformidade com ADRs relevantes (violação clara = critical; desvio sem justificativa = high)
6. Segurança profunda (IDOR, escalação, fluxos de token, exposição estrutural)
7. Qualidade dos testes (determinismo, asserções, antipatrões)
8. Riscos técnicos

NÃO re-execute a suíte de testes salvo se o sumário do QA indicar `tocou_area_critica: true` E `escopo_testes != "SUITE_COMPLETA"`, OU `executou_testes: false`, OU se detectar violação `critical` em `architecture`/`security`.
```

### 4.3 Interpretar o resultado do Tech Review

| Status | Ação |
|---|---|
| `approved` | Avançar para **4.5 (stage)** → marcar `Concluído` na task e no `task_plan.md` |
| `partial` | Há problemas (qualquer severidade). Enviar TODOS ao executor (4.4) |
| `rejected` | Há problemas (qualquer severidade). Enviar TODOS ao executor (4.4) |

> **Zero-débito técnico**: `partial` e `rejected` são tratados igual — ambos exigem correção de TODOS os problemas (`critical`, `high`, `medium`, `low`) antes da task avançar. A task só fica `approved` quando `problems: []`.

### 4.4 Loop de correção Tech Review (memória lazy)

Se Tech Review reprovou:

1. **Atualize a memória lazy** (crie se ainda não existe do 3.5):
   ```markdown
   ## JSON Tech Review
   ```json
   [JSON completo de 4.2]
   ```
   ```

2. **Extraia TODOS os problemas — sem filtro por severidade**:
   - `problems[]`: `id`, `severity`, `category`, `title`, `description`, `expected`, `impact`, `suggested_fix`, `adr_referenciada`
   - Inclua TODAS severidades (`critical`, `high`, `medium`, `low`) e categorias (incluindo `adr_compliance`).

3. **Monte o prompt de correção**:

   ```
   A task [ID] foi REPROVADA pela Revisão Técnica. Leia a memória lazy em [path do arquivo].

   ## Problemas Bloqueantes (DEVEM ser corrigidos — política débito-controlado)
   [Para cada problema com severity == critical OU high:]
   - **[P1] ([severity]) [category]**: [title]
     - Descrição: [description]
     - Esperado: [expected]
     - Impacto: [impact]
     - Correção sugerida: [suggested_fix]
     - ADR referenciada: [adr_referenciada se aplicável]

   ## Observações (médios/baixos — débito anotado, opcional corrigir agora)
   [Para cada problema com severity == medium OU low:]
   - **[P_]** ([severity]) [category]: [title] — [suggested_fix]

   Corrija OBRIGATORIAMENTE os críticos e altos da seção "Bloqueantes". Os médios/baixos da seção "Observações" são débito anotado: corrija se for trivial no mesmo escopo; caso contrário, deixa para cleanup futuro (já registrados em qa-observations.md pelo orquestrador). Mantenha a conformidade com a arquitetura e padrões do projeto. Não expanda escopo.

   Arquivos a corrigir:
   [lista de arquivos dos problemas]
   ```

4. **Classifique `requires_qa_revalidation`** aplicando a regra "Tech Review Correction — Classificação `requires_qa_revalidation`" de `.claude/rules/agent-spec-workflow-rules.md`:
   - Olhe `category` de cada item em `problems[]`.
   - Se TODOS estão em categorias `code_review_only` (ex.: `code_quality`, `naming`, `style`, `documentation`, `dead_code`, `imports`) → `requires_qa_revalidation = false`.
   - Se QUALQUER item está em `revalidation_required` (ex.: `architecture`, `security`, `tests`, `logic`, `data_handling`, `error_handling`, `performance`, `concurrency`, `adr_compliance`) ou categoria desconhecida/ausente → `requires_qa_revalidation = true`.
   - Aplique overrides (`tocou_area_critica`, `qa_security_flags_not_empty`, `task_risk == high`, mudança no `git diff --stat`) — qualquer um força `true`.
   - Persista `requires_qa_revalidation: <bool>` na memória lazy junto com a justificativa (categorias encontradas + overrides ativos).
5. **Dispare o executor** com prompt de correção (escale modelo se aplicável).
6. **Re-valide conforme `requires_qa_revalidation`**:
   - **`true`** → primeiro Gate 1 — QA (3.3) → se QA aprovar, Gate 2 — Tech Review (4.2).
   - **`false`** → **PULE QA**, vá direto a Gate 2 — Tech Review (4.2). Logue em `shared.qa_observations.path`: `T[N] retry — QA pulado (categorias code_review_only: <lista>)`.
7. **Limite máximo: 3 tentativas TOTAIS** por task (compartilhado entre QA e Tech Review).
8. **Ao aprovar final**: delete a memória lazy `T{N}.md`.

### 4.5 Stage da task aprovada (`git add`)

**Apenas quando Tech Review retornou `status: "approved"`**:

1. **Coletar a mesma `task_paths`** usada no diff de 4.1.
2. **Stage real**: `git add -- <task_paths>` (substitui o `git add -N` por adição definitiva).
3. **NÃO commitar** — o usuário decide quando agrupar tasks num commit.
4. **Logar**: `T[N] — staged: [lista de paths]`.

> Por que stage real ao final: a próxima task captura seu próprio `base_sha = git rev-parse HEAD` e gera `git diff <novo_base_sha> -- <novos_paths>`. Filtro por path isola tasks com paths disjuntos. Overlap real é raro (geralmente erro de planejamento) — usuário precisará commitar entre elas para resetar baseline.

**Erro no `git add`** (path inválido, etc.): NÃO falhe a task — registre em `shared.qa_observations.path` como observação não-bloqueante.

### 4.6 Escalar ao usuário (após 3 tentativas)

Se após 3 tentativas totais o QA ou Tech Review ainda reprovar:

1. **NÃO marque a task como concluída.**
2. **Marque como `Bloqueado`** no arquivo da task e no `task_plan.md`.
3. **Informe ao usuário** com o relatório completo:
   - Qual task está bloqueada
   - Quantas tentativas foram feitas
   - Quais problemas persistem (extrair do último JSON do QA e/ou Tech Review)
   - Qual gate está bloqueando (QA, Tech Review ou ambos)
   - Sugestão de ação
4. **Pergunte ao usuário** como proceder antes de continuar.
