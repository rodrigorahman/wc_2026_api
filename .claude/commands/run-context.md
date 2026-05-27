---
description: Gera log de contexto perdido do run atual (decisões, erros transitórios, surpresas, tempo subjetivo) antes do fim da sessão.
argument-hint: <feature> [version]
---

# /run-context — Log de Contexto Perdido

Você está sendo invocado para **registrar o que se perde quando esta sessão fechar**: decisões interativas, erros transitórios, surpresas no pipeline e percepção subjetiva de tempo. Esse arquivo serve para **debug do framework** — entender por que um run foi pelo caminho que foi, mesmo quando nada falhou.

## Argumentos

- `$1` = `{feature}` (obrigatório) — nome da feature em kebab-case (ex.: `cardapio-digital`).
- `$2` = `{version}` (opcional, default `v1`).

Se `$1` não foi passado, **pergunte qual feature** antes de continuar. Não chute.

## Path de saída

```
/docs/specs/features/{feature}/{version}/run-context.md
```

**Sobrescreva** se já existir — esse arquivo descreve apenas o run atual.

## O que coletar (revise sua própria transcrição da sessão)

### 1. Decisões interativas
Para **cada** `AskUserQuestion` que você emitiu neste run:
- Pergunta exata (1 linha).
- Resposta escolhida pelo usuário (1 linha).
- Impacto observado: o que mudou de fato no caminho/output por causa dessa escolha (1 linha).

Se não houve nenhuma `AskUserQuestion` no run, escreva `_Nenhuma._`

### 2. Erros transitórios
Qualquer erro/retry que **resolveu sozinho** e portanto **não foi parar** em `qa-observations.md`. Exemplos: timeout de tool, output truncado, retry de comando, race condition que sumiu na segunda tentativa, agent que falhou e foi relançado.

- O que falhou (1 linha).
- Como resolveu (1 linha — "retry funcionou", "ignorei", "mudei abordagem").

Se nenhum, `_Nenhum._`

### 3. Surpresas
Qualquer momento em que o pipeline **tomou caminho inesperado** — do seu ponto de vista de executor — e por quê. Exemplos: gate 2 reprovou algo que parecia trivial, QA pediu re-revalidation onde você não esperava, skill leu um path que não estava previsto, decisão de escalar modelo que parecia desnecessária (ou o oposto).

- O que aconteceu (1 linha).
- Por que foi surpresa (1 linha — o que você esperava vs. o que ocorreu).

Se nenhuma, `_Nenhuma._`

### 4. Tempo subjetivo
Sua **estimativa** (não medida exata) de quais tasks/skills foram lentas neste run e seu **palpite** do porquê. Exemplos: "T3 demorou — muitos arquivos no impacto", "Gate 2 lento — opus + diff grande", "alignment puxou specs de 4 features".

- Item lento (1 linha).
- Palpite (1 linha).

Se nada se destacou, `_Nada se destacou._`

## Regras de redação

- **Factual.** Sem auto-justificativa, sem "tentei mas...", sem "infelizmente".
- **Bullets curtos.** Uma linha por item. Sem parágrafos.
- **Sem promessas de melhoria.** Esse log é diagnóstico, não plano de ação.
- **Sem inferir o que não ocorreu.** Se a seção está vazia, escreva o marcador de vazio (`_Nenhuma._`) — não invente conteúdo.
- Não puxe dados de `qa-observations.md` nem de outros logs do run. Esse arquivo é **complementar**, registra exatamente o que **não** está em outro lugar.

## Template de saída

```md
# Run Context — {feature} {version}

> Log de contexto perdido do run encerrado em {YYYY-MM-DD HH:MM}.
> Diagnóstico do framework — não é plano de ação.

## 1. Decisões interativas

- **P:** {pergunta}
  **R:** {resposta}
  **Impacto:** {o que mudou}

## 2. Erros transitórios

- {o que falhou} — {como resolveu}

## 3. Surpresas

- {o que aconteceu} — esperava {X}, ocorreu {Y}

## 4. Tempo subjetivo

- {item lento} — palpite: {motivo}
```

## Fluxo

1. Resolva `{feature}` e `{version}` (defaults: version=`v1`).
2. Verifique que `/docs/specs/features/{feature}/{version}/` existe. Se não existir, **crie o diretório** e siga em frente — o log de contexto pode ser gerado mesmo que o run não tenha produzido outros artefatos.
3. Releia esta transcrição da sessão atual (decisões, erros, desvios, percepção de tempo).
4. Escreva o arquivo no path acima usando o template.
5. Reporte ao usuário **só o path do arquivo gerado** — sem resumo do conteúdo.
