# TECH ALIGNMENT

> Registro de uma **discussão de arquitetura de alto nível** (Tree of Thought): pontos de decisão, alternativas viáveis avaliadas e decisões tomadas. Ponto de partida **decidido** para o SCOPE (miniSpec) — **não é especificação**. O arquiteto downstream herda as decisões e define o COMO completo.
>
> **Legenda:**
> - `[HIPÓTESE]` = alternativa/decisão a validar (sem âncora forte no projeto).
> - `[candidata a ADR]` = decisão transversal/evergreen — registrar via `/agent-spec-adr-create`.
> - `a critério do arquiteto` = ponto deixado em aberto para o SCOPE.

---

## 1. Metadados

- **Feature**: proximos-jogos-selecoes
- **Versão**: v1
- **Framework**: miniSpec
- **Variante**: backend
- **Documento de definição**: `docs/specs/features/proximos-jogos-selecoes/v1/intent.md`
- **Discovery lido**: `pre-refinement.md`
- **ADRs consultadas**: 0001 (driver SQLite pure-Go), 0002 (uber-fx + go-standard-layout), 0003 (auth JWT/clock), 0005 (schema em inglês)
- **Data**: 2026-05-29
- **Status**: Decidido — Pronto para SCOPE

---

## 2. Contexto Técnico

Introduzir um domínio de leitura `match` que materializa o calendário de partidas da Copa 2026 e expõe, via RPC autenticado, os jogos futuros das seleções favoritas do usuário. A partida é uma entidade mínima — instante de início, mandante e visitante (referências às seleções), estádio, cidade e fase textual — **sem placar nem status**. A escrita das partidas é manual, direto no banco, fase a fase; a feature entrega o **schema** e o **serviço de leitura**, não há caminho de escrita por aplicação nem papel de administrador.

Invariantes em jogo:

- **Inferência de identidade**: o conjunto de favoritas vem do `user_id` do token (interceptor JWT → `SubjectFromContext`), nunca de parâmetro do cliente — alinhado ao padrão anti-IDOR já adotado em `GetMe`.
- **Corte temporal determinístico**: o "agora" provém do `clock.Clock` injetado, jamais do relógio do banco (`CURRENT_TIMESTAMP`). O recorte de "futuro" é o **instante atual** — partida com início estritamente posterior a `clock.Now()`, conforme o INTENT.
- **Unicidade por partida**: um jogo que envolve duas favoritas do mesmo usuário aparece uma única vez.
- **Ordenação**: cronológica crescente (do mais próximo ao mais distante).

A feature replica o molde arquitetural do domínio `nationalteam` (handler → service → repository, interface no consumidor, bind no `fx.Module`) e reusa a relação N:N `user_national_teams` (`usuario-multiplas-selecoes/v1`) como fonte das favoritas e o padrão de coluna em `national_teams` (`national-team-flag-url/v1`) para a sigla.

---

## 3. Pontos de Decisão (esqueleto — Fase 1)

| # | Ponto de decisão | Status |
|---|------------------|--------|
| D1 | Fronteira do domínio de leitura (domínio novo vs agregar) | discutido |
| D2 | Representação de data/hora e fuso da partida | discutido |
| D3 | Local da sigla de 3 letras (BRA/MEX) | discutido |
| D4 | Filtro "futuro" + deduplicação | discutido |
| D5 | Integridade do cadastro manual (sem caminho de escrita por aplicação) | decisão direta |

---

## 4. Registro de Decisões Técnicas (Tree of Thought — Fase 2)

### D1 — Fronteira do domínio de leitura

**Por que decidir**: define onde o RPC de leitura mora e como o consumo cross-domain (favoritas do usuário) é estruturado.

**Alternativas avaliadas:**

- **D1.A — Novo domínio `match`** (handler/service/repository + `fx.Module` + serviço proto próprio), replicando o molde `nationalteam`.
  - _Exemplo:_ o match service declara uma **interface fina** para ler as favoritas (`user_national_teams`), bindada no módulo — mesmo padrão que `auth/service` usa para `NationalTeamRepository`.
  - _Prós:_ coeso; segue a regra "fx.Module por domínio" (di-layers) e a referência arquitetural explícita do `nationalteam`; consumo cross-domain cai no padrão interface-no-consumidor. · _Contras:_ mais boilerplate que agregar.
  - _Viabilidade:_ replica pattern estabelecido 1:1; reusa fx, sqlc, interceptor de auth e `clock`.
- **D1.B — Agregar a um serviço existente** (ex.: `NationalTeamService`).
  - _Exemplo:_ adicionar o RPC ao serviço de catálogo de seleções.
  - _Prós:_ menos arquivos. · _Contras:_ mistura responsabilidades (partida não é seleção nem auth); polui um catálogo read-only; contrato proto no pacote errado.
  - _Viabilidade:_ viável tecnicamente, mas fere a convenção de um domínio por módulo (di-layers).

**Decisão**: `D1.A` — domínio `match` novo, replicando o molde. Único caminho consistente com a regra "fx.Module por domínio" e com a referência `nationalteam`.
**Rejeitadas**: D1.B (desvio da convenção de coesão de domínio).
**Trade-off aceito**: mais boilerplate de camadas/módulo em troca de coesão e conformidade com o pattern do projeto.
**ADR?**: não — reusa ADR-0002, sem decisão transversal nova.

### D2 — Representação de data/hora e fuso da partida

**Por que decidir**: as sedes estão em fusos distintos (México/EUA/Canadá); a representação define a confiabilidade do filtro "futuro" e a exibição da hora do pontapé.

**Alternativas avaliadas:**

- **D2.A — Instante único em UTC** (`TIMESTAMP`→`time.Time`, como `created_at` hoje).
  - _Exemplo:_ jogo gravado como `2026-06-12T22:00:00Z`; comparação de instante com instante no corte temporal.
  - _Prós:_ corte "futuro" inequívoco; reusa o override `TIMESTAMP→time.Time` e a convenção SQLite existentes; `clock.Now()` (UTC) compara direto. · _Contras:_ a hora local da sede não é reconstruível do instante puro sem o fuso da sede — exibição fica no cliente.
  - _Viabilidade:_ reusa override do sqlc + `clock`; zero dependência nova.
- **D2.B — Instante UTC + identificador de fuso da sede (IANA)**.
  - _Exemplo:_ `2026-06-12T22:00:00Z` + `America/Mexico_City` → hora local da sede reconstruível no servidor.
  - _Prós:_ mantém o corte como comparação de instante e preserva a hora da sede deterministicamente. · _Contras:_ dado extra por jogo a preencher no cadastro manual; embute `time/tzdata` para portabilidade sem CGO.
  - _Viabilidade:_ reusa `TIMESTAMP`+`time.Time`; fuso é texto; Go stdlib resolve IANA.
- **D2.C — Hora local da sede sem fuso ("wall clock")**.
  - _Exemplo:_ grava `2026-06-12T16:00` (hora da sede), sem fuso.
  - _Prós:_ card mostra o valor direto. · _Contras:_ quebra o corte "futuro" entre fusos — comparar wall-time da sede com o "agora" do servidor dá resultado errado. Fragiliza o núcleo da feature.
  - _Viabilidade:_ reusa `TIMESTAMP`, mas mina a invariante temporal.

**Decisão**: `D2.A` — instante único em UTC. Minimalista e correto na invariante central; a exibição da hora local da sede fica no cliente (que conhece a sede via cidade/estádio retornados no card).
**Rejeitadas**: D2.B (adiado — upgrade documentado se o produto exigir hora da sede autoritativa no servidor); D2.C (quebra o corte temporal).
**Trade-off aceito**: a hora local **da sede** não é reconstruível no servidor a partir do instante UTC isolado — responsabilidade de exibição empurrada ao cliente.
**ADR?**: não — segue a convenção `created_at` (TIMESTAMP→time.Time) já existente.

### D3 — Local da sigla de 3 letras (BRA/MEX)

**Por que decidir**: o card precisa de sigla de 3 letras; `NationalTeam` hoje não a tem, e a partida referencia seleções por FK.

**Alternativas avaliadas:**

- **D3.A — Coluna de sigla em `national_teams`**.
  - _Exemplo:_ espelha exatamente o que `national-team-flag-url/v1` fez com a bandeira (adicionar coluna + reseed das 16 seleções).
  - _Prós:_ fonte única por seleção; segue precedente direto do projeto; sem duplicar na partida; a sigla é intrínseca à seleção. · _Contras:_ toca entidade existente (migration + reseed de 16 linhas).
  - _Viabilidade:_ replica o precedente de bandeira 1:1.
- **D3.B — Sigla na partida**.
  - _Exemplo:_ guardar sigla de mandante/visitante na própria partida.
  - _Prós:_ não toca `national_teams`. · _Contras:_ duplica dado intrínseco da seleção em cada jogo; cadastro manual repete e erra; desnormaliza.
  - _Viabilidade:_ conceitualmente errado (denormalização).
- **D3.C — Derivar no cliente**.
  - _Prós:_ zero mudança no backend. · _Contras:_ joga dado de domínio para o cliente; cada cliente reimplementa.
  - _Viabilidade:_ rejeitado já no pre-refinement ("pertence ao dado de domínio").

**Decisão**: `D3.A` — coluna de sigla em `national_teams`, espelhando o precedente da bandeira. A partida só carrega FKs; nome, bandeira e sigla vêm da seleção.
**Rejeitadas**: D3.B (desnormalização propensa a erro); D3.C (dado de domínio fora do servidor).
**Trade-off aceito**: uma migration que toca a entidade existente + reseed das 16 seleções — pattern estabelecido e de baixo risco aqui.
**ADR?**: não — feature-scoped, segue pattern existente (falha C1/C4 dos critérios de ADR).

### D4 — Filtro "futuro" + deduplicação

**Por que decidir**: retornar jogos futuros das favoritas, deduplicando o jogo que envolve duas favoritas, ordenado crescente — definindo onde cada regra mora e como o "agora" determinístico entra.

**Alternativas avaliadas:**

- **D4.A — Tudo numa query parametrizada (sqlc)**: filtra partidas onde mandante **ou** visitante está nas favoritas do `user_id`, com instante de início acima do corte (corte passado como parâmetro), ordenado por início.
  - _Exemplo:_ seleciona a **linha da partida** filtrando por pertinência às favoritas — uma linha por jogo, então a dedup é estrutural (não há multiplicação por join).
  - _Prós:_ dedup grátis (filtra a linha da partida, não faz join multiplicador); filtro no banco; corte como parâmetro preserva o tempo determinístico; reusa o padrão repository-fino-sobre-sqlc; um round-trip. · _Contras:_ query com subconsulta/IN; o corte **tem que** vir como parâmetro — `CURRENT_TIMESTAMP` do SQLite burlaria a regra do `clock.Clock`.
  - _Viabilidade:_ reusa sqlc + `clock`; nenhum mecanismo novo.
- **D4.B — Filtro/dedup no serviço (Go)**: repositório traz as partidas; o service filtra e deduplica em memória.
  - _Prós:_ regra explícita no service. · _Contras:_ traz jogos passados à memória sem necessidade; dedup manual; duplica a lógica de "favorita" em Go; menos eficiente.
  - _Viabilidade:_ reusa `clock`, mas tira o filtro do banco.

**Semântica do corte:** o corte é o **instante atual** — partida com início estritamente posterior a `clock.Now()`, exatamente como o INTENT define ("partida com data/hora futura em relação ao momento atual"). Como o instante das partidas é gravado em UTC (D2.A), a comparação é instante × instante, independente de fuso.

- _Exemplo:_ agora = `2026-06-12T17:00Z`; jogo às `2026-06-12T13:00Z` (já iniciado) **não aparece**; jogo às `2026-06-12T22:00Z` aparece.
- _Mecânica:_ o servidor passa `clock.Now()` (instante UTC determinístico) como parâmetro de corte da query (D4.A).

**Decisão**: `D4.A` + corte pelo **instante atual** (`clock.Now()`). Filtro e dedup numa única query parametrizada; o corte vem do `clock.Clock`, nunca de `CURRENT_TIMESTAMP`. Service permanece orquestrador fino.
**Rejeitadas**: D4.B (ineficiente, duplica lógica de favorita e dedup); corte por **início do dia local do usuário** com fuso enviado pelo cliente (adiado — mantém-se a definição do INTENT por instante; ver seção 7, ponto de evolução futura).
**Trade-off aceito**: um jogo já iniciado no dia corrente não aparece (segue a definição estrita do INTENT por instante); a exibição da hora local fica no cliente (D2.A).
**ADR?**: não — feature-scoped (consumidor único); a disciplina de clock injetado já é coberta pelo contexto da ADR-0003 e pela rule de clock.

### D5 — Integridade do cadastro manual (decisão direta)

**Decisão direta**: como não há caminho de escrita por aplicação validando os dados, a **persistência** garante a integridade por constraints de schema (FK das duas seleções para `national_teams` + NOT NULL nos campos do card), seguindo o padrão das tabelas existentes (`users`, `user_national_teams`) e dependendo de `foreign_keys=ON`. Sem alternativas avaliadas — escolha única alinhada ao pattern do projeto. Invariantes adicionais (ex.: CHECK de mandante ≠ visitante, validação de formato de data) ficam `a critério do arquiteto do SCOPE` (ver seção 7).

---

## 5. Decisões Candidatas a ADR (transversais / evergreen)

Nenhuma — todas as decisões são feature-scoped. As decisões reusam ADRs ativas (0001, 0002, 0003, 0005) sem estendê-las nem contradizê-las.

---

## 6. Restrições e Invariantes Técnicas

- **Favoritas inferidas do token**: o serviço descobre as seleções pelo `user_id` do JWT (`SubjectFromContext`), nunca de parâmetro do cliente (anti-IDOR).
- **Corte temporal determinístico**: o "agora" vem do `clock.Clock` injetado; proibido `CURRENT_TIMESTAMP` do banco no filtro.
- **Corte = instante atual**: partida com início estritamente posterior a `clock.Now()`, conforme a definição do INTENT. Comparação instante × instante (partidas em UTC), independente de fuso.
- **Instante das partidas em UTC**: gravado como `TIMESTAMP`→`time.Time`, consistente com `created_at`.
- **Unicidade por partida**: jogo com duas favoritas aparece uma vez (dedup estrutural na query).
- **Ordenação cronológica crescente**.
- **Sigla intrínseca à seleção**: partida referencia seleções por FK; nome, bandeira e sigla vêm de `national_teams`.
- **Sem placar/status; sem limite/paginação na v1**; sem RPC de escrita nem papel de admin.
- **Stack inviolável**: driver `modernc.org/sqlite` sem CGO (ADR-0001); domínio em `fx.Module` próprio (ADR-0002); schema/código/proto em inglês (ADR-0005); query parametrizada via sqlc; erros como `status.Error(codes.X)` no service, handler mapper puro.

---

## 7. Pontos em Aberto (a critério do arquiteto do SCOPE)

1. Invariantes de integridade adicionais da partida além de FK/NOT NULL — ex.: CHECK de mandante ≠ visitante, validação de formato/coerência da data no cadastro manual — `a critério do arquiteto`.
2. Contrato proto exato do RPC (nome do serviço/método, shape do request/response, mapeamento dos campos do card) — `a critério do arquiteto`.
3. Estratégia fina de migração (numeração, par up/down, reseed das 16 seleções para a sigla) — `a critério do arquiteto`.

**Evolução futura (fora da v1):** corte por **início do dia local do usuário** com o fuso enviado pelo cliente na requisição — manteria visíveis os jogos já iniciados no dia local corrente. Adiado para preservar a definição do INTENT por instante; reavaliar se o produto exigir essa semântica.

---

## 8. Checklist Final

- [x] Metadados preenchidos (framework, variante, documento de entrada, discovery, ADRs consultadas)
- [x] Contexto técnico reescrito com vocabulário de arquiteto (sem narrativa, sem implementação)
- [x] Pontos de decisão (seção 3) listados
- [x] Cada ponto com 2-3 alternativas avaliadas (exemplo + prós/contras + viabilidade) OU marcado como decisão direta
- [x] Cada decisão registrada com escolhida + rejeitadas + justificativa + trade-off aceito
- [x] Conflitos com ADR/stack tratados como ponto de discussão (não descartados)
- [x] Decisões transversais listadas como candidatas a ADR (seção 5) — não criadas direto
- [x] Restrições e invariantes (seção 6) registradas
- [x] Pontos não decididos em "a critério do arquiteto" (seção 7)
- [x] Sem detalhes de implementação (endpoints, schemas, arquivos, tabelas, campos, middlewares)
- [x] Arquivo salvo como `tech-alignment.md` no path resolvido via `tech_alignment.path`
