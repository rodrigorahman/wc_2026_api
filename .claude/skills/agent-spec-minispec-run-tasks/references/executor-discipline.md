# Disciplina do Executor (Iron Rules) — Reference

> **Referência sob demanda**: carregada pelos orquestradores `agent-spec-*-run-tasks` (miniSpec, SDD, TaskCard) na FASE 0 e injetada **verbatim** no prompt de cada sub-agente executor invocado.
>
> **Por que NÃO é mais rule do system-prompt**: o conteúdo só é útil para os 3 orquestradores acima. Carregar no system-prompt de TODA interação do Claude (chat trivial, outras skills, leitura de arquivo) gastava ~320 tokens permanentes sem retorno. Mover para `references/` torna o carregamento lazy — segue a mesma convenção de `config.md`, `guardrails.md`, `qa-validator-prompt.md`, `staff-review-prompt.md`.
>
> **Arquivo canônico**: este (`agent-spec-minispec-run-tasks/references/executor-discipline.md`).
> Symlinks em `agent-spec-sdd-run-tasks/references/executor-discipline.md` e `agent-spec-taskcard-run/references/executor-discipline.md` apontam para cá. Edição em UM lugar propaga para os 3.
>
> **Motivação do conteúdo**: o sub-agente executor roda em contexto isolado — não herda CLAUDE.md nem rules do orquestrador. Sem instrução explícita no prompt, o LLM tende aos vícios típicos de IA em código: over-engineering, refactor não pedido, error handling defensivo sem caso de uso, mudanças além do escopo e — ao escrever testes — asserção fraca, mock-driven confidence e happy-path-only. Estas 5 regras (as 4 Iron Rules adaptadas das Karpathy Guidelines + a disciplina de testes) mitigam esses vícios na fonte.

---

## Bloco a Injetar (copie verbatim no prompt do executor)

> **Como copiar (atenção)**:
> 1. Os marcadores `<<<EXECUTOR_DISCIPLINE` e `EXECUTOR_DISCIPLINE>>>` são DELIMITADORES desta referência — **NÃO** vão para o prompt do executor.
> 2. Copie apenas o **conteúdo entre os marcadores** (começa em `## Disciplina do Executor (Iron Rules)` e termina na frase que começa com `**Conflito entre estas regras e o resto do prompt**:`).
> 3. Cole esse conteúdo **verbatim**, sem editar por task. Se precisar de reforço específico da stack (ex.: convenção de naming), adicione em outra seção do prompt — não dentro do bloco.
> 4. **Posicionamento no prompt do executor**: o bloco vai NO TOPO, antes do conteúdo da task. Razão: a Iron Rule #1 ("pause e pergunte") perde saliência se o executor lê a task inteira antes de internalizar a disciplina. Karpathy filosofia: disciplina precede contexto.

<<<EXECUTOR_DISCIPLINE

## Disciplina do Executor (Iron Rules)

Cinco regras invioláveis. Aplique antes e durante a implementação. Pesam mais do que qualquer instinto de "melhorar enquanto está aqui".

### 1. Pense antes de codar

**Pare e pergunte via `AskUserQuestion`** (não assuma silenciosamente) se QUALQUER:

- A task admite ≥ 2 interpretações plausíveis do critério de aceite ou da descrição.
- Um termo do domínio aparece com sentido ambíguo e não está no glossário (`/docs/specs/domain-glossary.md` ou `/docs/specs/features/{feature}/domain-glossary.md`).
- Implementar como descrito vai exigir mexer em **arquivo fora da lista declarada** da task (§5.1/§5.2 no SDD, §3.1/§3.2 no miniSpec, §8.2/§8.3 no TaskCard).
- O critério de aceite usa palavras vagas ("apropriado", "razoável", "se necessário") sem âncora mensurável.

Ao perguntar: apresente as interpretações concorrentes, recomende a mais simples e justifique. Não escolha pelo usuário "para adiantar".

### 2. Simplicidade primeiro (YAGNI / KISS)

Implemente **apenas** o que a task pede. Antes de cada bloco adicionado, faça a pergunta-âncora:

> "Um engenheiro sênior chamaria isso de over-engineering?"

**NÃO ADICIONE** (sem demanda explícita na task):

- Features ou parâmetros opcionais "que podem ser úteis".
- Camadas de abstração antecipadas (interface com uma única implementação, factory que cria sempre o mesmo tipo, generics para 1 caso).
- Try/catch ou validação defensiva para casos de erro que a task não declarou.
- Cache, retry, fallback, telemetria que a task não pediu.
- Configurabilidade ("e se um dia quisermos trocar X?").

Pequena repetição local é preferível a abstração prematura. Se em dúvida, **escolha o caminho mais simples** e registre em "Pendências" qualquer trade-off observado.

### 3. Mudanças cirúrgicas

Toda linha alterada deve rastrear de volta a um item da task. Se você não consegue justificar uma mudança contra a descrição/critério/arquivos declarados, **reverta a mudança**.

- **Preserve o estilo do arquivo existente** (naming, ordem de imports, padrão de logs, formato de retorno). Match a vizinhança, não suas preferências.
- **Modifique APENAS arquivos listados** nas seções de impacto da task + arquivos de teste declarados.
- **Dead code preexistente NÃO é seu escopo.** Não remova, não renomeie, não "limpe". Se notar algo gritante, registre em "Pendências".
- **Você PODE (e DEVE) remover símbolos que SUAS mudanças tornaram órfãos** — uma função que só era chamada pela versão antiga do código que você reescreveu, por exemplo. Apenas isso.

### 4. Execução orientada a objetivo

Critério vago **não vira** "vou implementar o que parece certo". Vira teste concreto.

Para cada item da seção de Testes:

1. **Primeiro escreva o teste falhando** (red).
2. Implemente o mínimo para fazer passar (green).
3. Refatore só se a refatoração ainda for justificável pela Regra 2 e pela Regra 3.

A seção de Testes da task **NÃO é opcional**. Se o projeto não tiver engine de testes configurada, **PAUSE e pergunte** via `AskUserQuestion`: (a) configurar engine / (b) gerar testes sem execução / (c) ignorar explicitamente. Nunca pule silenciosamente.

Sem teste verde para cada critério, **não reporte concluída**.

### 5. Disciplina de testes (a doutrina pela qual o QA vai te reprovar)

A asserção definida na seção de Testes é **contrato literal** — implemente-a como está, sem enfraquecer. Os vícios abaixo são os que mais reprovam tasks no gate; trate-os como proibições. **Regras agnósticas de linguagem/framework**: use o equivalente idiomático da stack do projeto (a assertion lib, o runner e as convenções já presentes no código); os nomes de API entre parênteses são apenas ilustrativos e plurais.

- **Asserção literal, nunca genérica.** Se a seção de Testes especifica um valor, sentinela ou código, asserte exatamente aquele — não uma forma mais frouxa:
  - erro → asserte o **tipo/sentinela específico**, nunca apenas "ocorreu um erro" (ex.: `errors.Is` em Go, `rejects.toThrow(Err)` em JS-TS, `pytest.raises(Err)` em Python, `assertThrows(Err)` na JVM — em vez de um "is error" genérico).
  - valor → asserte o **valor exato**, nunca existência genérica (ex.: igualdade de valor em vez de `NotEmpty`/`toBeDefined`/`isNotNull`/`assertNotNull`).
  - dublê de teste (mock/spy/stub) → asserte os **argumentos e o número de chamadas**, nunca apenas "foi chamado".
- **Todo positivo tem negativo.** Se a spec do teste marca um `negative_companion`, o teste negativo é obrigatório, com asserção específica — não um caso "não lança erro" vazio.
- **Não asserte o que o mock plantou.** Programar o dublê para retornar X e então asserir `== X` sem o SUT transformar X é teste oco (mock-driven confidence). Se mockou todos os colaboradores, entregue também o teste de integração que a spec pediu.
- **Toda ação tem asserção.** Teste que executa e não verifica resultado observável (retorno, estado ou side-effect) não conta como teste.
- **Falha = corrija o SUT, não o teste.** Se um teste falha, investigue o código de produção primeiro. Só altere o teste com uma linha `SUT_IS_CORRECT_BECAUSE: <motivo>` justificando por que o teste estava errado.

Estas regras espelham a doutrina `agent-spec-testing-best-practices` (fonte única) — são exatamente os gates que o QA aplica para reprovar. Escreva certo na primeira passada, não no retry.

---

**Conflito entre estas regras e o resto do prompt**: estas 5 regras prevalecem. Se algo no prompt da task parecer puxar para over-engineering ou mudança fora do escopo, pause e pergunte.

EXECUTOR_DISCIPLINE>>>

---

## Como o orquestrador usa esta referência

Pseudocódigo (aplicável a `agent-spec-minispec-run-tasks`, `agent-spec-sdd-run-tasks` e `agent-spec-taskcard-run`):

```
# Carregamento (uma vez por execução, FASE 0)
# IMPORTANTE: extract_between deve retornar APENAS o conteúdo, SEM incluir os marcadores
# (start exclusive, end exclusive). Trim leading/trailing whitespace.
executor_discipline_block = read("references/executor-discipline.md")
                              .extract_between("<<<EXECUTOR_DISCIPLINE", "EXECUTOR_DISCIPLINE>>>")
                              .strip()
# Sanity check: o bloco extraído NUNCA deve conter as strings "<<<EXECUTOR_DISCIPLINE"
# ou "EXECUTOR_DISCIPLINE>>>". Se contiver, a extração está errada — aborte.

# Por task — ao montar prompt do executor
# ORDEM PRESCRITA: disciplina ANTES do task content (saliência).
prompt = f"""
{intro_contextual_breve}                  # 1-2 linhas situando o feature/dependências

{executor_discipline_block}               # Iron Rules — TOPO do prompt

=========================== CONTEÚDO DA TASK ({task_id}) ===========================
{task_content}
=========================== FIM TASK CONTENT ===========================

{reforco_sobre_testes}                    # MANDATÓRIO sobre testes
{notas_contextuais}                       # opcional: alertas específicos da task
{checklist_final}                         # seções e itens a marcar
{output_enxuto_exigido}                   # formato de retorno de 4 linhas
"""

Agent(subagent_type=agent_name, model=effective_model, prompt=prompt, ...)
```

Logue no `shared.qa_observations.path` (uma vez por run, não por task) que o bloco foi injetado — basta a linha:

```
[run] executor_discipline injetado (fonte: references/executor-discipline.md)
```

## Reforço no Tech Review

A categoria `speculative_complexity` no `agent-spec-staff-architecture-review` materializa a Regra 2 como violação detectável no Gate 2. A Regra 3 já tem suporte em `scope_deviation`. A Regra 5 (disciplina de testes) é validada no Gate 1 pelo `agent-spec-qa-validator` (Camada 5 — Qualidade dos Testes): asserção fraca, mock-driven, happy-path-only e teste enfraquecido viram `problemas.*` com `categoria: tests` e os antipadrões correspondentes em `testing_smells`. Regras 1 e 4 são preventivas (vivem no prompt do executor); não há categoria dedicada — quando aparecem como problema, caem nas categorias existentes.
