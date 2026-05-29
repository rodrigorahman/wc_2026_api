# SCOPE -- MiniSpec (Backend)

> **Variante**: backend
> **Stack**: Go
> **Feature**: proximos-jogos-selecoes · **Versão**: v1
> **Entrada**: `docs/specs/features/proximos-jogos-selecoes/v1/intent.md` · **Tech Alignment**: `tech-alignment.md` (decisões D1–D5 herdadas)

## 1. O que está incluído
- [x] Novo domínio de leitura `match` (camadas `handler → service → repository` + `fx.Module` próprio), replicando o molde `nationalteam` (D1.A).
- [x] Tabela `matches` representando o calendário da Copa 2026: instante de início (UTC), seleção mandante e visitante (FKs para `national_teams`), estádio, cidade e fase textual — **sem placar nem status**.
- [x] Coluna `code` (sigla FIFA de 3 letras, ex.: `BRA`, `MEX`) em `national_teams`, com backfill das 16 seleções já semeadas (D3.A — espelha o precedente de `flag_url`).
- [x] RPC **autenticado** `MatchService.ListUpcomingMatches` que infere as seleções favoritas do `user_id` do token (`SubjectFromContext`) e devolve os jogos futuros dessas seleções.
- [x] Filtro "futuro" + deduplicação + ordenação cronológica crescente em **uma única query parametrizada** (sqlc), com o corte = `clock.Now()` (instante UTC) passado como parâmetro (D4.A).
- [x] Cada partida retornada traz: instante de início, mandante e visitante (cada um com `id`, `name`, `flag_url`, `code`), estádio, cidade e fase.
- [x] Comportamentos observáveis do INTENT: jogos passados não aparecem; partida com duas favoritas aparece uma única vez; usuário sem jogos futuros recebe lista vazia.
- [x] Registro do `match.Module` no composition root (`cmd/server`) e no helper de E2E (`internal/testutil/bufconn.go`), com o RPC fora da lista de métodos públicos (protegido por JWT, como `GetMe`).

---

## 2. O que está fora do escopo
- [ ] Placar e status da partida (ao vivo/encerrado) — adiado para eventual v2.
- [ ] RPC ou serviço de **escrita/edição** de partidas e qualquer papel de administrador — partidas são inseridas manualmente no banco, fase a fase.
- [ ] Limite, paginação ou filtros adicionais sobre o resultado — v1 retorna todos os jogos futuros das favoritas.
- [ ] Fuso horário da sede / hora local do estádio reconstruída no servidor (D2.B rejeitado) — a exibição da hora local fica no cliente, que conhece a sede via cidade/estádio retornados.
- [ ] Corte por "início do dia local do usuário" com fuso enviado pelo cliente (D4 — adiado; mantém-se o corte por instante do INTENT).
- [ ] Alteração da relação de favoritas (`user_national_teams`) — esta feature apenas **consome** a relação já persistida por `usuario-multiplas-selecoes/v1`.
- [ ] Exposição da `code` ou de `flag_url` por novos RPCs no domínio `nationalteam` — o `code` é consumido somente pela query de partidas.

---

## 3. Definições Técnicas

> **Conformidade com ADRs ativas** (inventário completo em §5):
> - **ADR-0001** — toda persistência usa o driver `modernc.org/sqlite`; migrations e queries não introduzem dependência CGO.
> - **ADR-0002** — `match` é um `fx.Module` por domínio; o bind concreto→interface ocorre no módulo (adapter de sentinela), nunca no consumidor.
> - **ADR-0003** — o RPC lê o `sub` do contexto (anti-IDOR), nunca de parâmetro; o corte temporal vem do `clock.Clock` injetado, jamais de `CURRENT_TIMESTAMP`.
> - **ADR-0005** — `matches`, coluna `code` e o contrato proto usam identificadores em **inglês**, sem bridge; aliases `home_*`/`away_*` na query servem para **desambiguar o JOIN**, não para traduzir idioma.

### 3.1 Endpoints / Rotas

API é gRPC. Novo serviço `MatchService` em `wc2026.match.v1`.

| Método (RPC)          | Serviço        | Descrição                                                                 | Auth                  | Códigos de status                       |
|-----------------------|----------------|---------------------------------------------------------------------------|-----------------------|------------------------------------------|
| `ListUpcomingMatches` | `MatchService` | Lista os jogos futuros das seleções favoritas do usuário autenticado, em ordem cronológica crescente. Favoritas inferidas do token. | **Protegido (JWT)**   | `OK`; `Unauthenticated` (token ausente/inválido); `Internal` (falha de persistência) |

Contrato proto (`api/proto/wc2026/match/v1/match.proto`):

```proto
syntax = "proto3";

package wc2026.match.v1;

import "google/protobuf/timestamp.proto";

option go_package = "github.com/rodrigorahman/wc_2026_api/gen/wc2026/match/v1;matchv1";

service MatchService {
  rpc ListUpcomingMatches(ListUpcomingMatchesRequest) returns (ListUpcomingMatchesResponse);
}

message ListUpcomingMatchesRequest {}

message ListUpcomingMatchesResponse {
  repeated Match matches = 1;
}

message Match {
  string id = 1;
  google.protobuf.Timestamp kickoff = 2;
  MatchTeam home_team = 3;
  MatchTeam away_team = 4;
  string stadium = 5;
  string city = 6;
  string stage = 7;
}

message MatchTeam {
  string id = 1;
  string name = 2;
  string flag_url = 3;
  string code = 4;
}
```

- `ListUpcomingMatchesRequest` é **vazio** — as favoritas vêm do token (não há parâmetro de cliente, anti-IDOR). Sem regra `buf.validate.*` (nenhum campo de entrada).
- `MatchTeam` é uma mensagem própria de `match.v1` (carrega `code`, ausente em `nationalteam.v1.NationalTeam`) — mantém o contrato `match` auto-contido, sem import cross-package.

#### 3.1.1 Exemplo de Payload (PUT/PATCH parcial)

N/A — o único RPC é de **leitura**, com request vazio. Não há verbo de escrita nem atualização parcial nesta feature.

### 3.2 Banco de Dados

#### Tabelas

| Tabela           | Colunas                                                                 | Tipos                                                  | Constraints                                                                                          | Índices                                  |
|------------------|-------------------------------------------------------------------------|--------------------------------------------------------|------------------------------------------------------------------------------------------------------|------------------------------------------|
| `matches` (novo) | `id`, `kickoff`, `home_team_id`, `away_team_id`, `stadium`, `city`, `stage` | `TEXT`, `TIMESTAMP`, `TEXT`, `TEXT`, `TEXT`, `TEXT`, `TEXT` | `id` PK; todas NOT NULL; `home_team_id`/`away_team_id` → `national_teams(id)`; `CHECK (home_team_id <> away_team_id)` | PK em `id` (sem índice secundário — ver §5) |
| `national_teams` (modificar) | `+ code`                                                    | `TEXT`                                                 | `NOT NULL DEFAULT ''` (preenchida no backfill das 16 seleções)                                       | (inalterados)                            |

- `kickoff` é o **instante UTC** de início, mapeado via o override `TIMESTAMP → time.Time` já existente no `sqlc.yaml` (mesma convenção de `created_at`). A comparação do corte é instante × instante, independente de fuso (D2.A).
- A integridade do cadastro manual é garantida por schema: FK das duas seleções (depende de `foreign_keys=ON`), NOT NULL em todos os campos do card e `CHECK` mandante ≠ visitante (D5 + ponto aberto #1 resolvido). Não há caminho de escrita por aplicação validando os dados.

#### Migrações

| Versão | Arquivo | Descrição |
|--------|---------|-----------|
| 000006 | `000006_add_code_to_national_teams.up.sql` / `.down.sql` | `ALTER TABLE national_teams ADD COLUMN code TEXT NOT NULL DEFAULT ''` + `UPDATE` das 16 seleções com a sigla FIFA. Down: `ALTER TABLE national_teams DROP COLUMN code`. Espelha 000005. |
| 000007 | `000007_create_matches.up.sql` / `.down.sql` | `CREATE TABLE matches (...)` com FKs, NOT NULL e `CHECK`. Down: `DROP TABLE matches`. |

Backfill das siglas (migration 000006), pareando com o seed 000002/000005:

| Seleção (`name`) | `code` | Seleção (`name`) | `code` |
|---|---|---|---|
| Brasil | BRA | Uruguai | URU |
| Argentina | ARG | Bélgica | BEL |
| França | FRA | Croácia | CRO |
| Alemanha | GER | México | MEX |
| Espanha | ESP | Estados Unidos | USA |
| Inglaterra | ENG | Japão | JPN |
| Portugal | POR | Coreia do Sul | KOR |
| Países Baixos | NED | Itália | ITA |

> A coluna `code` adicionada à tabela faz o sqlc regenerar o model `sqlc.NationalTeam` com um campo `Code` — **inócuo** para o domínio `nationalteam` (suas queries continuam `SELECT id, name, flag_url`; o `toNationalTeam` ignora o campo novo). Nenhuma alteração no código de `nationalteam`.

#### Queries (sqlc) — `internal/infra/db/queries/matches.sql`

```sql
-- name: ListUpcomingMatchesByUser :many
SELECT
    m.id,
    m.kickoff,
    m.stadium,
    m.city,
    m.stage,
    home.id       AS home_id,
    home.name     AS home_name,
    home.flag_url AS home_flag_url,
    home.code     AS home_code,
    away.id       AS away_id,
    away.name     AS away_name,
    away.flag_url AS away_flag_url,
    away.code     AS away_code
FROM matches m
JOIN national_teams home ON home.id = m.home_team_id
JOIN national_teams away ON away.id = m.away_team_id
WHERE m.kickoff > sqlc.arg(cutoff)
  AND (
        m.home_team_id IN (SELECT national_team_id FROM user_national_teams WHERE user_id = sqlc.arg(user_id))
     OR m.away_team_id IN (SELECT national_team_id FROM user_national_teams WHERE user_id = sqlc.arg(user_id))
  )
ORDER BY m.kickoff ASC;
```

- **Dedup estrutural**: filtra a **linha da partida** (não há join multiplicador com as favoritas) — um jogo com duas favoritas aparece uma única vez.
- **Corte determinístico**: `sqlc.arg(cutoff)` recebe `clock.Now()` do service; nunca `CURRENT_TIMESTAMP`.
- **Reuso de `user_id`** via `sqlc.arg(user_id)` nas duas subconsultas gera um único parâmetro nomeado.
- Aliases `home_*`/`away_*` desambiguam as duas junções a `national_teams` (necessidade do sqlc), **não** traduzem idioma (conforme ADR-0005).

### 3.3 Services / Regras de Negócio
- [x] **`repository.MatchRepository`** (concreto): thin wrapper sobre sqlc. Expõe `ListUpcomingMatchesByUser(ctx, userID string, cutoff time.Time) ([]Match, error)`, mapeando as linhas para os tipos de domínio `Match`/`MatchTeam`. Sem regra de negócio.
- [x] **`service.MatchService`**: orquestrador fino. Declara a interface `MatchRepository` (no consumidor); injeta `clock.Clock`. `ListUpcomingMatches(ctx, userID string) ([]Match, error)` calcula `cutoff := clk.Now()` e delega. Lista vazia **não** é erro (sem favoritas / sem jogos futuros → slice vazio). Nenhuma sentinela nova necessária.
- [x] **`handler.MatchHandler`**: mapper proto↔domínio puro. Declara a interface `MatchService` (no consumidor). Lê o `sub` via `interceptor.SubjectFromContext`; ausência → `status.Error(codes.Unauthenticated, ...)` (mesmo padrão de `GetMe`). Erro do service → `codes.Internal` (sem retraduzir códigos). Mapeia `Match`/`MatchTeam` de domínio para proto, `kickoff` via `timestamppb.New`.
- [x] **`match.Module`** (`fx.Module("match", ...)`): provê repository, service e handler; faz o bind concreto→interface no módulo via adapter fino (como `nationalteam`). **Consome** `clock.Clock` já provido no grafo (não re-provê — evita colisão de tipo no fx).

### 3.4 Integrações Externas (clientes / eventos)

| Integração | Tipo | Direção | Auth |
|------------|------|---------|------|
| N/A        | —    | —       | —    |

Sem integrações externas. Reuso interno: relação `user_national_teams` (favoritas) e `national_teams` (nome/bandeira/sigla).

### 3.5 Logs / Observabilidade (resumo)
- **Logs estruturados**: nenhum log novo específico. O RPC é coberto pelo interceptor de logging existente (`internal/infra/logger`) na cadeia. Token/credencial nunca logados.
- **Métricas chave**: N/A — não há instrumentação de métricas no projeto nesta versão.
- **Tracing**: N/A.
- **Alertas**: N/A.

### 3.6 Feature Flags

| Flag | Propósito | Default |
|------|-----------|---------|
| N/A  | —         | —       |

Sem feature flags — não há mecanismo de flags no projeto.

### 3.7 Versionamento de API
- **Estratégia**: versão no pacote proto / path (`wc2026.match.v1`), igual aos domínios existentes.
- **Versão atual**: v1.
- **Política de breaking changes**: novo serviço aditivo; não altera contratos existentes.

### 3.8 Dependências de Pacotes

| Pacote | Versão | Motivo |
|--------|--------|--------|
| (nenhum novo) | — | Reusa stack existente: `modernc.org/sqlite`, `uber-fx`, `sqlc`, `golang-migrate`, `google.golang.org/protobuf` (`timestamppb`), `grpc`. |

### 3.9 Visão em Árvore

```
api/proto/wc2026/
└── match/v1/
    └── match.proto                                          [N]

gen/wc2026/match/v1/                                         [N]  (gerado via make proto)

internal/domain/match/                                       [N]
├── module.go                                                [N]
├── module_test.go                                           [N]
├── handler/
│   ├── match_handler.go                                     [N]
│   └── match_handler_test.go                                [N]
├── service/
│   ├── match_service.go                                     [N]
│   └── match_service_test.go                                [N]
└── repository/
    ├── match_repository.go                                  [N]
    └── match_repository_test.go                             [N]

internal/infra/db/
├── migrations/
│   ├── 000006_add_code_to_national_teams.up.sql             [N]
│   ├── 000006_add_code_to_national_teams.down.sql           [N]
│   ├── 000007_create_matches.up.sql                         [N]
│   └── 000007_create_matches.down.sql                       [N]
├── queries/
│   └── matches.sql                                          [N]
└── sqlc/                                                    [M]  (regenerado via make sqlc)

internal/server/server.go                                    [M]
internal/testutil/bufconn.go                                 [M]
cmd/server/main.go                                           [M]

test/e2e/match_e2e_test.go                                   [N]
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

### 3.10 Arquivos Envolvidos

| Arquivo | Ação | Descrição |
|---------|------|-----------|
| `api/proto/wc2026/match/v1/match.proto` | criar | Contrato `MatchService.ListUpcomingMatches` + mensagens `Match`/`MatchTeam`. |
| `gen/wc2026/match/v1/**` | criar (gerado) | Stubs gerados via `make proto`. Não editar à mão. |
| `internal/infra/db/migrations/000006_add_code_to_national_teams.up.sql` | criar | `ADD COLUMN code` + backfill das 16 siglas. |
| `internal/infra/db/migrations/000006_add_code_to_national_teams.down.sql` | criar | `DROP COLUMN code`. |
| `internal/infra/db/migrations/000007_create_matches.up.sql` | criar | `CREATE TABLE matches` (FKs, NOT NULL, CHECK mandante≠visitante). |
| `internal/infra/db/migrations/000007_create_matches.down.sql` | criar | `DROP TABLE matches`. |
| `internal/infra/db/queries/matches.sql` | criar | Query `ListUpcomingMatchesByUser` (filtro futuro + dedup + ordenação). |
| `internal/infra/db/sqlc/**` | modificar (gerado) | Regenerado via `make sqlc` (novo model `Match`, query tipada, campo `Code` em `NationalTeam`). Não editar à mão. |
| `internal/domain/match/repository/match_repository.go` | criar | Repository concreto sobre sqlc + tipos de domínio `Match`/`MatchTeam` + mapper. |
| `internal/domain/match/repository/match_repository_test.go` | criar | Teste de integração (SQLite real via `TestNewDB`): futuro, dedup, ordenação, vazio. |
| `internal/domain/match/service/match_service.go` | criar | Service orquestrador + interface `MatchRepository` + injeção de `clock.Clock`. |
| `internal/domain/match/service/match_service_test.go` | criar | Teste com `clock.Clock` fixo: assegura que o corte passado ao repo é `clk.Now()`; caminho feliz + erro do repo. |
| `internal/domain/match/handler/match_handler.go` | criar | Handler mapper + interface `MatchService` + leitura de `SubjectFromContext`. |
| `internal/domain/match/handler/match_handler_test.go` | criar | Teste do mapper: sub ausente → `Unauthenticated`; mapeamento proto; erro → `Internal`. |
| `internal/domain/match/module.go` | criar | `fx.Module("match", ...)` com binds concreto→interface. |
| `internal/domain/match/module_test.go` | criar | Smoke test do wiring fx do módulo (espelha `nationalteam/module_test.go`). |
| `internal/server/server.go` | modificar | Registrar `MatchService` no `*grpc.Server` (novo param `*matchhandler.MatchHandler` + `RegisterMatchServiceServer`). `ListUpcomingMatches` **não** entra em `providePublicMethods` (protegido). |
| `cmd/server/main.go` | modificar | Adicionar `match.Module` ao `fx.New`. |
| `internal/testutil/bufconn.go` | modificar | Adicionar `match.Module` à montagem fx do helper de E2E. |
| `test/e2e/match_e2e_test.go` | criar | E2E: sem token → `Unauthenticated`; usuário autenticado sem favoritas → lista vazia. |

> **Legenda de Ações:** `criar` | `modificar` | `remover`

---

## 4. Critérios de Aceite (técnicos)
- [ ] `ListUpcomingMatches` chamado **sem** token (ou token inválido) retorna `codes.Unauthenticated` e o handler não é alcançado (RPC ausente de `providePublicMethods`). _(E2E)_
- [ ] Usuário autenticado com favoritas que possuem jogos futuros recebe as partidas em **ordem cronológica crescente** (`kickoff` ASC). _(integração — repository)_
- [ ] Partidas com `kickoff` **anterior ou igual** ao corte (`clock.Now()`) **não** retornam; apenas `kickoff` estritamente futuro aparece. _(integração — repository, com instantes ao redor de um corte fixo)_
- [ ] Uma partida que envolve **duas** favoritas do mesmo usuário aparece **uma única vez** (dedup estrutural). _(integração — repository)_
- [ ] Usuário autenticado **sem favoritas** ou cujas favoritas **não têm jogos futuros** recebe **lista vazia** (não erro). _(E2E + integração)_
- [ ] Cada `Match` retornado carrega `kickoff`, mandante e visitante com `id`/`name`/`flag_url`/`code`, `stadium`, `city`, `stage` corretos para os dados cadastrados. _(integração — repository)_
- [ ] O corte temporal usado na query é o `clock.Now()` injetado (não `CURRENT_TIMESTAMP`): com `clock.Clock` fixo, alterar o relógio muda o conjunto retornado de forma determinística. _(service + integração)_
- [ ] A tabela `matches` rejeita partida com `home_team_id == away_team_id` (CHECK) e com FK inexistente (`foreign_keys=ON`). _(integração — repository/migração)_
- [ ] As 16 seleções semeadas têm `code` preenchido (3 letras) após a migração 000006; nenhuma fica vazia. _(integração — migração)_
- [ ] `make sqlc`, `make proto` e `make test` verdes; `make build-all` (CGO off) permanece verde.

---

## 5. Observações

### ADRs Aplicáveis nesta Feature
| ADR | Status | Classificação | Como a feature obedece |
|-----|--------|---------------|------------------------|
| ADR-0001 (driver modernc pure-Go, sem CGO) | accepted | **APLICÁVEL** | Migrations/queries reusam o driver `sqlite` (modernc); nenhuma dependência CGO nova; `make build-all` cobre a portabilidade. |
| ADR-0002 (uber-fx + go-standard-layout) | accepted | **APLICÁVEL** | `match` é `fx.Module` por domínio; bind concreto→interface no módulo (adapter de sentinela); interface declarada no consumidor (service/handler). |
| ADR-0003 (JWT HS256 + clock injetado + acesso protegido) | accepted | **APLICÁVEL** | RPC protegido lê `sub` do contexto (anti-IDOR); corte temporal via `clock.Clock`, nunca relógio do banco. |
| ADR-0005 (schema em inglês, sem bridge) | accepted | **APLICÁVEL** | `matches`, coluna `code` e contrato proto em inglês; aliases `home_*`/`away_*` desambiguam JOIN (não traduzem idioma). |
| ADR-0004 (schema pt-BR + rename sqlc) | superseded | **N/A** | Substituída pela ADR-0005; ignorada. |

### Pontos de atenção
- **Acoplamento ao `clock.Clock` do grafo**: hoje o `clock.New` é provido dentro de `auth.Module`. O `match.Module` **consome** esse `clock.Clock` do escopo raiz (não pode re-prover — causaria colisão de tipo no fx). Funciona porque o provider de `auth` não é privado. Trade-off aceito: leve dependência implícita de ordem de módulos no `fx.New`. _Evolução possível (fora do escopo): extrair `clock.New` para um módulo de infra compartilhado._
- **`SubjectFromContext` cross-domain**: o handler de `match` importa `internal/domain/auth/interceptor` para ler o subject — é o ponto canônico onde a chave de contexto vive, e o tech-alignment já endossou esse caminho. Sem refactor de `auth` (fora do escopo).
- **Cobertura de dados ponta-a-ponta**: o helper `TestNewBufconnServer` não expõe o `*sql.DB` nem semeia partidas, então o **caminho-feliz com dados** de partida é coberto por **testes de integração** (`TestNewDB` permite inserir `matches` reais); o **E2E** cobre o stack autenticado (sem token → `Unauthenticated`; autenticado sem favoritas → vazio). Decisão deliberada para não inflar `testutil`.
- **Sem índice secundário em `matches`**: o dataset é fixo e pequeno (~104 jogos no torneio). Filtro/ordenação por `kickoff` sobre full scan é trivial; índice seria otimização especulativa (YAGNI). Reavaliar se o volume justificar.
- **`code` regenera o model `sqlc.NationalTeam`** (ganha campo `Code`), mas sem efeito no domínio `nationalteam` (queries e mapper inalterados).

### Candidatos a ADR
- **Nenhum.** Todas as decisões são feature-scoped e reusam ADRs ativas (0001/0002/0003/0005) sem estendê-las. Avaliação dos 5 critérios canônicos para as decisões candidatas (corte por `clock` injetado; RPC autenticado anti-IDOR; coluna `code`): falham **C1 (transversal)** e/ou **C4 (surpreendente sem contexto)** — já cobertas pelo contexto da ADR-0003 e pelo precedente de `flag_url`. Registro apenas como decisão técnica, sem candidatura.

### Glossário
- O glossário global ainda descreve "Seleção Favorita" como FK 1:1 (`usuarios.selecao_id`), porém o schema real evoluiu para N:N via `user_national_teams` (migration 000004) — alinhado ao que o tech-alignment assume. **Sinalizado**: recomenda-se rodar `/agent-spec-challenge-spec` para atualizar o glossário (favoritas = relação N:N; novo termo de domínio **Match/Partida** e **code/sigla**), mas isso está fora do escopo desta feature.
