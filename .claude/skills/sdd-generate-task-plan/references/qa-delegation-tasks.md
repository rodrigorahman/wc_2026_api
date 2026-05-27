# Delegação QA — Seção 6 das Tasks (Testes)

> Este arquivo é consultado pela skill `sdd-generate-task-plan` no momento de preencher a **seção 6 (Testes)** de cada task.

A seção 6 de cada task **NÃO** deve ser preenchida pelo engenheiro de tarefas. Você DEVE delegar para o subagente **`qa-test-generator`** que retorna um JSON estruturado. Depois, você converte o JSON em markdown no formato tabular da seção 6.

> Testes são parte da especificação de cada task — esta etapa é obrigatória.

---

## Pré-verificação: §14 do tech_spec já cobre os CTs? (skip QA quando §14 completa)

**Antes** de invocar qualquer `qa-test-generator`, verifique se o `tech_spec.md` já inclui **§14 Estratégia de Testes** com ≥ 10 CTs detalhados (input → expected → mock → rastreabilidade CA).

```
SE tech_spec.md tem §14 com >= 10 CTs detalhados E cada CT tem mapeamento para CA:
  → REDISTRIBUIÇÃO HEURÍSTICA: distribuir CTs existentes por task via match componente→task.
  → NÃO invocar qa-test-generator para essas tasks.
  → Invocar qa-test-generator APENAS para tasks onde a heurística não achou match (fallback).
SENÃO:
  → Invocar qa-test-generator normalmente para todas as tasks (fluxo padrão).
```

**Redistribuição heurística** (passo a passo):
1. Parseie a tabela §14 do tech_spec, extraindo: `CT-XX, componente, tipo, input, expected, mock, CA-XX`.
2. Para cada task em `task_plan`, extraia os componentes/arquivos da seção 5 (A Criar + A Modificar).
3. Faça match componente↔task via grep/regex dos paths:
   - Se CT menciona `internal/pings/handler/*` e a task T5 tem `internal/pings/handler/ping_handler.go` em 5.1 → CT-XX pertence a T5.
4. Para CTs sem match claro, agrupe e apresente ao usuário via `AskUserQuestion`:
   - "Identifiquei 3 CTs não distribuídos automaticamente: CT-15 (integração), CT-18 (E2E), CT-22 (smoke). Deseja: (a) atribuir manualmente, (b) invocar qa-test-generator para esses específicos, (c) criar uma task 'T-extra-tests'?"
5. Mostre a distribuição proposta ao usuário para aprovação:
   ```
   Distribuição proposta (extraída do tech_spec §14):
   T1: CT-01, CT-02, CT-03 (3 CTs)
   T2: CT-04, CT-05 (2 CTs)
   T3: CT-06 (1 CT)
   ...
   Total: 28/30 CTs distribuídos; 2 CTs em fallback (CT-15, CT-18).
   Aprovar? [s/N]
   ```

**Economia estimada**: em features onde `sdd-generate-tech-spec` já produz §14 completa, **elimina 70-90% das invocações QA**.

**Fallback robusto**: se a heurística falhar para qualquer task, o `qa-test-generator` ainda é invocado (não há perda de qualidade).

---

## Quando executar

Para **cada task**, após preencher as seções 1-5 e 7-8, **ANTES de salvar o arquivo da task**. Se várias tasks estão sendo criadas, dispare subagentes QA em **paralelo** para maximizar eficiência.

---

## Consolidação por camada (reduzir N subagentes para 4)

**Problema**: N tasks → N subagentes paga ~3k de system prompt + ~6k de MCP por invocação. Em 8 tasks isso é ~72k de overhead fixo repetido.

**Estratégia**: agrupe as tasks por **camada arquitetural** e dispare **1 subagente por grupo** que retorna CTs para TODAS as tasks do grupo em 1 JSON:

| Camada | Tipos de tasks agrupadas |
|---|---|
| **infra** | setup de projeto, config, docker, migrations schema, logger, envelope de erro |
| **dominio** | domain models, services de negócio, repositórios, validadores |
| **integracao** | handlers REST, gRPC, wiring de DI, middlewares |
| **e2e + packaging** | testes E2E, smoke, CI, README, Dockerfile final |

**Como invocar**:
1. Classifique cada task em uma das 4 camadas (inferir pelo nome + arquivos impactados).
2. Para cada camada com ≥ 1 task, dispare **1 subagente `qa-test-generator`**.
3. No `instrucoes`, inclua: "Você está gerando testes para um GRUPO de tasks relacionadas. Retorne JSON com chave por task ID (`{'T1': {...}, 'T2': {...}}`). Cada task mantém seu próprio array `casos_teste`."
4. No `arquivos`, passe o `qa_context.md` + TODAS as tasks do grupo (concatenadas) + arquivos relevantes compartilhados pelo grupo.

**Economia estimada**: 8 tasks → 4 subagentes = ~36-48k de overhead eliminado, **sem** perda de qualidade.

**Fallback**: se uma camada tem ≥ 4 tasks com escopo muito divergente, divida em 2 subagentes (ex.: dominio-pings + dominio-auth). Se uma camada tem só 1 task, segue o fluxo tradicional (1 subagente = 1 task).

---

## Passo 0: Extrair `qa_context.md` (OBRIGATÓRIO)

> **Motivo**: sem este passo, cada subagente QA lê o `tech_spec.md` inteiro (~10.5k tokens) apenas para localizar seções §3 (Componentes), §6 (Fluxos técnicos), §11 (CAs) e §14 (Testes). Com N subagentes = N × 10.5k de releitura desnecessária. O `qa_context` condensado (~1.5-2k tokens) resolve isso.

**Antes de disparar qualquer subagente QA**, extraia 1× um `qa_context.md` denso:

1. **Resolva o path** via `sdd.qa_context.path` (CLAUDE.md). O prefixo `.` sinaliza artefato intermediário — adicione ao `.gitignore` se ainda não estiver.
2. **Leia o `tech_spec.md`** uma única vez.
3. **Extraia em formato condensado** (idealmente <2k tokens):
   - **Mapa CA→CT**: tabela com `CA-01 → CT-01, CT-02, CT-05 / CA-02 → CT-03, CT-04 / ...` a partir da rastreabilidade do tech_spec.
   - **Componentes** (§3 condensada): nome + camada + responsabilidade principal (1 linha cada).
   - **Fluxos técnicos críticos** (§6 condensada): apenas os fluxos invocados por ≥ 1 task.
   - **Critérios de Aceitação** (§11 condensada): lista de `CA-XX: título + regra em 1 linha`.
   - **Estratégia de Testes** (§14 condensada): lista de `CT-XX: tipo + input → expected` em 1 linha cada.
   - **Paths relevantes**: lista de arquivos que serão tocados (migrações, queries, etc.).
4. **Salve o `qa_context.md`** no path resolvido.
5. **A partir de agora, cada subagente QA recebe o path do `qa_context.md`** na lista `arquivos` do Passo 1 — NÃO passe `tech_spec.md` completo.

**Ganho estimado**: ~8k × N subagentes ≈ 60-80k tokens economizados em features médias (8 tasks).

**Fallback**: se o `tech_spec.md` for pequeno (<4k tokens), pule este passo e use o `tech_spec.md` diretamente — o overhead de extração não compensa.

---

## Passo 1: Preparar a lista de arquivos

Monte a lista de `arquivos` que o subagente deve ler para CADA task. Inclua:

- **`qa_context.md`** (OBRIGATÓRIO): caminho resolvido via `sdd.qa_context.path`. **Este substitui a passagem do `tech_spec.md` completo** na maioria dos casos.
- **PRD aprovado**: caminho resolvido via `sdd.prd.path` — passado como **referência opcional** para o QA consultar sob demanda.
- **TECH_SPEC completo**: NÃO incluir por padrão. Se o `qa_context.md` não tiver sido gerado (ex.: tech_spec pequeno) OU se a task tocar área pouco coberta pelo contexto condensado, incluir aqui via `sdd.tech_spec.path`.
- **Regras do projeto**: `CLAUDE.md`, `.claude/rules/*.md` (já são carregadas automaticamente no contexto do subagente — NÃO re-liste aqui).
- **Migrações**: arquivos de migração relacionados à task (ex: `internal/db/migrations/*.sql`).
- **Queries**: arquivos de query relacionados à task (ex: `internal/db/queries/*.sql`).
- **Testes existentes**: arquivos de teste relacionados aos arquivos impactados pela task (ex: `*_test.go`, `*.spec.ts`).
- **Código-fonte existente**: arquivos listados na seção 5 da task (a criar ou modificar).

---

## Passo 2: Preparar as instruções

Monte o campo `instrucoes` com:

1. O conteúdo completo da **task parcial (seções 1-5)** que você montou até o momento.
2. Os **critérios de aceite técnico** da task (seção 4).
3. Os **arquivos impactados** pela task (seção 5) — para o QA saber quais camadas testar.
4. O **tipo da task** (cria handler, cria service, cria repository, cria migração, etc.).
5. Qualquer contexto adicional relevante para o QA.

---

## Passo 3: Disparar o subagente

Lance o subagente usando a ferramenta `Agent` com:

- **subagent_type**: `qa-test-generator`
- **description**: "QA gerar testes task TN"
- **prompt**: Monte o prompt com os 2 parâmetros obrigatórios:

```
Você foi invocado com os seguintes parâmetros:

1. **arquivos**: [lista de caminhos dos arquivos preparados no Passo 1]
2. **instrucoes**: [conteúdo preparado no Passo 2]

OBRIGATÓRIO: Antes de gerar casos de teste, invoque a skill `testing-best-practices` e aplique os 7 gates (Invariant First, Owning Layer, Real Execution, Failure→Fix Production, No Snapshot Without Contract, No Self-Set Mock, Negative Companion). Cada caso de teste DEVE conter `invariant`, `owning_layer`, `existing_suite`, `real_execution_boundary`, `negative_companion`. Detalhes em `.claude/skills/testing-best-practices/references/ai-escreve-testes.md`.
```

> **Modelo**: não passe `model` no `Agent({...})` — confie no default configurado para o subagente.

---

## Passo 4: Converter JSON em Markdown (seção 6)

O subagente retorna um JSON com `casos_teste[]`. Você DEVE converter para o **formato tabular** da seção 6 usando o mapeamento abaixo.

### Mapeamento de tipos

| Campo `tipo` no JSON | Subseção destino |
|---------------------|-----------------|
| `UNITARIO` | **6.1 Testes Unitários** |
| `INTEGRACAO` | **6.2 Testes de Integração** |
| `E2E` | **6.3 Testes E2E** |
| `SEGURANCA` | **6.4 Cenários de Erro** |
| `PERFORMANCE` | **6.4 Cenários de Erro** |

### Mapeamento de categorias para 6.4

Além dos testes tipo `SEGURANCA` e `PERFORMANCE`, inclua em 6.4 todos os `casos_teste` com `categoria` igual a:
- `tratamento_erro`
- `caso_extremo`
- `fronteira` (quando o teste foca em comportamento de erro)

### Formato de saída por subseção

O formato DEVE seguir o **formato tabular** idêntico ao da seção de **Estratégia de Testes** do TECH_SPEC (seção 17 Web | 18 Mobile | 19 Backend, conforme a `variant` registrada em `sdd_state.yaml`). Isso garante consistência entre os dois documentos e facilita a validação visual.

Infira o nome do arquivo de teste a partir do componente testado:
- Handler → `[nome]_handler_test.go`
- Service → `[nome]_service_test.go`
- Repository → `[nome]_repository_test.go`

Infira o nome da função de teste a partir do título do CT:
- Use formato `TestNomeMetodo_CenarioDescritivo` (Go convention, adaptar para a linguagem do projeto).

**6.1 Testes Unitários** — formato tabular agrupado por componente:

```markdown
#### [Camada]: [NomeComponente] (`arquivo_test.go`)

Mock: [interfaces mockadas]

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-XX | TestMetodo_Cenario | CA-XX | Verificar que [comportamento] quando [condição] | dados entrada | resultado esperado | dependências mockadas |
```

**6.2 Testes de Integração** — formato tabular com Setup acima:

```markdown
#### [CamadaA + CamadaB] (`arquivo_test.go`)

Setup: [banco in-memory, migrações, fixtures]

| CT | Teste | CA | Objetivo | Fluxo | Validação |
|----|-------|----|----------|-------|-----------|
| CT-XX | TestIntegracao_Cenario | CA-XX | Verificar que [comportamento] quando [condição] | Passos do fluxo | Assertions esperadas |
```

**6.3 Testes E2E** — formato descritivo por fluxo:

```markdown
#### Fluxo: [Nome do Fluxo] (CT-XX)
- **CA**: CA-XX, CA-YY
- **Objetivo**: (1 frase descrevendo o que este fluxo E2E valida de ponta a ponta)
- **Pré-condições**: (estado inicial do sistema)
- **Passos**:
  1. Passo 1
  2. Passo 2
- **Validações**: (assertions sobre dados e estado final)
```

**6.4 Cenários de Erro** — formato tabular:

```markdown
| Cenário | CA | Objetivo | Trigger | Código/Status | Log Esperado |
|---------|----|----------|---------|---------------|-------------|
| Descrição do cenário | CA-XX | Verificar que [constraint] impede [operação] | Ação trigger | Código erro | Mensagem log |
```

### Testes Existentes a Modificar

Após as subseções 6.1-6.4, adicione a tabela de testes existentes. Infira a partir de:
- Campo `recomendacoes` do JSON (se mencionar testes existentes).
- Arquivos de teste já existentes para os componentes impactados pela task (seção 5.2 — Arquivos a Modificar).

```markdown
### Testes Existentes a Modificar
| Arquivo | Motivo da Modificação |
|---------|----------------------|
| [arquivo] | [motivo] |
```

Se nenhum teste existente precisa ser modificado: `> Nenhum teste existente impactado.`

### Informações adicionais do JSON

- **`cenarios_nao_cobertos`**: adicione como nota após a seção 6.4.
- **`recomendacoes`**: use para complementar testes ou identificar testes existentes a modificar.
- **`erros_leitura`**: se houver, mencione quais arquivos não puderam ser lidos.

---

## Passo 5: Validar como engenheiro de tarefas

Antes de integrar a seção 6 na task:

1. Verifique **coerência** com as seções 1-5 da task (os componentes testados existem na seção 5?).
2. Verifique que os testes cobrem os **critérios de aceite técnico** da seção 4.
3. Ajuste nomenclatura de arquivos e funções de teste para seguir os padrões do projeto.
4. Para tasks que NÃO envolvem código (ex: documentação, configuração), preencha "N/A — task não envolve código testável".

---

## Passo 6: Integrar e salvar

1. Insira a seção 6 convertida na task.
2. Salve o arquivo `tasks/TN.md`.
3. Avance para a próxima task automaticamente — **NÃO peça aprovação isolada da seção 6**.
