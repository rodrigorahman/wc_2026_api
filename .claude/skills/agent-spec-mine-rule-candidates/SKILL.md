---
name: agent-spec-mine-rule-candidates
description: |
  Consolida sinais coletados em `shared.rule_candidates.path` ao longo dos
  últimos N runs do framework agent-spec e produz uma lista enxuta de
  candidatos a regra (com evidência de repetição em features distintas),
  pronta para entrega ao `agent-spec-curate-project-rules`. Use SEMPRE que o usuário
  pedir para "minerar regras", "ver o que tem virado pergunta repetida",
  "quais convenções precisam virar regra", "rodar a mineração", "olhar os
  rule_candidates", "consolidar candidatos a regra" ou variações — também
  antes de release / sprint review, quando quiser pagar débito de
  convenção implícita antes que apodreça. Acione também quando o usuário
  descrever "toda hora preciso explicar a mesma coisa pro executor" e
  pedir para descobrir o que é.
when_to_use: |
  - Depois de ≥3 features concluídas, para ver o que repetiu.
  - Antes de release, como parte do passe de "saúde do framework".
  - Quando rules existentes parecem incompletas (executor ainda pergunta muito).
  - Quando staff-review repete o mesmo `convention_drift` em features diferentes.
  - Quando o usuário suspeita de convenção implícita que ninguém escreveu.
do_not_invoke_for: |
  - Decidir o destino/forma de UMA regra específica → use `agent-spec-curate-project-rules`.
  - Coletar sinais em tempo real → quem coleta são os agentes, não esta skill.
  - Auditar rules existentes em busca de bloat/duplicação → use `agent-spec-curate-project-rules` (passe de auditoria).
  - Identificar bugs ou regressões → use Tech Review / agent-spec-qa-validator.
disable-model-invocation: true
---

# /agent-spec-mine-rule-candidates — mineração de candidatos a regra

> Consolida o que **já foi capturado** durante os runs. Esta skill **não decide** se algo vira regra; só agrupa, filtra por repetição e entrega para `agent-spec-curate-project-rules` (que aplica teste de fricção, escolhe escopo e forma).

A regra do framework é simples: **um sinal isolado não vira regra**. Se um padrão aparece numa única feature, pode ser idiossincrasia. Se aparece em **≥2 features distintas** com o mesmo sinal, é convenção implícita que vale a pena formalizar — ou, no mínimo, debater conscientemente.

---

## Fase 0 — Sanity check

Antes de minerar, garanta que o cenário faz sentido:

1. **Confirme o caminho canônico**: `shared.rule_candidates.path` em [`agent-spec-workflow-rules.md`](../../rules/agent-spec-workflow-rules.md). Não invente diretório.
2. **Descubra quais features têm o arquivo**:
   ```
   docs/specs/features/*/v*/rule-candidates.md
   ```
   (use glob compatível com a stack — a estrutura é a do framework agent-spec)
3. **Se não houver nenhum arquivo**: pare e explique ao usuário que nenhum run ainda emitiu sinais. Possíveis causas: (a) instrumentação dos agentes ainda não rodou; (b) features recentes não acionaram `*-run-tasks`. **Não invente candidatos lendo PRDs/specs** — tech-spec é fonte fraca (ver doutrina em `agent-spec-curate-project-rules`).
4. **Pergunte o escopo da varredura via `AskUserQuestion`**:
   - Quantos runs recentes considerar (default = 5).
   - Se restringir por path da feature (ex.: só backend, só web).

---

## Fase 1 — Coleta

Lê todos os `rule-candidates.md` no escopo definido e produz uma lista normalizada:

```
[
  {
    feature: "cardapio-digital",
    version: "v1",
    timestamp: "2026-05-29T14:30:00Z",
    source: "agent-spec-sdd-run-tasks",
    signal: "executor_askquestion",
    evidence: "Devo retornar 404 ou 422 em pedido inexistente?",
    context: "T03 / handler de pedido"
  },
  ...
]
```

**Validações ao parsear**:
- Linhas com `signal` fora do vocabulário canônico (ver `agent-spec-workflow-rules.md`) → loga como warning e ignora. Não tenta "inferir" o sinal.
- Linhas sem `evidence` → ignora silenciosamente (a doutrina exige evidência verificável; emissor descumpriu).
- Timestamps malformados → assume `null` (não bloqueia mineração).

---

## Fase 2 — Agrupamento por similaridade

Agrupa registros em **clusters de candidato**. Critério de pertencer ao mesmo cluster:

| Sinal | Critério de similaridade |
|---|---|
| `executor_askquestion` | Mesma pergunta normalizada (lowercase, sem pontuação, similaridade lexical ≥0.7) **OU** mesmo tema declarado pelo modelo (ex.: "status HTTP para recurso ausente"). |
| `pre_refinement_decision` | Mesma decisão normalizada **OU** mesmo tema declarado. |
| `exemplar_file_read` | Mesmo arquivo exemplar **OU** mesmo padrão estrutural (ex.: "exemplar de handler", "exemplar de service"). |
| `repeated_fixture` | Mesmo path de fixture **OU** mesma forma estrutural da fixture (entidade base). |
| `repeated_assertion_shape` | Mesmo formato de assert normalizado (placeholder dos dados, mesma ordem de campos). |
| `convention_drift` | Mesma convenção divergente (categoria + path-padrão). |
| `scope_deviation` | Mesmo tipo de desvio (ex.: "tocou config global", "alterou shared lib"). |
| `speculative_complexity` | Mesma forma de over-engineering (ex.: "interface com 1 implementação", "config-flag preventivo"). |

**Não misture sinais diferentes no mesmo cluster** — `executor_askquestion` sobre status HTTP é candidato diferente de `convention_drift` sobre status HTTP, mesmo que tematicamente relacionados (a forma da regra é diferente).

---

## Fase 3 — Filtro de repetição (gatekeeper)

Um cluster só vira candidato se:

1. **Aparece em ≥2 features DISTINTAS**. Repetição no mesmo run **não conta** (pode ser sintoma de uma única task mal-estruturada, não de convenção ausente).
2. **Tem evidência citável**: pelo menos 1 linha do cluster com `path:linha` ou ID de task referenciável.
3. **Não está coberto por rule existente**: faça um grep rápido (`.claude/rules/`, `CLAUDE.md`, e — se o host é diferente — equivalentes descobertos pelo `agent-spec-curate-project-rules`) procurando termo-chave do cluster. Se já há rule, marque o cluster como **`coverage_check_failed`** e descarte (com nota).

> **Por que descartar quando já há rule**: significa que a rule existe mas o agente que emitiu o sinal **não a aplicou**. Isso é problema de matcher (rule não está carregando no escopo certo) ou de fraseado (rule não está convincente). Esse é caso para `agent-spec-curate-project-rules` no modo auditoria, não para criar regra nova.

---

## Fase 4 — Candidate cards

Para cada cluster aprovado, monte um cartão:

```markdown
### Candidate: <enunciado curto>

- **Sinal**: <signal>
- **Ocorrências**: N (em M features distintas)
- **Evidências**:
  - {feature-A}/v1 — T03 — "evidence literal"
  - {feature-B}/v2 — T07 — "evidence literal"
  - ...
- **Tema sugerido**: <1 frase>
- **Escopo sugerido (palpite)**: global / por path (`globs sugeridos`) / inline em CLAUDE.md
- **Próximo passo**: passar para `agent-spec-curate-project-rules` aplicar teste de fricção.
```

**Regras de redação do cartão**:
- Enunciado em 1 linha (substantivo + decisão, não verbo no imperativo — quem decide imperativo é `agent-spec-curate-project-rules`).
- Evidências literais, sem parafrasear (o que o agente emitiu).
- Escopo é **palpite** baseado nos paths dos contextos — `agent-spec-curate-project-rules` pode mover.

---

## Fase 5 — Handoff para `agent-spec-curate-project-rules`

Apresente os cartões ao usuário em ordem decrescente de ocorrências:

```
Encontrei {K} candidatos. Quer que eu encaminhe para `agent-spec-curate-project-rules`?
- [ ] Todos
- [ ] Selecionar individualmente
- [ ] Só os top-N
- [ ] Salvar a lista e decidir depois
```

Para cada candidato selecionado, invoque a `agent-spec-curate-project-rules` passando:
- O cartão como contexto.
- Pergunta-âncora: "Este candidato passa no teste de fricção? Se sim, qual escopo e forma?"

**Esta skill não escreve em rules.** Toda gravação é do `agent-spec-curate-project-rules`.

---

## Saída sempre persistível

Salve o relatório de mineração em:

```
/docs/specs/.rule-mining/{YYYY-MM-DD}-mining-report.md
```

(`.rule-mining/` listado em `.gitignore` é OK — relatórios são efêmeros; o que importa é o que vira regra, registrado em `.claude/rules/` ou `CLAUDE.md`.)

Conteúdo do relatório:
1. Escopo da varredura (features cobertas, janela temporal).
2. Estatísticas (total de sinais, por categoria, descartados por motivo).
3. Cartões finais.
4. Decisão do usuário por cartão (aceito / recusado / parking).

---

## Limites desta skill

- **Não substitui post-mortem**: mineração olha sinais agregados; post-mortem olha um run específico em profundidade. São complementares.
- **Não infere regras de código**: leitor preguiçoso que tenta extrair regra direto do diff erra (não há prova de repetição). Use a fila de sinais — quem instrumenta sabe o que é convenção implícita.
- **Não decide colocação**: escopo (global, matcher, inline) é responsabilidade do `agent-spec-curate-project-rules`. A skill só sugere palpite.
- **Não dispara automaticamente**: `disable-model-invocation: true`. Roda só quando o usuário pede ou quando um workflow superior (ex.: release-check) invoca explicitamente.

---

## Sinais de uso saudável

- A mineração **produz poucos cartões** (≤5 numa janela de 5 runs) → instrumentação está calibrada; convenções estão escritas.
- A mineração **produz muitos cartões repetidos do mesmo cluster** ao longo do tempo → o cluster não está virando regra por algum motivo (escopo errado? fraseado fraco?). Disparar `agent-spec-curate-project-rules` em modo auditoria sobre as rules adjacentes.
- A mineração **não acha arquivos** apesar de runs recentes → instrumentação dos agentes não está rodando. Auditar `*-run-tasks`, `agent-spec-qa-validator`, `agent-spec-staff-architecture-review` para conferir emissão.

---

## Resumo executivo

> Coleta passiva (durante o run) + mineração offline (esta skill) + decisão de regra (`agent-spec-curate-project-rules`). Aqui só agrupamos sinais que **se repetiram em features distintas** e empacotamos para a skill que decide. Nunca escrevemos rule diretamente. Nunca inferimos regra que não tem sinal repetido.
