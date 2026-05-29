---
description: Regras comuns do framework agent-spec — Critical Paths (heurística de áreas sensíveis), paths compartilhados entre workflows (qa_observations, temp_memory, pre_refinement, tech_alignment) e convenções de nomenclatura.
paths:
  - "docs/specs/**"
  - "docs/prds/**"
  - "docs/adr/**"
  - ".claude/skills/sdd-*/**"
  - ".claude/skills/minispec-*/**"
  - ".claude/skills/taskcard-*/**"
  - ".claude/skills/adr-*/**"
  - ".claude/skills/agent-spec-pre-refinement/**"
  - ".claude/skills/agent-spec-generate-tech-alignment/**"
  - ".claude/skills/agent-spec-challenge-spec/**"
  - ".claude/skills/agent-spec-backend-contract-handoff/**"
---

# agent-spec — Regras Comuns dos Workflows

> Carregada automaticamente quando o Claude está operando qualquer workflow do framework agent-spec (SDD, miniSpec, TaskCard, ADR) ou skills compartilhadas (agent-spec-pre-refinement, tech-alignment).
> Centraliza Critical Paths, paths compartilhados e convenções. Paths específicos de cada workflow estão em arquivos separados (`agent-spec-sdd-workflow-rules.md`, `agent-spec-minispec-workflow-rules.md`, `agent-spec-taskcard-workflow-rules.md`, `agent-spec-adr-workflow-rules.md`).

---

## Paths Compartilhados

> Variáveis dinâmicas: `{feature}`, `{version}`, `{task_id}` (ex.: `T1`, `T2`, `TC-001`). Substitua antes de ler/salvar.
> **NUNCA** use paths hardcoded — use os templates abaixo.

### Pré-Refinamento (entrada opcional do discovery — compartilhado entre SDD e miniSpec)
- **pre_refinement.path**: `/docs/specs/features/{feature}/{version}/pre-refinement.md`

### Tech Alignment (compartilhado entre SDD e miniSpec)
- **tech_alignment.path**: `/docs/specs/features/{feature}/{version}/tech-alignment.md`

### Domain Glossary — Dois Níveis (Global + Feature)

O glossário de domínio é dividido em **dois níveis** para acomodar termos que atravessam features (entidades de negócio) e termos restritos a uma única feature (regras operacionais específicas):

- **domain_glossary.global.path**: `/docs/specs/domain-glossary.md`
- **domain_glossary.feature.path**: `/docs/specs/features/{feature}/domain-glossary.md`

#### Quando vai pro GLOBAL vs FEATURE

| Vai pro **GLOBAL** se… | Fica no **FEATURE** se… |
|---|---|
| O termo é uma entidade de negócio que aparece (ou vai aparecer) em ≥ 2 features | É um conceito operacional restrito a essa feature |
| Existe relacionamento entre entidades de domínio | É uma regra/política específica da feature |
| Ex.: entidades centrais do produto (substantivos referenciados por múltiplas features) | Ex.: parâmetros/limites operacionais, estados de máquina internos, regras específicas do fluxo desta feature |

**Default em caso de dúvida**: GLOBAL. É mais fácil descer um termo do global pro feature do que descobrir depois que duas features divergiram silenciosamente.

#### Por que dois níveis (e não um só)

- **Por que ter GLOBAL**: entidades centrais do produto (substantivos que representam coisas do mundo real do negócio) tendem a aparecer em múltiplas features ao longo do tempo. Sem glossário cross-feature, cada feature redefine os termos e diverge ao longo do tempo. O global garante uma única fonte canônica para o vocabulário do **projeto/produto**.
- **Por que ter FEATURE também**: nem todo termo é compartilhado. Regras operacionais e conceitos transitórios poluiriam o global se fossem registrados lá. O glossário-feature preserva esse nível de detalhe sem inflar o canônico.
- **Por que SEM `/{version}/` em ambos**: tanto o global quanto o feature são fontes canônicas de **terminologia**, com vida útil maior do que uma versão específica. v1, v2, v3 da mesma feature compartilham o mesmo glossário-feature; todas as features compartilham o mesmo glossário-global.

#### Precedência na leitura

Skills consumidoras leem **os dois**, nesta ordem:

1. `domain_glossary.global.path` — termos canônicos do domínio.
2. `domain_glossary.feature.path` (se existir) — termos específicos da feature.
3. **Conflito** (mesmo termo nos dois): o FEATURE sobrescreve. Raro e intencional — só faz sentido quando a feature redefine deliberadamente um termo do domínio. Quando isso acontecer, a skill consumidora deve sinalizar a sobrescrita ao usuário.

#### Quando criar

Lazy — só quando alguma skill de spec (PRD, Intent, Tech Spec, Scope) ou de challenge (`/agent-spec-challenge-spec`) identificar terminologia que merece registro canônico. Features triviais ou puramente técnicas podem nunca ter glossário-feature, e projetos pequenos podem rodar muito tempo sem glossário-global.

#### Estrutura mínima (idêntica para ambos os níveis)

```md
# Glossário de Domínio — {Escopo}   ← {Escopo} = "Projeto" no global, ou "{Feature}" no feature

## Termos

**{Termo Canônico}**:
Definição em 1 frase do que o termo É (não o que faz).
_Evitar_: {alias1}, {alias2}

## Relacionamentos
- Uma **{TermoA}** produz uma ou mais **{TermoB}**
- Um **{TermoB}** pertence a exatamente um **{TermoC}**

## Ambiguidades resolvidas
- "{termo ambíguo}" era usado tanto para **{TermoX}** quanto **{TermoY}** — resolvido: são conceitos distintos.
```

#### Quem escreve

- Skills de geração (PRD / Intent / Tech Spec / Scope): **leem** ambos os níveis e validam terminologia contra eles. **Não escrevem** — apenas sinalizam termos novos ao final.
- Skill `/agent-spec-challenge-spec`: **dona** da criação/atualização. Durante o stress-test, ao canonizar um termo, decide com o usuário se ele vai pro **global** (cross-feature) ou **feature** (local) seguindo o critério acima.

### Observações de QA / Tech Review
- **shared.qa_observations.path**: `/docs/specs/features/{feature}/{version}/qa-observations.md`

### Candidatos a Regra (rule mining — append-only durante o run)
- **shared.rule_candidates.path**: `/docs/specs/features/{feature}/{version}/rule-candidates.md`

> **Para que serve**: log append-only de **sinais** que podem virar regra de projeto. Cada agente do framework (executores via `*-run-tasks`, `agent-spec-qa-validator`, `agent-spec-staff-architecture-review`) emite linhas conforme detecta sinais canônicos durante o run. **Nenhum agente decide se vira regra** — a skill `agent-spec-mine-rule-candidates` consolida sinais de múltiplos runs e entrega clusters para `agent-spec-curate-project-rules` aplicar teste de fricção e definir colocação.
>
> **Por que separar de `qa_observations`**: `qa_observations.md` é log de **decisão de pipeline** (retry classification, lote paralelo, gates pulados); é consumido pelo eval de pipeline. `rule-candidates.md` é log de **convenção/decisão repetida**; é consumido pela mineração offline. Misturar polui ambos os consumidores.
>
> **Lifecycle**: criado lazy (só na primeira emissão), versionado normalmente (commitado junto com a feature), nunca apagado pelo orquestrador. Mineração lê histórico cross-feature.

#### Vocabulário canônico de sinais

| Sinal | Quem emite | O que captura |
|---|---|---|
| `executor_askquestion` | `*-run-tasks` | Executor disparou `AskUserQuestion` (convenção ausente forçou pergunta). |
| `pre_refinement_decision` | `*-run-tasks` | Decisão registrada na subseção "Decisões já tomadas (fora de negociação)" do agent-spec-pre-refinement (seção 11). |
| `exemplar_file_read` | `*-run-tasks` | Executor leu arquivo "exemplar" para imitar estilo (convenção não escrita). |
| `repeated_fixture` | `agent-spec-qa-validator` | Mesma fixture/mock/setup usado em ≥2 testes do run. |
| `repeated_assertion_shape` | `agent-spec-qa-validator` | Padrão de assert idêntico em ≥3 lugares. |
| `convention_drift` | `agent-spec-staff-architecture-review` | Finding categoria `convention_drift` (já existe no vocabulário do Tech Review). |
| `scope_deviation` | `agent-spec-staff-architecture-review` | Finding categoria `scope_deviation`. |
| `speculative_complexity` | `agent-spec-staff-architecture-review` | Finding categoria `speculative_complexity`. |

> **Vocabulário fechado**: não invente novos sinais. Se um padrão recorrente não cabe em nenhum dos 8, é candidato a expansão do vocabulário — abra discussão antes de emitir.

#### Schema do arquivo

```markdown
# Rule candidates — {feature}/{version}

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
| 2026-05-29T14:30:00Z | agent-spec-sdd-run-tasks | executor_askquestion | "Devo retornar 404 ou 422 em pedido inexistente?" | T03 / handler de pedido |
| 2026-05-29T14:35:12Z | agent-spec-qa-validator | repeated_fixture | `shared/fixtures/order_basic.json` em 4 testes | t03_handler_test.go |
| 2026-05-29T14:40:48Z | staff-review | convention_drift | log com struct vs `zap.Field` inconsistente | `services/payments/processor.go:48` |
```

**Regras de emissão**:
- Uma linha por sinal. Nunca consolidar múltiplos sinais numa linha.
- `evidence`: texto curto + (quando possível) `path:linha` clicável. Sem evidência verificável → não emita.
- `context`: ID da task + descrição curta do escopo (ex.: `T05 / service de pagamento`).
- Append puro. Nunca reescrever linhas anteriores.

#### Persistência pelo orquestrador

Os agentes `agent-spec-qa-validator` e `agent-spec-staff-architecture-review` retornam sinais via campo `rule_candidates_emitidos[]` no JSON (não escrevem em arquivo). O orquestrador (`agent-spec-sdd-run-tasks`, `agent-spec-minispec-run-tasks`, `agent-spec-taskcard-run`) é responsável por **traduzir esses sinais em linhas append-only** no `shared.rule_candidates.path`. Além disso, o próprio orquestrador emite 3 sinais que só ele observa.

**Regra de criação lazy do arquivo**: o `rule-candidates.md` só nasce quando o **primeiro sinal qualificado** é emitido no run. Se nada qualifica, **não crie** o arquivo (evita poluir o histórico da feature com arquivos vazios). Ao criar, escreva o cabeçalho:

```markdown
# Rule candidates — {feature}/{version}

> Append-only. Emitido por agentes do framework durante o run. Consumido por `agent-spec-mine-rule-candidates`.

| timestamp (ISO-8601) | source | signal | evidence | context |
|---|---|---|---|---|
```

**Trigger points por orquestrador**:

| Momento | Ação | Sinais resultantes |
|---|---|---|
| **Após QA aprovar/rejeitar** (`agent-spec-qa-validator`) | Ler `rule_candidates_emitidos[]` do JSON e **anexar uma linha por item**, com `source: "agent-spec-qa-validator"`. | `repeated_fixture`, `repeated_assertion_shape` |
| **Após Tech Review aprovar/parcial/rejeitar** (`agent-spec-staff-architecture-review`) | Mesmo procedimento, com `source: "staff-review"`. | `convention_drift`, `scope_deviation`, `speculative_complexity` |
| **Executor disparou `AskUserQuestion`** durante a execução | Append linha com `source: "{nome-do-orquestrador}"`, `signal: "executor_askquestion"`, `evidence: <pergunta literal>`, `context: <task_id> / <descrição curta>`. | `executor_askquestion` |
| **Fase 0 do orquestrador, ao carregar agent-spec-pre-refinement** | Se a subseção "Decisões já tomadas (fora de negociação)" do agent-spec-pre-refinement (seção 11) tem itens, append **uma linha por decisão** com `signal: "pre_refinement_decision"`, `evidence: <decisão literal>`, `context: agent-spec-pre-refinement / {feature}`. | `pre_refinement_decision` |
| **Executor leu arquivo "exemplar"** (declarado em `arquivos_referencia` da task ou explicitamente citado pelo executor como modelo) | Append linha com `signal: "exemplar_file_read"`, `evidence: <path do arquivo lido>`, `context: <task_id> / <descrição curta>`. | `exemplar_file_read` |

**Tradução JSON → linha de tabela**:

Para cada item de `rule_candidates_emitidos[]`:

```
| {ISO-8601 do momento da emissão} | {source} | {item.signal} | {item.evidence} | {item.context} |
```

O campo `occurrences[]` do JSON **não vai para a tabela** — fica disponível no JSON original do gate (já persistido pelo pipeline) para a skill `agent-spec-mine-rule-candidates` consultar quando precisar das linhas exatas.

**Deduplicação intra-run**: antes de anexar, o orquestrador grepa o `rule-candidates.md` por `{signal} | {evidence}` (case-insensitive). Se já existe linha idêntica no mesmo run, **pule** (evita duplicar quando QA + Tech Review reportam o mesmo padrão por caminhos diferentes). Deduplicação cross-feature é responsabilidade da `agent-spec-mine-rule-candidates`.

**Falhas não-bloqueantes**: se o append falhar (path inválido, permissão, etc.), registre em `shared.qa_observations.path` como observação e siga. **Nunca** rejeite a task por falha de instrumentação de rule mining.

**Log do orquestrador**: emita uma linha em `shared.qa_observations.path` ao final do run com a contagem total de candidatos persistidos:

```
[run] rule_candidates: N sinais persistidos em <shared.rule_candidates.path> (qa=X, staff=Y, orquestrador=Z)
```

Se N == 0, **não** crie o arquivo nem logue (evita ruído).

### Memória Temporária (lazy — só nasce em rejeição de gate)
- **shared.temp_memory.dir**: `/docs/specs/features/{feature}/{version}/tasks/.tmp/`
- **shared.temp_memory.pattern**: `{task_id}.md`

> **Por que dentro da pasta da feature**: o diretório `.claude/.tmp/` exige autorização explícita a cada gravação (Claude Code trata `.claude/` como área protegida). Movendo para dentro de `tasks/.tmp/` (já writable como qualquer arquivo da feature) eliminamos o prompt de permissão e mantemos a memória co-localizada com as tasks que ela descreve.
>
> **Limpeza**: o diretório `tasks/.tmp/` deve estar listado em `.gitignore` para evitar que arquivos efêmeros (memória lazy) sejam versionados. A skill orquestradora deleta cada arquivo após aprovação dos gates.
>
> **Contexto da execução (NÃO mais em arquivo)**: `base_sha` e sumário do executor (4-6 linhas) passam **inline em `instrucoes`** do QA e do Tech Review. A versão anterior gravava um arquivo `{task_id}-execution-summary.md` com `git diff --stat`, hashes SHA-256 pré/pós e paths consolidados — campos que QA/Tech Review na prática não consultavam (Tech Review GERA diff sozinho via `git diff <base_sha> -- <path>`). Cortado em prol de fluxo mais simples e ~300-800 tokens × 2 gates × N tasks economizados por run.

### Specs (varredura cross-feature)
- **shared.specs_root**: `/docs/specs`
- **shared.specs_glob**: `/docs/specs/**/*.md`

---

## Critical Paths — Heurística de Áreas Sensíveis

> Usada por `agent-spec-sdd-run-tasks`, `agent-spec-minispec-run-tasks` e `agent-spec-taskcard-run` para detectar áreas sensíveis e escalar modelo (executor e gates). **Agnóstica de linguagem/stack** — categorização por **semântica do path**, não por layout específico (Go `internal/`, Java `src/main/`, JS `src/`, Python `app/`, Dart `lib/`).

### Categorias Canônicas

| Categoria | Exemplos de match (qualquer linguagem/stack) |
|---|---|
| **auth** | `**/auth/**`, `**/authentication/**`, `**/login/**`, `**/sessions/**`, `**/oauth/**` |
| **security** | `**/security/**`, `**/permissions/**`, `**/authorization/**`, `**/access-control/**`, `**/rbac/**` |
| **crypto** | `**/crypto/**`, `**/encryption/**`, `**/hashing/**`, `**/jwt/**`, `**/tokens/**`, `**/keys/**` |
| **db_migrations** | `**/migrations/**`, `**/migrate/**`, `**/db/migrations/**`, `**/schema/migrations/**`, arquivos `*.sql` em pastas de migração |
| **secrets/config** | `**/secrets/**`, `**/credentials/**`, arquivos `.env*`, `secrets.*` |
| **api_contracts** | `**/openapi*`, `**/swagger*`, `**/proto/**`, `**/graphql/schema*`, `**/contracts/**` |
| **payments** | `**/payment/**`, `**/billing/**`, `**/checkout/**`, `**/transaction/**` |

### Como aplicar (runtime)

1. Para a task em execução, examine os arquivos declarados nas seções de impacto e o `git diff --name-only`.
2. Faça match de cada path contra as categorias acima (case-insensitive, semântico).
3. Se QUALQUER path bater com QUALQUER categoria → `diff_touches_critical_path = true`.
4. Use o resultado para escalar modelo (gates e executor) conforme regras de cada skill.

### Executor model rules (compartilhadas — aplicadas quando frontmatter NÃO declara `model:`)

```
- match: path em categoria "auth"           → opus
- match: path em categoria "security"       → opus
- match: path em categoria "crypto"         → opus
- match: path em categoria "db_migrations"  → opus
- match: path em categoria "secrets/config" → opus
- match: task_risk == "high"                → opus
- match: files_to_create_count >= 10        → opus
- default                                   → sonnet
```

### Gates inference rules (compartilhadas — aplicadas quando frontmatter NÃO declara `gates:`)

> **Motivação**: o post-mortem `cadastro-pratos-franquia` mostrou que rodar Tech Review por default em **todas** as 10 tasks gastou ~30-50min em wiring/config triviais que o TR jamais reprovaria. Inferir `gates` por tipo de task elimina esse overhead.
>
> **Filosofia**: Tech Review é caro (modelo Sonnet/Opus + leitura de ADRs + diff completo). Aplique apenas onde adiciona valor — área crítica, refactor cross-module, padrão novo. Para CRUD/wiring/config seguindo pattern existente, **QA basta**.

Ordem de avaliação (primeira que casar vence; ausência → `[qa, tech_review]`):

```
- match: tipo == "docs" OU "config_isolada" OU "constantes_isoladas"
         (sem código executável de domínio)                            → none

- match: tipo == "wiring/registry" puro
         (apenas Wire/DI providers, rotas em router, barrel exports,
         registro em init)                                              → [qa]

- match: tipo == "crud_handler" SOBRE pattern_existente
         (handler/route/controller seguindo padrão do projeto, repositorio
         sem regra de domínio nova, DTO trivial)                        → [qa]

- match: tipo == "service_simples"
         (service que delega ao repository com 0-1 sentinela; nenhuma
         regra de negócio complexa, nenhum side-effect externo)         → [qa]

- match: path em "auth" | "security" | "crypto" | "secrets/config"      → [qa, tech_review]
- match: path em "db_migrations"                                        → [qa, tech_review]
- match: tipo == "padrao_novo" OU "candidato_adr"                       → [qa, tech_review]
- match: tipo == "refactor_cross_module" (≥ 3 módulos/pacotes)          → [qa, tech_review]
- match: tipo == "service_complexo" (≥ 2 sentinelas, side-effects ext.) → [qa, tech_review]
- match: task_risk == "high"                                            → [qa, tech_review]
- match: files_to_create_count >= 10                                    → [qa, tech_review]
- default                                                                → [qa, tech_review]
```

**Como o gerador de tasks classifica `tipo`** (heurística textual sobre o nome + arquivos da task):

| Sinais textuais | `tipo` inferido |
|---|---|
| `wire`, `provider`, `register`, `routes`, `swag`, `barrel`, `index.ts`, `mod.rs`, `__init__.py` | `wiring/registry` |
| `migration`, `schema`, `.sql` em pasta de migração, `prisma/migrations/` | `db_migrations` |
| `handler`, `controller`, `route`, `endpoint` + sem palavra-chave "novo padrão" + segue exemplo de outro handler do projeto | `crud_handler` |
| `service` + ≤ 1 sentinela declarada + sem integração externa nova | `service_simples` |
| `service` + ≥ 2 sentinelas OU upload/S3/HTTP externo OU race-condition tratada | `service_complexo` |
| `constante`, `const`, `enum` em arquivo isolado | `constantes_isoladas` |
| `config`, `env`, `viper`, `.env` em arquivo isolado | `config_isolada` |
| `docs`, `.md`, `swagger.yaml` puro | `docs` |
| Toca ≥ 3 pacotes / módulos distintos | `refactor_cross_module` |
| Decide algo que vira ADR (sinalizado pelo hook ADR do gerador) | `padrao_novo` |

> **Default conservador**: na dúvida entre `[qa]` e `[qa, tech_review]`, escolha `[qa, tech_review]`. Pular Tech Review indevidamente em área crítica é mais caro do que rodá-lo num CRUD trivial.

**Log obrigatório**: o orquestrador de execução (`*-run-tasks`) DEVE logar a fonte de `gates` antes de invocar:

```
[T5] gates: [qa] (inferido: tipo=crud_handler, sem critical_paths)     model: sonnet
[T6] gates: [qa, tech_review] (declarado no frontmatter)                model: sonnet
[T1] gates: [qa, tech_review] (inferido: tipo=refactor_cross_module)    model: opus
```

---

## Tech Review Correction — Classificação `requires_qa_revalidation`

> Usada por `agent-spec-sdd-run-tasks`, `agent-spec-minispec-run-tasks` e `agent-spec-taskcard-run` no loop de correção do Tech Review (Gate 2). Decide se a re-rodada após correção precisa **passar pelo QA novamente** (re-validar lógica/comportamento) ou pode **pular o QA e ir direto a um novo Tech Review** (apenas conformidade técnica/code-review). Otimiza tokens e tempo evitando re-QA quando nada mudou no comportamento do código.

### Categorias do JSON do Tech Review

O `agent-spec-staff-architecture-review` retorna problemas categorizados (campo `categoria` em cada item de `problemas.criticos[]` / `altos[]` / `medios[]` / `baixos[]`). As categorias relevantes para classificação:

| Categoria | Tipo | Justificativa |
|---|---|---|
| `architecture` | **revalidation_required** | Mudança estrutural altera fluxo, dependências e contratos — refazer testes |
| `security` | **revalidation_required** | Correção de vulnerabilidade afeta lógica de validação/autorização — re-QA mandatório |
| `tests` | **revalidation_required** | Implica mudar/criar testes — QA precisa re-executar a suíte |
| `logic` | **revalidation_required** | Bug de lógica corrigido muda comportamento — re-QA |
| `data_handling` | **revalidation_required** | Mudança em parsing, validação ou serialização afeta entrada/saída |
| `error_handling` | **revalidation_required** | Mudança em tratamento de erro altera fluxo de exceções e respostas |
| `performance` | **revalidation_required** | Otimização que muda algoritmo/estrutura pode quebrar casos limite |
| `concurrency` | **revalidation_required** | Mudança em concorrência altera comportamento sob carga |
| `code_quality` | code_review_only | Refactor sem mudança de comportamento (extrair função, simplificar expressão) |
| `naming` | code_review_only | Renomear variáveis/métodos/classes (sem mudar API pública) |
| `style` | code_review_only | Formatação, espaçamento, ordem de imports |
| `documentation` | code_review_only | Comentários, docstrings, README |
| `dead_code` | code_review_only | Remoção de código nunca executado |
| `imports` | code_review_only | Reorganização/limpeza de imports |
| `adr_compliance` | **revalidation_required** | Conformidade com ADR pode exigir mudança estrutural — conservador |
| `speculative_complexity` | **revalidation_required** | Remoção de abstração/feature especulativa altera surface area e pode quebrar usos inadvertidos — conservador |

> **Default conservador**: categoria desconhecida ou ausente → `revalidation_required = true`. Nunca pule QA por dúvida.

### Algoritmo de Classificação

```
problemas_corrigir = problemas.criticos[] + altos[] + medios[] + baixos[]

para cada p em problemas_corrigir:
    se p.categoria está em revalidation_required → return requires_qa_revalidation = true
    se p.categoria está ausente/desconhecida    → return requires_qa_revalidation = true (default conservador)

# Chegou aqui: TODOS os problemas estão em code_review_only
return requires_qa_revalidation = false
```

### Sinais Adicionais (override)

Independente da categoria, FORÇAR `requires_qa_revalidation = true` se QUALQUER:

- `tocou_area_critica == true` (path em categoria sensível — ver "Critical Paths" acima).
- `qa_security_flags_not_empty` (QA original já reportou flags de segurança).
- `task_risk == "high"` no frontmatter.
- O patch sugerido pelo Tech Review **adiciona/remove arquivos** ou muda a forma do diff (`git diff --stat` muda nº de arquivos vs. iteração anterior).

### Aplicação no Loop de Correção

Após receber o JSON do Tech Review com `status: rejected`/`partial`:

1. Calcule `requires_qa_revalidation` (algoritmo + overrides).
2. Persista o resultado na memória lazy (`shared.temp_memory.dir`/`{task_id}.md`) sob `requires_qa_revalidation:` e log do motivo.
3. Aplique a correção do executor.
4. **Próxima rodada de validação**:
   - Se `requires_qa_revalidation == true` → volte ao Gate 1 (QA) → depois Gate 2 (Tech Review).
   - Se `requires_qa_revalidation == false` → **PULE QA**, vá direto para Gate 2 (Tech Review).
5. Logue em `shared.qa_observations.path` qual caminho foi tomado e a contagem de problemas por categoria.

> **Por que economizar QA aqui**: correções estritamente de code-review (renomear, formatar, extrair função sem mudar comportamento) **não alteram o comportamento testado pelo QA**. Re-rodar QA nesse caso queima tokens e tempo sem ganho. A classificação é determinística e conservadora (default re-QA), garantindo que mudanças de lógica nunca pulem o QA.

### Log Obrigatório da Decisão (auditoria)

Para cada aplicação do algoritmo, o orquestrador DEVE persistir em `shared.qa_observations.path` o bloco abaixo (substitua valores):

```markdown
### {task_id} — retry classification
- attempt: {N}
- problemas_por_categoria: { architecture: 0, code_quality: 2, naming: 1, adr_compliance: 0, ... }
- overrides_ativos: [tocou_area_critica: false, task_risk: low, qa_security_flags: [], diff_stat_changed: false]
- requires_qa_revalidation: false
- decisao: PULE QA (próxima rodada vai direto a Tech Review)
- justificativa: "todos os problemas em code_review_only (code_quality + naming)"
```

> **Por que log obrigatório**: o post-mortem `cadastro-pratos-franquia` levantou suspeita de que T10 (`naming/style`) foi re-QA indevidamente. Sem log auditável, impossível distinguir bug no algoritmo de execução correta. Com o log, o eval pode validar cada decisão.

### Categorias do `agent-spec-qa-validator` (Gate 1)

> **Importante**: a tabela de categorias acima descreve o JSON do **Tech Review** (Gate 2). O `agent-spec-qa-validator` (Gate 1) também emite o campo `categoria` em cada `problemas.*[]` usando o **mesmo vocabulário canônico**. Quando o loop de correção é disparado por rejeição do QA (não do Tech Review), aplique o mesmo algoritmo sobre o JSON do QA — Camada 6 (`adr_compliance`), Camada 5 (`tests`, `code_quality`) e Camadas 1-4 (corretude, robustez, segurança superfície, completude) usam as mesmas categorias canônicas.

---

## Execução Paralela de Tasks (Fase de paralelismo declarado)

> Usada por `agent-spec-sdd-run-tasks` e `agent-spec-minispec-run-tasks` quando o `task_plan.md` marca tasks com `Pode Rodar em Paralelo? = Sim` na mesma fase. **NÃO se aplica a `agent-spec-taskcard-run`** — TaskCard é por definição 1 task por vez.
>
> **Motivação**: o post-mortem `cadastro-pratos-franquia` declarou T1+T2+T3+T4 paralelos no task_plan, mas o orquestrador ignorava a coluna. Rodaram sequenciais (~40min); em paralelo real seriam ~10min. Economia: ~30min por feature com fase paralela.

### Condições para Paralelizar (TODAS obrigatórias)

Um **lote paralelizável** de tasks é um subconjunto de tasks `prontas` (deps satisfeitas) que satisfaz:

1. **Mesma fase** no task_plan.md (coluna `Fase`).
2. **Todas marcadas `Pode Rodar em Paralelo? = Sim`**.
3. **Paths disjuntos**: a união de paths impactados (seções de arquivos a criar/modificar de cada task) **não tem interseção** entre as tasks do lote. Calcule:
   ```
   for ti, tj in pairs(lote):
       if (ti.paths ∩ tj.paths) != ∅:
           remova ti e tj do lote; rode sequencial
   ```
4. **Sem dependência transitiva implícita**: se T2 modifica arquivo que T1 importa (mesmo que `Dependências` não declare), saída de T1 pode afetar build de T2. Heurística: se T1 cria símbolo público (função/tipo/classe) que T2 referencia textualmente nos arquivos da T2, **remova T2 do lote**.
5. **Limite de paralelismo**: máximo `MAX_PARALLEL = 4` tasks por lote. Lotes maiores quebram em ondas de 4. Razão: tool limits do Claude Code + custo de coordenação cresce não-linearmente.

### Mecânica de Execução Paralela

Para cada task `ti` do lote:

1. **`base_sha` comum** capturado UMA VEZ antes do lote: `base_sha = git rev-parse HEAD`. Todas as tasks do lote usam o MESMO `base_sha` para o filtro `git diff <base_sha> -- <paths>`.
2. **Lançamento concorrente**: numa ÚNICA mensagem do orquestrador, despachar todos os `Agent({...})` do executor das tasks do lote em paralelo (multiple tool calls no mesmo turn).
3. **Aguardar TODOS** os executores retornarem antes de prosseguir.
4. **Persistir `executor_summary[ti]` em memória** (output enxuto de cada executor) — sem arquivo intermediário. `base_sha` (comum) + `executor_summary[ti]` (por task) viajam inline no prompt dos gates.
5. **Gates em paralelo POR TASK**: cada `ti` tem seu próprio pipeline `Agent(agent-spec-qa-validator)` → `Agent(agent-spec-staff-architecture-review)` que pode rodar em paralelo com os pipelines de outras tasks do lote.
   - **Dentro de uma task**: QA → Tech Review continua **sequencial** (Tech Review precisa do sumário do QA).
   - **Entre tasks**: pipelines isolados → totalmente paralelizáveis.
6. **Stage real sequencial**: após TODOS os Tech Reviews do lote aprovarem, faça `git add` numa ordem determinística (ID da task ascendente). Razão: garantir que o próximo `base_sha` (capturado para a próxima fase) seja reprodutível.
7. **Falha em um membro do lote**:
   - Se UMA task falhou em QA ou Tech Review → entra em loop de correção isoladamente (não trava as outras).
   - As demais tasks do lote que aprovaram são `staged` e marcadas concluídas normalmente.
   - A task em loop continua até esgotar 3 tentativas; se bloquear, marca `Bloqueado` e segue.

### Pseudo-algoritmo

```
fase_atual = primeira_fase_com_tasks_prontas()
tasks_fase = tasks de fase_atual com Status="A Fazer"

# Detecta lote paralelizável
candidatos = [t for t in tasks_fase if t.paralelo == "Sim"]
lote = aplique_guards(candidatos)   # paths disjuntos + sem dep transitiva textual
lote = lote[:MAX_PARALLEL]

# Tasks fora do lote: sequenciais
sequenciais = tasks_fase - lote

# Execução do lote
base_sha = git rev-parse HEAD
dispatch_parallel([Agent(executor, ti) for ti in lote])
aguarde_todos()
executor_summary = {ti.id: ti.output_enxuto for ti in lote}   # em memória, não em arquivo

# Gates paralelos por task (recebem base_sha + executor_summary[ti] INLINE)
dispatch_parallel([pipeline_gates(ti) for ti in lote])
aguarde_todos()

# Stage determinístico
for ti in sorted(lote_aprovados, key=lambda t: t.id):
    git add -- <ti.paths>

# Em seguida, rode as sequenciais da mesma fase
for ti in sequenciais: ...

# Próxima fase
```

### Log Obrigatório do Lote

```
[Fase 1] lote paralelo: T1, T2, T3, T4 (paths disjuntos confirmados)
[Fase 1] base_sha=abc1234
[Fase 1] dispatch_parallel: 4 executores em paralelo
[Fase 1] aprovados: T1, T2, T4 | em retry: T3 (QA: critical em CT-010)
[Fase 1] staged sequencial: T1 → T2 → T4
```

### Fallback Automático para Sequencial

Se QUALQUER guard falhar (paths sobrepostos, dep transitiva textual, lote > MAX_PARALLEL), o orquestrador faz **fallback determinístico para sequencial** e logra o motivo em `qa-observations.md`:

```
[Fase 1] paralelismo descartado: T2.paths ∩ T3.paths = ["internal/api/handlers/franchise_dish/wire_provider.go"]
[Fase 1] fallback: sequencial T1 → T2 → T3 → T4
```

---

## Convenções

| Elemento | Convenção | Exemplo |
|----------|-----------|---------|
| Nome da feature (`{feature}`) | kebab-case, minúsculas, sem acentos | `autenticacao-oauth2`, `cardapio-digital` |
| Versão (`{version}`) | `v1`, `v2`, ... (incremental) | `v1` |
| Diretório da feature | `/docs/specs/features/{feature}/{version}/` | `/docs/specs/features/cardapio-digital/v1/` |
| Variante (`{variant}`) | `web`, `mobile` ou `backend` — registrada em `sdd_state.yaml`/`minispec_state.yaml` (raiz e em `steps.<step>`) e na seção 1 do `tech_spec.md`/`scope.md`. **NÃO** entra no path (mantém-se `tech_spec.md`/`scope.md` sem sufixo). | `variant: backend` |
