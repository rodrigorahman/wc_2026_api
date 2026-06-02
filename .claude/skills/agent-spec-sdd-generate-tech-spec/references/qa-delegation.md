# Delegação QA — Estratégia de Testes (multi-variante)

> Este arquivo é consultado pela skill `agent-spec-sdd-generate-tech-spec` no momento de preencher a **Seção de Estratégia de Testes** do TECH_SPEC. A numeração varia por variante:
>
> - **Web**: seção **17** (sub: 17.1 Unit, 17.2 Integ, 17.3 E2E, 17.4 Erros)
> - **Mobile**: seção **18** (sub: 18.1, 18.2, 18.3, 18.4)
> - **Backend**: seção **19** (sub: 19.1, 19.2, 19.3, 19.4)

A seção de testes do TECH_SPEC **NÃO** deve ser preenchida pelo arquiteto. Você DEVE delegar para o subagente **`agent-spec-qa-test-generator`** que retorna um JSON estruturado. Depois, você converte o JSON em markdown para a seção correta.

---

## Quando executar

Após coletar todas as decisões técnicas e preencher as seções anteriores à de Testes, **ANTES de salvar o arquivo final**.

---

## Passo 1: Preparar a lista de arquivos

Monte a lista de `arquivos` que o subagente deve ler. Inclua TODOS os caminhos relevantes:

- **PRD aprovado**: path resolvido a partir de `sdd.prd.path` (CLAUDE.md)
- **Regras do projeto**: `CLAUDE.md`, `.claude/rules/*.md` (se existirem)
- **Migrações / schema**: arquivos de migração/esquema relacionados à feature, conforme o padrão do codebase (ex.: `*/migrations/*.sql`, `*/migrate/*.go`, `prisma/migrations/*`, `db/migrate/*.rb`, `alembic/versions/*.py`)
- **Camada de dados**: arquivos de queries/repositories relacionados à feature, conforme a stack (ex.: `*.sql` com SQLC, schema Prisma, repositórios/DAOs)
- **Testes existentes**: busque arquivos de teste do projeto na convenção da stack (ex.: `*_test.go`, `*.spec.ts`/`*.test.ts`, `test_*.py`, `*_test.dart`, `*Test.java`) para o subagente entender padrões
- **Código-fonte existente**: arquivos que serão modificados pela feature

---

## Passo 2: Preparar as instruções

Monte o campo `instrucoes` com:

1. **Frente da TECH SPEC** (`frente: web | mobile | backend`) — decidida em FASE 0 da skill. Esse campo orienta o subagente a aplicar a **matriz de stacks de teste** correta (Passo 3).
2. O conteúdo completo do **TECH_SPEC parcial (seções já preenchidas)** que você montou até o momento.
3. Os **critérios de aceitação (CA-XX)** extraídos do PRD.
4. Os **componentes arquiteturais** envolvidos (handlers, services, repositories, BLoCs, hooks, páginas, etc.) — variam por frente.
5. Qualquer contexto adicional relevante para o QA.

---

## Passo 3: Disparar o subagente

Lance o subagente usando a ferramenta `Agent` com:

- **subagent_type**: `agent-spec-qa-test-generator`
- **description**: "QA gerar testes tech_spec"
- **prompt**: Monte o prompt com os 3 parâmetros obrigatórios:

```
Você foi invocado com os seguintes parâmetros:

1. **frente**: <web | mobile | backend>          ← NOVO. Deriva de FASE 0 da skill agent-spec-sdd-generate-tech-spec.
2. **arquivos**: [lista de caminhos dos arquivos preparados no Passo 1]
3. **instrucoes**: [conteúdo preparado no Passo 2]

OBRIGATÓRIO: Antes de gerar casos de teste, invoque a skill `agent-spec-testing-best-practices` e aplique os 7 gates (Invariant First, Owning Layer, Real Execution, Failure→Fix Production, No Snapshot Without Contract, No Self-Set Mock, Negative Companion). Cada caso de teste DEVE conter os campos `invariant`, `owning_layer`, `existing_suite`, `real_execution_boundary`, `negative_companion`. Detalhes em `.claude/skills/agent-spec-testing-best-practices/references/ai-escreve-testes.md`.
```

> **Modelo**: não passe `model` no `Agent({...})` — confie no default configurado para o subagente.

### Matriz de stacks de teste por frente

Inclua no campo `instrucoes` esta matriz para guiar o subagente:

| Frente | Unitários (típicos) | Integração (típicos) | E2E (típicos) | Cenários de Erro / Especiais |
|--------|---------------------|----------------------|---------------|------------------------------|
| **web** | Vitest / Jest + Testing Library, MSW para mocks de fetch | Componente + store + API mockada (MSW) | Playwright \| Cypress \| WebdriverIO | Erros de rede, fallback offline, validação a11y/i18n |
| **mobile** | Jest (RN) \| `flutter_test` \| XCTest \| JUnit/Espresso; bloc_test, mockito, mockk | Repository + DB local in-memory; mocks de hardware | Patrol \| Detox \| Appium \| XCUITest \| Espresso | Permissão negada, sem rede, conflito offline-first, hardware indisponível |
| **backend** | Vitest/Jest \| `go test` \| pytest \| JUnit; mocks de interfaces | Handler + service + DB real (testcontainers / sqlite in-memory) | HTTP black-box (supertest, RestAssured, custom client) | Constraints, rate-limit, timeouts de dependências, idempotência |

> Se o subagente desconhecer um framework do stack do projeto, ele deve **propor** o equivalente idiomático e nomear claramente. Não invente frameworks.

---

## Passo 4: Converter JSON em Markdown (Seção de Testes)

O subagente retorna um JSON com `casos_teste[]`. Você DEVE converter para o formato markdown da seção de testes correspondente à variante:

| Frente | Seção destino do TECH_SPEC |
|--------|----------------------------|
| Web    | **17** (subseções 17.1 / 17.2 / 17.3 / 17.4) |
| Mobile | **18** (subseções 18.1 / 18.2 / 18.3 / 18.4) |
| Backend | **19** (subseções 19.1 / 19.2 / 19.3 / 19.4) |

> Nas instruções abaixo, `X` representa a numeração da seção principal (17, 18 ou 19) — substitua conforme a variante.

### Mapeamento de tipos

| Campo `tipo` no JSON | Subseção destino |
|---------------------|-----------------|
| `UNITARIO` | **X.1 Testes Unitários** |
| `INTEGRACAO` | **X.2 Testes de Integração** |
| `E2E` | **X.3 Testes End-to-End** |
| `SEGURANCA` | **X.4 Cenários de Erro** (subseção segurança) |
| `PERFORMANCE` | **X.4 Cenários de Erro** (subseção performance) |

### Mapeamento de categorias para X.4

Além dos testes tipo `SEGURANCA` e `PERFORMANCE`, inclua em X.4 todos os `casos_teste` com `categoria` igual a:
- `tratamento_erro`
- `caso_extremo`
- `fronteira` (quando o teste foca em comportamento de erro)

### Formato de saída por subseção

**X.1 Testes Unitários** — agrupe os testes pela camada arquitetural correspondente à frente:

- **Web**: `Componente: [NomeComponente]`, `Hook: [useNomeHook]`, `Store/Slice: [NomeSlice]`
- **Mobile**: `BLoC/ViewModel: [Nome]`, `Widget: [Nome]`, `Repository: [Nome]`
- **Backend**: `Service: [NomeService]`, `Apresentação: [NomeHandler]`, `Dados: [NomeRepository]`

```markdown
### [Camada]: [Nome] (`arquivo_de_teste`)
| Cenário | Método/Caso | Input | Resultado Esperado | Mock/Setup |
|---------|-------------|-------|--------------------|------------|
| [título do CT] | [extrair dos passos] | [dados_entrada] | [resultado_esperado] | [pre_condicoes] |
```

**X.2 Testes de Integração**:

```markdown
### Integração: [Camada A + Camada B]
- **Setup**: [extrair de pre_condicoes — DB in-memory, MSW, mocks de hardware, fixtures]
- **Cenários**:
  | Cenário | Fluxo | Resultado Esperado |
  |---------|-------|--------------------|
  | [título] | [passos resumidos] | [resultado_esperado] |
```

**X.3 Testes E2E**:

```markdown
### Fluxo: [título do CT]
- **Framework**: [Playwright/Cypress | Patrol/Detox/Appium | HTTP black-box / RestAssured / supertest] — escolher conforme `frente`
- **Pré-condições**: [pre_condicoes]
- **Passos**:
  1. [passo 1]
  2. [passo 2]
- **Pós-condições**: [resultado_esperado]
- **Validações**: [observacoes]
```

**X.4 Cenários de Erro**:

```markdown
| Cenário | Trigger | Comportamento Esperado | Código/Status (ou UI Esperada) | Log/Observabilidade |
|---------|---------|------------------------|--------------------------------|----------------------|
| [título] | [dados_entrada] | [resultado_esperado] | [extrair do contexto] | [extrair do contexto] |
```

### Tabela de Rastreabilidade

Monte a tabela a partir do campo `criterios_aceitacao_validados` de cada caso de teste:

```markdown
### Rastreabilidade: Critérios de Aceite → Testes

| Critério (CA-XX) | Testes Unitários | Testes Integração | Testes E2E |
|------------------|-----------------|------------------|------------|
| CA-01            | [CTs tipo UNITARIO com CA-01] | [CTs tipo INTEGRACAO com CA-01] | [CTs tipo E2E com CA-01] |
```

### Informações adicionais do JSON

- **`cenarios_nao_cobertos`**: adicione como nota após a tabela de rastreabilidade
- **`recomendacoes`**: adicione como nota após cenários não cobertos
- **`erros_leitura`**: se houver, mencione quais arquivos não puderam ser lidos

---

## Passo 5: Validar como arquiteto

Antes de integrar a seção de testes no TECH_SPEC:

1. Verifique **coerência** com as seções anteriores (componentes, fluxos, APIs/telas/integrações mencionados nos testes existem?).
2. Verifique que **todos os CA-XX** do PRD têm pelo menos um teste na tabela de rastreabilidade.
3. Verifique que os **frameworks de teste** propostos batem com a `frente` informada (Web → Playwright/Cypress; Mobile → Patrol/Detox/Appium; Backend → HTTP black-box adequado à stack).
4. Ajuste nomenclatura de componentes se o subagente usou nomes diferentes dos definidos nas seções anteriores.
5. Complemente se algum cenário crítico ficou de fora.
6. **Conformidade com `agent-spec-testing-best-practices`** (NOVO):
   - `mock_budget_observado` no JSON é `true`?
   - `gates_aplicados` contém os 7 gates?
   - Cada caso de teste tem `invariant`, `owning_layer`, `existing_suite`, `real_execution_boundary`, `negative_companion` preenchidos?
   - Pelo menos um caso por feature tem `real_execution_boundary != "none"`?
   - Cada caso positivo (`categoria: caminho_feliz | interacao_usuario`) tem `negative_companion.presente: true` apontando para um caso negativo?
   - Se algum item falhar, **re-disparar** o subagente com instrução pontual para corrigir, OU rejeitar o JSON e abrir solicitação no chat.
   - **`stack_discovery.discovery_needed`**: se `true`, o subagente não conseguiu resolver um detalhe **não-derivável do código** (ex.: framework E2E não padronizado). Recomende ao usuário rodar **`/agent-spec-testing-stack-bootstrap`** para descobrir a stack (com questionário do não-derivável) e gerar a rule `.claude/rules/agent-spec-testing-stack.md` — depois reexecute a delegação. Não bloqueie a geração por isso; siga best-effort com o proposto.

---

## Passo 6: Integrar e continuar

1. Insira a seção de testes convertida no TECH_SPEC (seção 17 Web | 18 Mobile | 19 Backend).
2. Preencha a seção **Arquivos Envolvidos** (seção 20 Web | 21 Mobile | 22 Backend).
3. Preencha o **Checklist Final** (última seção do template).
4. Salve o arquivo e apresente ao usuário.

**NÃO peça aprovação isolada da seção de testes** — apresente o TECH_SPEC completo para validação final.

---

## Regra de deduplicação de CTs (OBRIGATÓRIA)

Cada **CT** (caso de teste) DEVE aparecer em **NO MÁXIMO 1 task** na tabela de rastreabilidade (§14 mapeamento CA→CT). Isso evita que o mesmo teste seja implementado 2× em tasks diferentes — problema detectado em execução real.

Regras específicas:

1. **CTs compartilhados entre tasks**: se um CT valida integração entre módulos de 2 tasks (ex.: CT-28 de SQL injection entre T6 e T7), atribua-o **apenas à task que IMPLEMENTA o código que é validado**. A outra task apenas "consome" — não precisa listar o CT na sua rastreabilidade.

2. **Testes manuais / smoke / validação humana**: ficam na task que canonicamente os possui (geralmente a última task da fase, ou a task de E2E/packaging). NÃO duplique em tasks anteriores.

3. **Validação cruzada**: se o mesmo comportamento precisa ser validado em camadas diferentes (unit + integração), use CTs distintos (CT-10 unit em T3, CT-11 integração em T5) — NÃO o mesmo CT em ambas.

Durante a geração do tech_spec, verifique explicitamente:
- [ ] Cada CT aparece em exatamente 1 linha da tabela CA→CT→Task.
- [ ] Nenhum CT é referenciado em múltiplas tasks.
- [ ] Smoke/manual tests estão em 1 task canônica.
