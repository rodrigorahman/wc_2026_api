# TECH ALIGNMENT

> Registro de uma **discussão de arquitetura de alto nível** (Tree of Thought): pontos de decisão, alternativas viáveis avaliadas e decisões tomadas. Ponto de partida **decidido** para o TECH_SPEC (SDD) ou SCOPE (miniSpec) — **não é especificação**. O arquiteto downstream herda as decisões e define o COMO completo.
>
> **Legenda:**
> - `[HIPÓTESE]` = alternativa/decisão a validar (sem âncora forte no projeto).
> - `[candidata a ADR]` = decisão transversal/evergreen — registrar via `/agent-spec-adr-create`.
> - `a critério do arquiteto` = ponto deixado em aberto para o TECH_SPEC/SCOPE.

---

## 1. Metadados

- **Feature**: <kebab-case>
- **Versão**: v1
- **Framework**: <SDD | miniSpec>
- **Variante**: <web | mobile | backend | a confirmar>
- **Documento de definição**: <path do prd.md ou intent.md>
- **Discovery lido**: <pre-refinement.md? pre-alignment.md? handoff.md? — ou "—">
- **ADRs consultadas**: <IDs relevantes de docs/adr/INDEX.md — ou "—">
- **Data**:
- **Status**: Draft | Decidido | Pronto para TECH_SPEC/SCOPE

---

## 2. Contexto Técnico

> Reescrita técnica afiada do problema/feature: o que precisa ser construído, em vocabulário de arquiteto, com invariantes e conexões explícitas. Sem narrativa do refinamento, sem detalhe de implementação.

<!-- LLM-ONLY: 1-3 parágrafos curtos (ou bullets) reenquadrando a feature tecnicamente. Torne explícitas as invariantes e o que está em jogo. Remova este comentário no arquivo gerado. -->

---

## 3. Pontos de Decisão (esqueleto — Fase 1)

> Os pontos de arquitetura que esta feature força a decidir. Cada um é expandido na seção 4.

| # | Ponto de decisão | Status |
|---|------------------|--------|
| D1 | <1 linha> | discutido / decisão direta / em aberto |
| D2 | <1 linha> | discutido / decisão direta / em aberto |
| D3 | <1 linha> | discutido / decisão direta / em aberto |

---

## 4. Registro de Decisões Técnicas (Tree of Thought — Fase 2)

> Uma entrada por ponto de decisão. Mostra o caminho percorrido (alternativas avaliadas) e o destino (decisão + trade-off), tornando a escolha defensável.

### D1 — <ponto de decisão>

**Por que decidir**: <1-2 frases — o que está em jogo>

**Alternativas avaliadas:**

- **D1.A — <abordagem>**: <descrição em 1 frase>.
  - _Exemplo:_ <exemplo concreto>
  - _Prós:_ <...> · _Contras:_ <...>
  - _Viabilidade:_ <reusa X já existente / requer Y novo / conflita com ADR-NNN / `[HIPÓTESE]`>
- **D1.B — <abordagem>**: ...
  - _Exemplo:_ ...
  - _Prós:_ ... · _Contras:_ ...
  - _Viabilidade:_ ...
- **D1.C — <abordagem>**: ...

**Decisão**: `D1.B` — <justificativa, decidida com o usuário>
**Rejeitadas**: D1.A (<motivo>), D1.C (<motivo / adiado>)
**Trade-off aceito**: <o que abrimos mão ao escolher B>
**ADR?**: não | `[candidata a ADR]` → `/agent-spec-adr-create "<titulo>"`

### D2 — <ponto de decisão>

_(mesma estrutura)_

<!-- LLM-ONLY: repita o bloco por ponto. Para decisões cravadas direto pelo usuário (sem TOT), registre como: "Decisão direta: <X> — <motivo>; sem alternativas avaliadas (usuário cravou / escolha única viável)". Remova este comentário no arquivo gerado. -->

---

## 5. Decisões Candidatas a ADR (transversais / evergreen)

> Decisões que viram padrão do projeto (afetam ≥ 2 features ou estendem/contradizem um padrão existente). **NÃO criar a ADR aqui** — orientar o usuário a rodar `/agent-spec-adr-create`, que revalida os critérios.

| Decisão | Por que é transversal | Comando sugerido |
|---|---|---|
| <ref D?> | <motivo> | `/agent-spec-adr-create "<titulo>"` |

<!-- LLM-ONLY: se nenhuma decisão for transversal, escreva "Nenhuma — todas as decisões são feature-scoped." Remova este comentário no arquivo gerado. -->

---

## 6. Restrições e Invariantes Técnicas

> Condições que qualquer implementação desta feature deve respeitar (vindas do PRD/Intent, da stack, de ADRs ou das decisões acima).

- <ex.: "backend é fonte de verdade no merge — sem reconciliação de campos">
- <ex.: "sincronismo idempotente por chave natural">

---

## 7. Pontos em Aberto (a critério do arquiteto do TECH_SPEC/SCOPE)

> Decisões que conscientemente NÃO foram fechadas aqui — sem âncora suficiente ou sem impacto arquitetural que justifique decidir antes do TECH_SPEC.

1. <ponto> — `a critério do arquiteto`
2. <ponto> — `a critério do arquiteto`

---

## 8. Checklist Final

- [ ] Metadados preenchidos (framework, variante, documento de entrada, discovery, ADRs consultadas)
- [ ] Contexto técnico reescrito com vocabulário de arquiteto (sem narrativa, sem implementação)
- [ ] Pontos de decisão (seção 3) listados
- [ ] Cada ponto com 2-3 alternativas avaliadas (exemplo + prós/contras + viabilidade) OU marcado como decisão direta
- [ ] Cada decisão registrada com escolhida + rejeitadas + justificativa + trade-off aceito
- [ ] Conflitos com ADR/stack tratados como ponto de discussão (não descartados)
- [ ] Decisões transversais listadas como candidatas a ADR (seção 5) — não criadas direto
- [ ] Restrições e invariantes (seção 6) registradas
- [ ] Pontos não decididos em "a critério do arquiteto" (seção 7)
- [ ] Sem detalhes de implementação (endpoints, schemas, arquivos, tabelas, campos, middlewares)
- [ ] Arquivo salvo como `tech-alignment.md` no path resolvido via `tech_alignment.path`
