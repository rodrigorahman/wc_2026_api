# Pré-Refinamento — Brainstorm de Produto

> Artefato **intermediário** (anterior ao PRD / INTENT / TaskCard), produto de um brainstorm em **Tree of Thought**: divergir os rumos possíveis, podar com o usuário e convergir.
>
> **Legenda:**
> - Linhas sem marcação = **FATO** (afirmado pelo usuário).
> - `[HIPÓTESE]` = inferência da skill que precisa ser validada.
> - `[DÚVIDA]` = ponto em aberto, detalhado na seção 13.
> - `[fora do escopo do projeto]` = rumo que extrapola o que este projeto se propõe a ser.

---

## 1. Metadados

- **Nome da Ideia / Feature**: Próximos jogos das seleções favoritas
- **Fonte da ideia**: texto livre (+ imagem do card da home)
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Versão**: v1
- **Status**: Draft
- **Relacionados**: `usuario-multiplas-selecoes/v1` (relação N:N usuário↔seleções, consumida por esta feature), `national-team-flag-url/v1` (bandeira da seleção, usada no card)

---

## 2. Ideia Resumida (uma frase)

Cadastrar o calendário de partidas da Copa 2026 (data/hora, duas seleções, estádio, cidade, fase) e expor um serviço autenticado que devolve os **próximos jogos** das seleções favoritas do usuário, para popular um card na home do app.

---

## 3. Esqueleto do Tema (Fase 1 — ramos da árvore)

> O usuário convergiu já na Fase 1 para o recorte mais simples: "uma tabela de jogos com os dados do card, **sem placar nem status** — apenas os dados cadastrados. Os jogos de grupos, oitavas, quartas, semi e final serão cadastrados conforme acontecem."

| # | Ramo | Status (Fase 1) |
|---|------|-----------------|
| A | Modelo da partida (`Match`) com os dados do card | explorar (simplificado: sem placar/status) |
| B | Origem/ingestão dos dados dos jogos | explorar |
| C | Serviço "próximos jogos" das favoritas | explorar |
| D | Dados de apoio do card (estádio, cidade, fase, abreviação) | explorar |
| E | Contrato de consumo (como o cliente chama) | explorar |

---

## 4. Árvore de Rumos (Fase 2 — Tree of Thought)

### Ramo A — Modelo da partida (`Match`)

**Direções candidatas:**

- **A1 — Partida mínima (só os dados do card)**: data/hora, duas seleções (FK), estádio, cidade, fase. Sem placar, sem status.
  - _Exemplo:_ `{ data_hora: 2026-06-12T16:00, home: Brasil, away: México, estádio: Azteca, cidade: Cidade do México, fase: "Grupo A" }`
  - _Viabilidade:_ requer tabela nova `matches`; reusa `national_teams` por FK (home/away).
- **A2 — Partida com placar/status (ao vivo, encerrado)**: acrescenta `status` e gols.
  - _Exemplo:_ card mostraria "2 x 1 · ENCERRADO".
  - _Viabilidade:_ requer atualização contínua dos jogos — fora do recorte.

**Direção escolhida**: **A1** — o usuário foi explícito: "não quero placar nem nada de status, apenas os dados cadastrados".
**Podadas / adiadas**: A2 (adiado — pode virar v2 se a home passar a mostrar resultados ao vivo).

### Ramo B — Origem / ingestão dos dados dos jogos

**Direções candidatas:**

- **B1 — Inserção direta na base de dados**: os registros de partida são inseridos manualmente no SQLite (`INSERT`), fase a fase, conforme a Copa avança.
  - _Exemplo:_ ao terminar a fase de grupos, o operador insere os 8 jogos das oitavas direto na tabela.
  - _Viabilidade:_ requer apenas a tabela + migration que a cria; **nenhum RPC de escrita, nenhum papel de admin**.
- **B2 — RPC de admin (`CreateMatch`)**: endpoint protegido por papel de administrador.
  - _Exemplo:_ um painel admin chamaria `CreateMatch`.
  - _Viabilidade:_ exigiria introduzir autorização por papel (admin) — decisão arquitetural nova que o projeto não tem hoje. `[fora do escopo do projeto]` neste momento.
- **B3 — Seed único com todos os 104 jogos**: uma migration de seed completa.
  - _Exemplo:_ seed igual ao das seleções.
  - _Viabilidade:_ não casa com o mata-mata (times definidos só após a fase anterior).

**Direção escolhida**: **B1** — "direto na base de dados". A feature entrega o **schema** (a tabela) e o **serviço de leitura**; o cadastro dos dados é manual no banco, fase a fase.
**Podadas / adiadas**: B2 (`[fora do escopo do projeto]` — introduziria papel de admin); B3 (incompatível com cadastro incremental por fase).

### Ramo C — Serviço "próximos jogos" das favoritas

**Direções candidatas:**

- **C1 — Infere as favoritas do token, retorna todos os jogos futuros**: RPC autenticado descobre as seleções favoritas pelo `user_id` do token (via `user_national_teams`), retorna partidas com data/hora `>= agora`, ordenadas do mais próximo.
  - _Exemplo:_ usuário com Brasil e Argentina favoritados recebe a lista cronológica de todos os jogos futuros dessas duas seleções.
  - _Viabilidade:_ reusa o interceptor de auth, `user_national_teams` e o pattern de query sqlc; requer query com join + filtro temporal.
- **C2 — Igual a C1, mas com limite N + paginação**: retorna só os próximos N jogos.
  - _Exemplo:_ card mostra os próximos 5.
  - _Viabilidade:_ mesma base, acrescenta limit/cursor. `[DÚVIDA]` se o card precisa de paginação.

**Direção escolhida**: **C1** — infere do token e retorna os jogos futuros ordenados.
**Podadas / adiadas**: C2 (adiado — paginação/limite pode entrar se o volume justificar; ver dúvida 3).

### Ramo D — Dados de apoio do card

**Direções candidatas (decididas item a item):**

- **D1 — Fase como campo único de texto (`stage`)**: um campo livre que hoje recebe "Grupo A" e amanhã "Oitavas de final", "Final".
  - _Exemplo:_ `stage = "Grupo A"`; depois `stage = "Oitavas de final"`.
  - _Viabilidade:_ mínimo; comporta o mata-mata futuro sem nova migration. **Escolhido.**
- **D2 — Estádio + cidade (dois campos de texto na partida)**:
  - _Exemplo:_ `estádio = "Azteca"`, `cidade = "Cidade do México"`.
  - _Viabilidade:_ campos simples na tabela `matches`. **Escolhido.**
- **D3 — Abreviação da seleção (BRA/MEX)**: o card usa sigla de 3 letras que `national_teams` não tem hoje.
  - _Exemplo:_ `Brasil → BRA`, `México → MEX`.
  - _Viabilidade:_ `[HIPÓTESE]` adicionar coluna `abbreviation` em `national_teams` (16 linhas) — a partida só referencia FK; nome, bandeira e sigla vêm da seleção. Alternativa: derivar sigla no front (descartada — pertence ao dado de domínio).

**Direção escolhida**: campo único `stage` (D1) + estádio e cidade separados (D2) + abreviação na seleção (D3, como hipótese a confirmar).
**Podadas / adiadas**: campo `group` separado de `phase` (descartado — `stage` único basta).

### Ramo E — Contrato de consumo

**Direção escolhida**: RPC **autenticado** que infere as seleções do `user_id` no token (alinhado a "os times que o usuário selecionou no cadastro"). A alternativa de receber a lista de times por parâmetro foi descartada por ignorar as favoritas já persistidas.

---

## 5. Problema

- **Qual é a dor real hoje?** A home do app não tem como mostrar ao usuário os próximos jogos das seleções que ele escolheu no cadastro — não existe nenhum dado de partida no sistema.
- **Como o problema aparece no dia a dia?** O usuário favoritou Brasil e Argentina no cadastro, mas o app não consegue dizer "o próximo jogo do Brasil é dia 12/06 contra o México, no Azteca, Grupo A".
- **Quem sente o impacto?** O usuário final (torcedor) que quer acompanhar suas seleções; e o produto, que perde uma seção de engajamento natural na home.
- **Por que resolver agora?** É insumo direto da home do app durante a Copa — tem janela de valor definida pelo calendário do torneio.

---

## 6. Objetivo Principal

- **Resultado esperado:** o app consegue popular o card da home com os próximos jogos das seleções favoritas de cada usuário, a partir de um calendário de partidas mantido no banco.
- **Mudança de estado:** passa a existir a entidade *partida* no sistema e um serviço que cruza partidas futuras com as seleções favoritas do usuário autenticado.

---

## 7. Público / Usuário Envolvido

- **Persona primária**: usuário final autenticado (torcedor) que consome os próximos jogos das suas seleções na home.
- **Persona secundária**: operador/dev que cadastra as partidas direto no banco, fase a fase (não há papel de admin no sistema).
- **Contexto de uso**: app mobile, tela inicial (home), durante o período da Copa 2026.

---

## 8. Escopo Inicial (resultado da convergência)

- [ ] Tabela de partidas (`matches`) com: data/hora, seleção mandante (FK), seleção visitante (FK), estádio, cidade, fase (`stage` texto). Sem placar, sem status. _(Ramo A1, D1, D2)_
- [ ] Migration que cria a tabela `matches`. Cadastro dos dados é manual no banco, fase a fase. _(Ramo B1)_
- [ ] `[HIPÓTESE]` Coluna `abbreviation` em `national_teams` (sigla de 3 letras, ex.: BRA, MEX), para o card. _(Ramo D3)_
- [ ] Serviço/RPC **autenticado** que infere as seleções favoritas do token (`user_national_teams`) e retorna os jogos futuros (data/hora `>= agora`) ordenados do mais próximo. _(Ramo C1, E)_

> Ponto de partida para o INTENT/TaskCard — não é definitivo.

---

## 9. Fora do Escopo (podado / adiado)

- Placar e status da partida (ao vivo / encerrado) — _podado; pode virar v2 se a home passar a mostrar resultados_ (A2).
- RPC de criação/edição de partida e papel de administrador — _`[fora do escopo do projeto]`: introduziria autorização por papel, que o projeto não tem_ (B2).
- Seed único com todos os jogos — _incompatível com cadastro incremental por fase_ (B3).
- Paginação / limite N no serviço de busca — _adiado; entra se o volume justificar_ (C2).
- Campo de grupo separado da fase — _descartado; `stage` único basta_ (D).

---

## 10. Ancoramento no Projeto (guarda de escopo)

- **O que o projeto É** (CLAUDE.md / README): API gRPC em Go (uber-fx + SQLite pure-Go + sqlc) para o Álbum da Copa do Mundo 2026 — auth JWT e catálogo de seleções nacionais.
- **PRDs / specs existentes consultados** (`/docs/specs/**/*.md`):
  - `usuario-multiplas-selecoes/v1` — **base direta**: migrou a relação usuário↔seleção para N:N via `user_national_teams`. É exatamente a fonte das "seleções favoritas" que o serviço de busca consome.
  - `national-team-flag-url/v1` — adjacente: adicionou `flag_url` à seleção, usada no card (bandeira). A abreviação (D3) seguiria o mesmo padrão de coluna em `national_teams`.
  - `arquitetura-base/v1` — pattern de domínio (handler/service/repository + fx module + migration + sqlc + proto) a ser replicado pelo novo domínio `match`.
  - `domain-glossary.md` — vocabulário canônico: `NationalTeam` (não "Selection"/"Team"), `User`. Convenção de idioma: dado/DB em pt-BR, código Go/proto em inglês.
- **Capacidades reutilizáveis** (apenas para viabilidade):
  - **Persistência**: SQLite + migrations embarcadas (`embed.FS`) + sqlc; tabela `national_teams` (id, name, flag_url) e `user_national_teams` (N:N).
  - **Autenticação / autorização**: interceptor de auth JWT já expõe o `user_id` em RPCs protegidos — o serviço de busca o usa para inferir as favoritas. Não há papel de admin.
  - **Outros módulos internos**: domínios `auth` e `nationalteam` como molde para o domínio `match`; `clock.Clock` injetável para o "agora" determinístico do filtro temporal.
- **Conflitos / sobreposições detectados**: nenhum. A feature introduz um domínio novo (`match`) sem colidir com os existentes. Atenção: o glossário trata "Seleção Favorita" como o relacionamento `user_national_teams`, não como entidade — manter essa leitura.

---

## 11. Premissas e Decisões já tomadas

**Premissas:**

- [HIPÓTESE] A sigla de 3 letras (BRA/MEX) será armazenada como coluna `abbreviation` em `national_teams`, e a partida referenciará as seleções por FK (sem duplicar nome/bandeira/sigla na partida).
- [HIPÓTESE] "Próximo jogo" = partida com data/hora estritamente futura em relação ao "agora" do servidor (via `clock.Clock`); jogos passados não retornam.
- [HIPÓTESE] Uma partida que envolve duas seleções favoritas do mesmo usuário deve aparecer **uma única vez** na lista (deduplicação).
- [HIPÓTESE] O serviço retorna todos os jogos futuros das favoritas (sem limite), ordenados crescente por data/hora.

**Decisões já tomadas (fora de negociação):**

- "Não quero placar nem nada de status. Apenas os dados cadastrados mesmo."
- "Cadastraremos os jogos de grupos, oitavas, quartas, semi e final conforme vão acontecendo."
- Ingestão dos jogos: "Direto na base de dados."
- Fase modelada como campo único de texto (`stage`).
- Local registrado como estádio + cidade.
- Serviço de busca infere as seleções do token e retorna os jogos futuros.

---

## 12. Riscos e Pontos de Atenção

- **Risco de produto / aceitação**: card vazio para usuário cujas seleções já foram eliminadas (sem jogos futuros) → mitigação: o app trata lista vazia com um estado adequado (fora do escopo do backend).
- **Risco de escopo** (pode explodir?): pressão para adicionar placar/status/notificações → mitigação: recorte travado em "sem placar/status"; itens em v2 (seção 9).
- **Risco técnico ou operacional**: fuso horário dos jogos (México/EUA/Canadá) afeta o que é "futuro" e a hora exibida; cadastro manual no banco é propenso a erro humano (FK inválida, data malformada) → mitigação: definir armazenamento de data/hora (UTC) na tech-spec; constraints de FK e NOT NULL na tabela.
- **Risco de privacidade / segurança / compliance**: serviço expõe dados de jogo (públicos) atrelados às favoritas do usuário autenticado → baixo; manter o RPC protegido pelo interceptor de auth.

---

## 13. Dúvidas em Aberto

1. [DÚVIDA] Confirmar a hipótese de adicionar `abbreviation` em `national_teams` (vs. guardar a sigla na própria partida vs. derivar no front). Toca uma entidade existente.
2. [DÚVIDA] Como armazenar a data/hora da partida considerando os fusos das sedes (México/EUA/Canadá)? UTC no banco com exibição no cliente, ou hora local da sede? Impacta o filtro "futuro" e o "16h00" do card.
3. [DÚVIDA] O card/serviço precisa de limite ou paginação (próximos N), ou retornar todos os jogos futuros das favoritas é suficiente para a v1?
4. [DÚVIDA] O RPC de busca entra em um service novo (`MatchService`) ou é agregado a um service existente? (detalhe de contrato — pode ficar para a tech-spec).

---

## 14. Síntese do Brainstorm

- **Absorvido no escopo inicial (seção 8)**: tabela `matches` mínima (A1) com `stage` único (D1) e estádio+cidade (D2); cadastro manual direto no banco (B1); abreviação na seleção (D3, hipótese); RPC autenticado que infere favoritas e retorna jogos futuros (C1, E).
- **Descartado com justificativa**: placar/status (A2 — fora do recorte); RPC de admin e papel de administrador (B2 — fora do escopo do projeto); seed único (B3 — incompatível com mata-mata); campo grupo separado de fase (D — `stage` basta); busca por parâmetro de times (E — ignora favoritas persistidas).
- **Adiado para v2/v3**: placar/status ao vivo; paginação/limite no serviço (C2).
- **Provocações que mudaram o rumo**: a pergunta "como os jogos entram?" levou o usuário a escolher inserção direta no banco, eliminando a necessidade de RPC de escrita e de papel de admin — o que reduziu significativamente o escopo e a complexidade arquitetural.

---

## 15. Recomendação de Framework

### 15.1 Complexidade Observada

| Dimensão | Valor detectado | Confirmação |
|---|---|---|
| Amplitude — # rumos/US que sobreviveram | 2-3 (modelo de partida + serviço de busca; sigla na seleção como apoio) | confirmado |
| Personas | dev+1 (usuário final consome 1 RPC; operador cadastra no banco) | confirmado |
| Novidade | incremento (domínio `match` novo, mas replicando pattern já estabelecido) | confirmado |
| Decisão arquitetural transversal nova? | não — reusa fx/sqlc/migration embarcada/interceptor de auth/`clock` | inferido |

### 15.2 Framework Recomendado

**Escolhido**: `miniSpec`

**Justificativa**: a feature é um **incremento** que cria um domínio novo (`match`) coordenando 2-3 unidades coesas — migration (`matches` + `abbreviation` em `national_teams`), query sqlc com join temporal e RPC autenticado — replicando o pattern do projeto. Amplitude de 2-3 rumos + novidade de incremento (não ajuste pontual) pedem o contrato leve do miniSpec, sem o peso de PRD+TechSpec do SDD.

### 15.3 Alternativas Consideradas

**Por que NÃO TaskCard** (vizinho mais próximo, leve): ultrapassa um ajuste pontual — introduz uma entidade/domínio novo, um RPC novo e uma migration que cria tabela e altera `national_teams`, coordenando várias camadas. Não é uma mudança cirúrgica num único arquivo nem um CRUD repetindo pattern (o cadastro é manual no banco; a parte de API é só leitura especializada), então o CRUD fast-path também não se aplica.

**Por que NÃO SDD** (vizinho mais distante, pesado): não há múltiplas personas reais no sistema (não existe papel de admin; cadastro é manual), nenhuma decisão arquitetural transversal nova (reusa todos os mecanismos existentes) e não é greenfield de projeto. Rodar SDD aqui seria over-process para uma feature que cabe no contrato do miniSpec.

### 15.4 Próximo Passo

```bash
/agent-spec-minispec-generate-intent "próximos jogos das seleções favoritas — tabela de partidas (data/hora, 2 seleções, estádio, cidade, stage) e RPC autenticado que retorna os jogos futuros das seleções favoritas do usuário"
```

### 15.5 Quando Reconsiderar a Recomendação

- **Upgrade para SDD** se emergir: necessidade de RPC de admin com autorização por papel (decisão arquitetural nova → ADR); placar/status ao vivo com atualização em tempo real; suporte a múltiplos torneios.
- **Downgrade para TaskCard** se: a hipótese da `abbreviation` cair e o serviço virar um `SELECT` trivial sem nova coluna em `national_teams`, reduzindo a feature a uma única migration + uma query.

---

## 16. Checklist Final

- [x] Ideia resumida em uma frase clara
- [x] Esqueleto (seção 3) com ramos, validado com o usuário na Fase 1
- [x] Árvore de rumos (seção 4): direções candidatas + exemplo + viabilidade + escolha/poda
- [x] Rumos fora do escopo do projeto marcados como `[fora do escopo do projeto]`
- [x] Problema, público, escopo inicial e fora de escopo delimitados
- [x] Ancoramento (seção 10) preenchido com PRDs/capacidades concretos
- [x] Toda inferência marcada `[HIPÓTESE]`; dúvidas listadas como perguntas objetivas
- [x] Síntese (seção 14) registra absorvido / descartado / adiado
- [x] Complexidade (15.1) preenchida
- [x] Framework recomendado (15.2) justificado com 2 dimensões decisivas
- [x] Alternativas (15.3) explicam por que NÃO o vizinho mais próximo
- [x] Comando exato (15.4) escrito
- [x] Gatilhos (15.5) de reclassificação listados
- [x] Pronto para alimentar INTENT / TaskCard
