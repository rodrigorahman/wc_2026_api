---
description: Gera post-mortem do run (outcome, friction points, causa-raiz, lacunas do framework, ação sugerida) — diagnóstico estrutural para evoluir skills/agents/rules.
argument-hint: <feature> [version]
---

# /post-mortem — Análise Estrutural do Run

Você está sendo invocado para produzir um **post-mortem do run** focado em **sinal estrutural** — coisas que travaram, causa-raiz, e lacunas do framework (skills, agents, rules) que aumentaram fricção. Esse arquivo é **complementar à `run-context.md`**:

| Arquivo | Pergunta que responde |
|---|---|
| `run-context.md` | "O que se perde quando a sessão fecha?" (decisões, ruído efêmero) |
| `post-mortem.md` | "O que precisa mudar no framework por causa deste run?" (sinal estrutural) |

**Não duplique conteúdo entre os dois.** Se algo já está em `run-context.md`, **referencie** (`ver run-context §X`) em vez de repetir.

## Argumentos

- `$1` = `{feature}` (obrigatório) — kebab-case (ex.: `cardapio-digital`).
- `$2` = `{version}` (opcional, default `v1`).

Se `$1` não veio, pergunte antes de continuar.

## Path de saída

```
/docs/specs/features/{feature}/{version}/post-mortem.md
```

**Sobrescreva** se já existir — esse arquivo descreve apenas o run atual. Histórico cumulativo vive no git.

## O que coletar

### 1. Outcome
Estado final do run em **uma frase**: o que foi entregue, o que ficou pendente, e por quê.

Formato:
```
Run encerrado com {N} task(s) aprovadas, {M} bloqueada(s) em {gate}, {K} não iniciada(s). Motivo do encerramento: {usuario_pediu|bloqueio|sessao_morta|escopo_concluido}.
```

### 2. Friction points (eventos que custaram tempo/tokens)
Diferente de erros transitórios (esses vão pra `run-context`). Aqui registre **eventos com custo perceptível** que pioraram o run mesmo quando resolveram. Cada item:

- **Onde:** skill/agent/gate específico (ex.: `qa-validator`, `staff-architecture-review-agent`, `sdd-run-tasks`).
- **Sintoma:** o que aconteceu (1 linha).
- **Custo:** tempo perdido ou tokens queimados — estimativa (1 linha).

Se nenhum friction relevante, `_Nenhum._`

### 3. Causa-raiz (análise — máximo 3)
Para os **até 3** friction points mais caros, escreva a **causa-raiz real** — não o sintoma. Use a heurística "5 porquês" mental, mas registre só a conclusão.

Formato por item:
```
**{Friction X}** — causa-raiz: {explicação em 1-2 linhas}.
Onde fica documentado/codificado o comportamento atual: {path do arquivo + linha aproximada se aplicável}.
```

Se não houve friction com causa-raiz interessante, `_Nada de novo._`

### 4. Lacunas do framework (skills, agents, rules)
Coisas que **deveriam existir e não existem**, ou existem mas estão incompletas. Categorize:

- **Skill ausente:** "faltou skill que faça X" (1 linha).
- **Agent ambíguo:** agent existe mas o critério de invocação/output está confuso (1 linha + path).
- **Rule conflitante:** duas rules em `.claude/rules/` dão sinais contraditórios (1 linha + paths).
- **Path/convenção implícita:** algo que o framework assume mas não documenta em rules (1 linha).

Se nenhuma, `_Nenhuma._`

### 5. Ação sugerida (até 3 itens, priorizada)
**Não implemente.** Apenas registre o que mudaria no framework se você tivesse uma sessão dedicada a refactor:

- **#1 (alto impacto):** {ação concreta em 1 linha} — afeta {skill/agent/rule}.
- **#2:** ...
- **#3:** ...

Critério de seleção: maior redução de fricção × menor custo de mudança. Ignore micro-otimizações.

Se nada relevante, `_Nada sugerido._`

## Regras de redação

- **Estrutural, não anedótico.** Se um problema só apareceu uma vez e não tem padrão, vai pra `run-context.md`, não aqui.
- **Sem auto-justificativa.** "Era esperado, mas..." → corta.
- **Sem ação a meio-caminho.** Ou identifica uma ação clara, ou registra `_Nada sugerido._`. Não escreva "talvez seja útil considerar...".
- **Path-grounded.** Toda lacuna/causa-raiz deve apontar **onde** no framework está o comportamento (path do arquivo `.claude/...` ou `docs/...`), exceto quando a lacuna é justamente a ausência do path.
- **Bullets curtos.** Uma a duas linhas por item. Sem parágrafos longos.
- **Não invente friction.** Se o run foi limpo, escreva o marcador vazio em cada seção. Um post-mortem honesto de run limpo é mais útil que um post-mortem inflado.

## Template de saída

```md
# Post-mortem — {feature} {version}

> Análise estrutural do run encerrado em {YYYY-MM-DD HH:MM}.
> Foco: o que precisa mudar no framework. Detalhes efêmeros em `run-context.md`.

## 1. Outcome

{frase única conforme formato da seção 1}

## 2. Friction points

- **Onde:** {skill/agent/gate}
  **Sintoma:** {o que aconteceu}
  **Custo:** {tempo/tokens estimados}

## 3. Causa-raiz

**{Friction X}** — causa-raiz: {explicação}.
Onde: `{path}`.

## 4. Lacunas do framework

- **Skill ausente:** {descrição}
- **Agent ambíguo:** {descrição} — `{path}`
- **Rule conflitante:** {descrição} — `{pathA}` vs `{pathB}`
- **Path/convenção implícita:** {descrição}

## 5. Ação sugerida

- **#1 (alto impacto):** {ação} — afeta `{path}`.
- **#2:** {ação} — afeta `{path}`.
- **#3:** {ação} — afeta `{path}`.
```

## Fluxo

1. Resolva `{feature}` e `{version}` (default `v1`).
2. **Leia `run-context.md`** se já existir, para evitar duplicação. Se não existir, sugira ao usuário rodar `/run-context` primeiro (mas não bloqueie — siga em frente se ele recusar).
3. Releia esta transcrição da sessão atual, mas filtre **só eventos com sinal estrutural** (friction, lacunas, padrões repetidos). Descarte o ruído efêmero.
4. Para cada friction caro, faça análise mental de causa-raiz antes de escrever.
5. Escreva o arquivo no path acima usando o template.
6. Reporte ao usuário:
   - O path do arquivo gerado.
   - As **#1 e #2 ações sugeridas** (ou `nenhuma`), para o usuário decidir se quer abrir tasks de refactor agora.
