# Pré-Refinamento — Definição Inicial da Ideia

> Este documento é um artefato **intermediário**, anterior ao PRD / INTENT / TaskCard.
> Serve para reduzir ambiguidade cedo e preparar terreno para a próxima etapa do workflow.
>
> **Legenda:**
> - Linhas sem marcação = **FATO** (afirmado pelo usuário).
> - `[HIPÓTESE]` = inferência da skill que precisa ser validada.
> - `[DÚVIDA]` = ponto em aberto, detalhado na seção 12.

---

## 1. Metadados

- **Nome da Ideia / Feature**:
- **Autor**:
- **Data**:
- **Versão**: v1
- **Status**: Draft | Refinado | Pronto para próxima etapa
- **Relacionados**: (notas, tickets, PRDs anteriores)

---

## 2. Ideia Resumida (uma frase)

<!-- LLM-ONLY: Reescreva a ideia do usuário em UMA frase clara, sem ambiguidade. Se a frase original já for suficiente, mantenha-a. Remova este comentário no arquivo gerado. -->

---

## 3. Problema

- **Qual é a dor real hoje?**
- **Como o problema aparece no dia a dia?** (exemplos concretos se houver)
- **Quem sente o impacto?** (quem perde tempo, dinheiro, confiança)
- **Por que resolver agora?**

---

## 4. Objetivo Principal

- **Qual é o resultado esperado ao final?**
- **Qual mudança de comportamento/estado deve acontecer?**

<!-- LLM-ONLY: Descreva apenas O QUE deve ser alcançado, NÃO como implementar. -->

---

## 5. Público / Usuário Envolvido

- **Persona primária**: quem usa/recebe o valor?
- **Persona secundária** (se houver):
- **Contexto de uso**: onde, quando, em qual dispositivo/ambiente?

---

## 6. Contexto

- **O que está acontecendo ao redor dessa ideia?** (iniciativas relacionadas, incidentes, feedback)
- **Existe solução parcial hoje?** (sistema legado, planilha, processo manual)
- **Há dependências conhecidas** de outras áreas, times ou sistemas?

---

## 7. Escopo Inicial (rascunho)

Itens que **parecem** fazer parte da primeira versão:

- [ ]
- [ ]
- [ ]

> Esta lista NÃO é definitiva — é um ponto de partida para o PRD/INTENT/TaskCard.

---

## 8. Fora do Escopo (rascunho)

Itens que **explicitamente** NÃO fazem parte dessa primeira versão:

- [ ]
- [ ]

---

## 9. Restrições

- **Prazo / janela de tempo**:
- **Integrações obrigatórias** (se já conhecidas):
- **Regras de negócio / compliance / privacidade**:
- **Decisões já tomadas** (fora de negociação):

---

## 10. Aproveitamento de Capacidades Existentes

> Lista explícita do que o projeto **já tem** que deve ser reutilizado antes de cogitar novidade.

- **Persistência**:
- **Autenticação / autorização**:
- **Mensageria / filas / sincronismo**:
- **Armazenamento de arquivos / objetos**:
- **Observabilidade / logging**:
- **Padrão arquitetural**:
- **Outros módulos internos reutilizáveis**:

<!-- LLM-ONLY: Inspecione CLAUDE.md, go.mod, docker/, pkg/, sql/ e .cursor/rules/ para preencher esta seção. Só cite tecnologia NOVA se houver justificativa concreta; nesse caso, marque [HIPÓTESE] e registre como dúvida aberta. Remova este comentário no arquivo gerado. -->

---

## 11. Premissas

Suposições assumidas (implícita ou explicitamente) para que a ideia faça sentido:

- [HIPÓTESE]
- [HIPÓTESE]

---

## 11.1 Inventário de Impacto (grep de consumidores — OBRIGATÓRIO se a ideia introduz singleton/wire/provider)

> Quando a ideia mencionar centralização de SDK, provider DI, singleton ou refactor de fábrica para injeção, **liste TODOS os consumidores legados do símbolo a ser confinado**. Marque `N/A — sem centralização` se não aplicável.

**Símbolo confinado**: `<ex.: aws.NewAWS(`>
**Comando executado**:
```
grep -rE "<simbolo>" --include="*.<ext>" . | grep -v "<dir do próprio provider>"
```

| Arquivo:Linha | Trecho | Tratamento |
|---|---|---|
| `pkg/x/y.go:42` | `awsClient := aws.NewAWS(region)` | refactor obrigatório |
| `internal/handlers/upload.go:18` | `s3 := aws.NewAWS(...)` | refactor obrigatório |
| `pkg/remote-params/remote-params-prd.go:55` | `aws.NewAWS(cfg.Region)` | **refactor obrigatório (descoberta tardia → cascade)** |

> **Por que**: o post-mortem `cadastro-pratos-franquia` (T1) gastou 1h+ porque scope listou 2 consumidores e omitiu 1. Grep no pre-refinement antecipa o impacto.

---

## 12. Riscos e Pontos de Atenção

- **Risco técnico ou operacional**:
- **Risco de produto / aceitação**:
- **Risco de privacidade / segurança / compliance**:
- **Risco de escopo** (pode explodir facilmente?):

---

## 13. Dúvidas em Aberto

Perguntas objetivas que precisam ser respondidas antes da próxima etapa:

1. [DÚVIDA]
2. [DÚVIDA]
3. [DÚVIDA]

---

## 14. Separação: Fato × Hipótese × Dúvida (visão consolidada)

| Categoria | Item | Observação |
|-----------|------|------------|
| FATO      |      |            |
| HIPÓTESE  |      | Precisa validar com: |
| DÚVIDA    |      | Bloqueia: |

<!-- LLM-ONLY: Esta tabela consolida a separação das 3 categorias para leitura rápida. Preencha com os itens mais relevantes de cada categoria do documento. -->

---

## 14b. Brainstorm de Produto (divergir antes de convergir)

> Seção **opcional** — preenchida quando a clareza inicial foi baixa/média ou quando a ideia tem espaço para compor melhor o produto antes de fechar.
>
> Objetivo: explorar ângulos, variações e hipóteses alternativas **antes** de escolher o framework. Divergir aqui evita que o produto saia sub-especificado ou mirando só 1 dos ângulos possíveis.

### 14b.1 Variações do produto (quem mais poderia usar?)

Liste 2-4 variações que ampliam o alcance ou mudam a persona alvo:

- **Variação A** — _(ex.: "versão self-service vs guided onboarding")_: público, proposta, custo incremental.
- **Variação B** — _(ex.: "B2C vs B2B com billing por seat")_: público, proposta, custo incremental.
- **Variação C** — _(ex.: "versão mobile-first vs desktop-first")_: público, proposta, custo incremental.

### 14b.2 Ângulos alternativos (que problema adjacente resolve?)

Explore ângulos que a ideia original talvez não tenha pensado:

- **Ângulo adjacente 1**: _(ex.: "a mesma base de dados poderia alimentar relatórios para gestão — 1 task extra, 10× valor")_
- **Ângulo adjacente 2**: _(ex.: "se expuser webhook, terceiros integram sem esforço nosso")_
- **Ângulo adjacente 3**: _(ex.: "rastrear eventos já existentes e consolidar — barato e entrega insight imediato")_

### 14b.3 Features adjacentes de baixo custo (potencializadores)

Liste features que custam pouco mas multiplicam o valor da ideia principal:

- [ ] _(ex.: "exportar CSV" — 1 dia de trabalho, desbloqueia auditoria)_
- [ ] _(ex.: "notificação por email em eventos-chave" — reusa stack existente)_
- [ ] _(ex.: "histórico de alterações com diff" — desbloqueia UX de rollback)_

### 14b.4 Riscos de produto não pensados

Cenários onde o produto pode falhar ou decepcionar — **para planejar antes que aconteça**:

- **Risco 1**: _(ex.: "usuário usa 1× e não volta porque não sente valor imediato")_ → mitigação: _(...)_
- **Risco 2**: _(ex.: "concorrente lança feature similar em 2 meses")_ → mitigação: _(...)_
- **Risco 3**: _(ex.: "custo operacional cresce linearmente com usuários, sem plano de redução")_ → mitigação: _(...)_

### 14b.5 Perguntas provocativas (desafiam premissas)

2-3 perguntas que questionam as premissas da ideia original:

1. _(ex.: "Se tivéssemos 1/10 do tempo, qual seria a versão mínima que ainda entrega valor?")_
2. _(ex.: "Existe alternativa comercial pronta que resolve 80%? Vale construir?")_
3. _(ex.: "Qual é a métrica que diz que falhou, e estamos prontos para matar em 90 dias se não atingir?")_

### 14b.6 Síntese do brainstorm

> Após explorar 14b.1-14b.5, registre aqui **o que foi absorvido** na definição final e **o que foi descartado/adiado**.

- **Absorvido no escopo inicial (seção 7)**: _(listar — pode ter feito novas entradas em 7)_
- **Descartado com justificativa**: _(listar — explicar por que não entra nesta versão)_
- **Adiado para v2/v3**: _(listar — criar seção "Futuro" mental)_

---

## 15. Recomendação de Framework (Strategy Selector)

> Esta seção é **preenchida automaticamente pela skill** com base na rubric de sinais definida em `.claude/skills/pre-refinement/SKILL.md` (seção "Strategy Selection").
>
> A recomendação é **sugestão informada, não bloqueante**. O usuário pode seguir outro framework editando esta seção ou ignorando o próximo passo sugerido.

### 15.1 Sinais Observados

| Sinal | Valor detectado | Confirmação |
|---|---|---|
| S1 — # User Stories implícitas | _(0 / 1-3 / 4-8 / 9+)_ | inferido / confirmado pelo usuário |
| S2 — Stakeholders | _(só dev / dev+1 / múltiplas personas)_ | inferido / confirmado |
| S3 — Novidade técnica | _(bugfix-spike / incremento / greenfield)_ | inferido / confirmado |
| S4 — Artefatos tocados (est.) | _(1-3 / 3-10 / 10+)_ | inferido |
| S5 — Tempo estimado | _(<1h / <1 dia / 1-5 dias / 1-3 semanas+)_ | inferido |
| S6 — Decisões arquiteturais novas | _(sim / não)_ | inferido |
| S7 — Onboarding necessário | _(sim / não)_ | inferido |
| S8 — Risco de regressão | _(baixo / médio / alto)_ | inferido |
| S9 — CRUD-pattern-repeat | _(sim / não — passa em todos 6 critérios do gate)_ | inferido |

### 15.2 Framework Recomendado

**Escolhido**: `<SDD | miniSpec | TaskCard | TaskCard CRUD Fast-Path | Conversa direta>`

**Justificativa** (2-3 frases citando os sinais decisivos):

_(LLM preenche — ex.: "S2=múltiplas personas + S6=sim + S3=greenfield → SDD. A formalização PRD+TechSpec é justificada pela presença de personas B2B e pela decisão arquitetural nova que precisará virar ADR.")_

### 15.3 Alternativas Consideradas

**Por que não <alternativa mais próxima>**:

_(LLM preenche — ex.: "miniSpec seria insuficiente: não captura contrato formal para integração mobile, e não tem espaço para ADR. Economizaria ~60% dos tokens mas deixaria dívida documental significativa.")_

**Por que não <alternativa mais distante>**:

_(LLM preenche — ex.: "TaskCard é sub-dimensionado: escopo atravessa múltiplos módulos; perda de rastreabilidade US→task inviabiliza review futuro.")_

### 15.4 Próximo Passo

```bash
<comando-exato-para-rodar>
# ex.: /sdd-generate-prd "sistema de trocas entre colecionadores"
```

### 15.5 Quando Reconsiderar a Recomendação

- **Upgrade para o framework acima** se durante a execução emergirem: _(LLM lista 2-3 gatilhos específicos, ex.: ">4 US novas aparecem", "decisão arquitetural nova não prevista", "persona nova identificada")_
- **Downgrade para o framework abaixo** se: _(LLM lista 1-2 gatilhos, ex.: "escopo se fecha a 1 módulo único", "zero integrações novas")_

---

## 16. Checklist Final

- [ ] Ideia resumida em uma frase clara
- [ ] Problema descrito com dor concreta
- [ ] Público identificado
- [ ] Escopo inicial e fora de escopo delimitados
- [ ] Toda inferência marcada `[HIPÓTESE]`
- [ ] Seção de "Aproveitamento de Capacidades Existentes" preenchida com reuso concreto
- [ ] Nenhuma proposta de tecnologia nova sem justificativa registrada
- [ ] Dúvidas em aberto listadas como perguntas objetivas
- [ ] **Brainstorm (seção 14b) executado** quando a clareza foi baixa/média ou a ideia tinha espaço para compor (ou justificado como "N/A — ideia fechada")
- [ ] **Síntese 14b.6** registra o que foi absorvido e o que foi descartado/adiado
- [ ] **Tabela 15.1 de sinais preenchida** (S1-S8)
- [ ] **Framework recomendado justificado** com pelo menos 2 sinais decisivos (seção 15.2)
- [ ] **Alternativas consideradas** preenchidas — pelo menos a mais próxima (seção 15.3)
- [ ] **Comando exato** escrito na seção 15.4
- [ ] **Gatilhos de reclassificação** listados na seção 15.5
- [ ] Pronto para alimentar PRD / INTENT / TaskCard
