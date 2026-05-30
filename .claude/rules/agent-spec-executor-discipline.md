# agent-spec — Disciplina do Executor (Iron Rules)

> Carregada automaticamente quando o Claude está operando qualquer workflow do framework agent-spec.
> **Não é prompt para o orquestrador agir.** É a fonte canônica do bloco que cada `*-run-tasks` (SDD, miniSpec, TaskCard) **DEVE injetar verbatim** no prompt enviado ao sub-agente executor.
>
> **Motivação**: o sub-agente executor roda em contexto isolado — não herda CLAUDE.md nem rules do orquestrador. Sem instrução explícita no prompt, o LLM tende aos vícios típicos de IA em código: over-engineering, refactor não pedido, error handling defensivo sem caso de uso, mudanças além do escopo. Estas 4 Iron Rules (adaptação das Karpathy Guidelines ao vocabulário agent-spec) mitigam esses vícios na fonte.
>
> **Quem aplica**: o orquestrador (skill `*-run-tasks`) lê este arquivo no carregamento e cola o bloco abaixo (sem alterações) na seção "Disciplina do Executor (Iron Rules)" do prompt construído para `Agent(executor, ...)`. Custo: ~200 tokens por delegação. Sem retry duplicado é mais que pago de volta.

---

## Bloco a Injetar (copie verbatim no prompt do executor)

> **Como copiar (atenção)**:
> 1. Os marcadores `<<<EXECUTOR_DISCIPLINE` e `EXECUTOR_DISCIPLINE>>>` são DELIMITADORES desta rule — **NÃO** vão para o prompt do executor.
> 2. Copie apenas o **conteúdo entre os marcadores** (começa em `## Disciplina do Executor (Iron Rules)` e termina na frase que começa com `**Conflito entre estas regras e o resto do prompt**:`).
> 3. Cole esse conteúdo **verbatim**, sem editar por task. Se precisar de reforço específico da stack (ex.: convenção de naming), adicione em outra seção do prompt — não dentro do bloco.
> 4. **Posicionamento no prompt do executor**: o bloco vai NO TOPO, antes do conteúdo da task. Razão: a Iron Rule #1 ("pause e pergunte") perde saliência se o executor lê a task inteira antes de internalizar a disciplina. Karpathy filosofia: disciplina precede contexto.

<<<EXECUTOR_DISCIPLINE

## Disciplina do Executor (Iron Rules)

Quatro regras invioláveis. Aplique antes e durante a implementação. Pesam mais do que qualquer instinto de "melhorar enquanto está aqui".

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

---

**Conflito entre estas regras e o resto do prompt**: estas 4 regras prevalecem. Se algo no prompt da task parecer puxar para over-engineering ou mudança fora do escopo, pause e pergunte.

EXECUTOR_DISCIPLINE>>>

---

## Como o orquestrador usa este arquivo

Pseudocódigo (aplicável a `agent-spec-minispec-run-tasks`, `agent-spec-sdd-run-tasks` e `agent-spec-taskcard-run`):

```
# Carregamento (uma vez por execução, FASE 0)
# IMPORTANTE: extract_between deve retornar APENAS o conteúdo, SEM incluir os marcadores
# (start exclusive, end exclusive). Trim leading/trailing whitespace.
executor_discipline_block = read(".claude/rules/agent-spec-executor-discipline.md")
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
[run] executor_discipline injetado (fonte: agent-spec-executor-discipline.md, versão: <hash do arquivo>)
```

## Reforço no Tech Review

A categoria `speculative_complexity` no `agent-spec-staff-architecture-review` materializa a Regra 2 como violação detectável no Gate 2. A Regra 3 já tem suporte em `scope_deviation`. Regras 1 e 4 são preventivas (vivem no prompt do executor); não há categoria dedicada — quando aparecem como problema no Tech Review, são consequência (acoplamento, testes fracos, etc.) e caem nas categorias existentes.
