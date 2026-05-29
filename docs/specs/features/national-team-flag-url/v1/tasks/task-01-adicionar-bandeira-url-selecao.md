# TASKCARD - Execucao Rapida (com Guardrails LLM)

## 1. Identificacao
- **ID**: TC-001
- **Nome da Task**: Adicionar campo de URL da bandeira (`bandeira_url`) ao domínio NationalTeam (tabela `selecoes`) e expor no contrato gRPC
- **model**: opus
- **risk**: high
- **gates**: [qa, tech_review]   <!-- inferido: critical_paths db_migrations + api_contracts -->
- **Variante**: backend
- **source**: no_discovery
- **Responsavel**: agent-spec-taskcard-run
- **Data**: 2026-05-29
- **Status**: A Fazer
- **Dependencias**: nenhuma
- **Relacionados**: ADR-0001, ADR-0002, ADR-0004

---

## 2. Contexto
A tabela `selecoes` hoje tem apenas `id` e `nome`. O sistema precisa exibir a bandeira de cada seleção em diversos pontos, então é necessário persistir a URL completa da imagem da bandeira e expô-la no contrato gRPC do domínio NationalTeam (que hoje retorna apenas `id` e `name`). As 16 seleções já populadas pelo seed (migration `000002`) devem receber a URL no mesmo momento da migração.

---

## 3. Objetivo da Task
Adicionar a coluna `bandeira_url` (TEXT NOT NULL) à tabela `selecoes`, popular as 16 seleções existentes com URLs reais de bandeira (flagcdn.com), e propagar o campo por toda a leitura do domínio: query sqlc → repository → service → handler → mensagem proto `NationalTeam` (novo campo `flag_url`), de modo que `ListNationalTeams` passe a retornar a URL da bandeira.

---

## 4. Escopo
### 4.1 Inclui
- [ ] Migration aditiva `000005` que adiciona `bandeira_url` a `selecoes` e faz backfill das 16 seleções existentes.
- [ ] Atualização da query `ListNationalTeams` (e `GetNationalTeamByID`) em `selecoes.sql` para incluir `bandeira_url`.
- [ ] Entrada `bandeira_url: "FlagURL"` no bloco `rename` do `sqlc.yaml` (ponte de idioma ADR-0004).
- [ ] Regeneração do sqlc (`make sqlc`) e do proto (`make proto`).
- [ ] Novo campo `flag_url` na mensagem `NationalTeam` do proto.
- [ ] Propagação do campo `FlagURL` nos tipos de domínio de repository, service e no mapeamento do handler.

### 4.2 Fora do escopo
- [ ] Endpoint/RPC de escrita (criar/editar seleção ou sua bandeira) — não existe hoje e não será criado.
- [ ] Validação de formato/acessibilidade da URL em runtime.
- [ ] Qualquer alteração no domínio `auth` ou em `usuario_selecoes`.
- [ ] Upload/armazenamento de imagem — apenas a URL é persistida.

---

## 5. Descricao de Execucao (COMO fazer)

Siga o pattern já estabelecido no domínio `nationalteam`. A mudança é aditiva e cirúrgica.

**1. Migration `000005` (criar par up/down)** — `internal/db/migrations/`:
- `up`: adicionar a coluna e fazer o backfill na mesma migration (a tabela já contém 16 linhas, então uma coluna `NOT NULL` exige valor para as linhas existentes).
  - `ALTER TABLE selecoes ADD COLUMN bandeira_url TEXT NOT NULL DEFAULT '';`
  - Em seguida, um `UPDATE` por seleção, casando pelo `id` do seed (`000002_seed_selecoes.up.sql`), com a URL `https://flagcdn.com/w320/{codigo}.png`:

  | id (seed 000002)                     | nome           | codigo flagcdn |
  | ------------------------------------ | -------------- | -------------- |
  | a1f3c5e7-0001-4000-8000-000000000001 | Brasil         | br             |
  | a1f3c5e7-0002-4000-8000-000000000002 | Argentina      | ar             |
  | a1f3c5e7-0003-4000-8000-000000000003 | França         | fr             |
  | a1f3c5e7-0004-4000-8000-000000000004 | Alemanha       | de             |
  | a1f3c5e7-0005-4000-8000-000000000005 | Espanha        | es             |
  | a1f3c5e7-0006-4000-8000-000000000006 | Inglaterra     | gb-eng         |
  | a1f3c5e7-0007-4000-8000-000000000007 | Portugal       | pt             |
  | a1f3c5e7-0008-4000-8000-000000000008 | Países Baixos  | nl             |
  | a1f3c5e7-0009-4000-8000-000000000009 | Itália         | it             |
  | a1f3c5e7-0010-4000-8000-000000000010 | Uruguai        | uy             |
  | a1f3c5e7-0011-4000-8000-000000000011 | Bélgica        | be             |
  | a1f3c5e7-0012-4000-8000-000000000012 | Croácia        | hr             |
  | a1f3c5e7-0013-4000-8000-000000000013 | México         | mx             |
  | a1f3c5e7-0014-4000-8000-000000000014 | Estados Unidos | us             |
  | a1f3c5e7-0015-4000-8000-000000000015 | Japão          | jp             |
  | a1f3c5e7-0016-4000-8000-000000000016 | Coreia do Sul  | kr             |

  - O `DEFAULT ''` é deliberado: é a forma mais simples e segura em SQLite de adicionar coluna `NOT NULL` a uma tabela já populada sem rebuild de tabela (que seria arriscado pela FK `usuario_selecoes.selecao_id → selecoes.id`). Como não existe query de `INSERT` de seleção no código (apenas seed e leitura), não há caminho de inserção que dependa do default. Registrado em §11.
- `down`: `ALTER TABLE selecoes DROP COLUMN bandeira_url;` (suportado pelo `modernc.org/sqlite`).

**2. Ponte de idioma (`sqlc.yaml`)**: adicionar `bandeira_url: "FlagURL"` no bloco `rename` (ADR-0004 — nunca mapear idioma à mão).

**3. Queries (`internal/db/queries/selecoes.sql`)**: incluir `bandeira_url` no `SELECT` de `ListNationalTeams` e de `GetNationalTeamByID`, mantendo as duas casando com o model completo da tabela (evita sqlc gerar struct `Row` separada). Queries permanecem parametrizadas.

**4. Regenerar código** (NÃO editar à mão): `make sqlc` (gera `internal/db/sqlc/**`) e `make proto` (gera `internal/pb/**`). Antes do `make proto`, editar o `.proto`.

**5. Proto (`proto/wc2026/nationalteam/v1/national_team.proto`)**: adicionar `string flag_url = 3;` à mensagem `NationalTeam` (campo em inglês, ADR-0004).

**6. Repository (`national_team_repository.go`)**: adicionar `FlagURL string` ao struct de domínio `NationalTeam` e mapear `row.FlagURL` em `toNationalTeam`. A assinatura de `GetNationalTeamByID` (retorna apenas o nome) permanece inalterada.

**7. Service (`national_team_service.go`)**: adicionar `FlagURL string` ao struct `NationalTeam` do service e propagá-lo no mapeamento de `ListNationalTeams`.

**8. Handler (`national_team_handler.go`)**: preencher `FlagUrl: t.FlagURL` ao montar o `nationalteamv1.NationalTeam` na resposta. Handler permanece mapper puro.

### 5.1 Exemplo de Payload
N/A — sem payload parcial. Não há endpoint de escrita; a única alteração de contrato é o campo de saída `flag_url` em `ListNationalTeamsResponse`.

---

## 6. Guardrails de Execucao (LLM) - DEVE / NAO DEVE
> Quebrar qualquer item aqui **invalida a task**.

### 6.1 DEVE
- Obedecer ADR-0004: schema em pt-BR (`bandeira_url`), código/proto em inglês (`FlagURL`/`flag_url`); a tradução vive **exclusivamente** no `rename` do `sqlc.yaml` — sem `AS alias`, sem struct-espelho manual, sem conversão ad-hoc no repository.
- Obedecer ADR-0001: nenhuma dependência nova com CGO; migration usa SQL puro aplicável pelo driver `modernc.org/sqlite`.
- Obedecer ADR-0002: o campo flui pelas camadas existentes `repository → service → handler` sem novo wiring (a mudança é aditiva nos structs já providos).
- Manter as migrations como par `*.up.sql` / `*.down.sql` versionado e parametrização das queries (sem concatenação de SQL).
- Regenerar `internal/db/sqlc/**` e `internal/pb/**` via `make sqlc` / `make proto` e commitar o gerado.
- Alterar apenas arquivos listados na seção 8.
- Manter `GetNationalTeamByID` (repository/adapter) com a assinatura atual — não vaze `FlagURL` para quem não precisa.

### 6.2 NAO DEVE
- Não editar à mão arquivos gerados (`internal/db/sqlc/**`, `internal/pb/**`).
- Não fazer rebuild da tabela `selecoes` (risco com a FK de `usuario_selecoes`).
- Não criar RPC/endpoint de escrita, validação de URL, cache ou abstração nova "por precaução".
- Não tocar nos domínios `auth` ou `usuario_selecoes`.
- Não renomear/limpar dead code preexistente fora do escopo.

---

## 7. Passos Sugeridos (checklist executavel)
- [ ] Criar `000005_add_bandeira_url_to_selecoes.up.sql` (ADD COLUMN + 16 UPDATEs com URLs flagcdn).
- [ ] Criar `000005_add_bandeira_url_to_selecoes.down.sql` (DROP COLUMN).
- [ ] Adicionar `bandeira_url: "FlagURL"` ao `rename` do `sqlc.yaml`.
- [ ] Atualizar `selecoes.sql` (incluir `bandeira_url` nos SELECTs de List e GetByID).
- [ ] Rodar `make sqlc` e conferir `models.go`/`selecoes.sql.go` regenerados (`FlagURL`).
- [ ] Adicionar `string flag_url = 3;` ao `national_team.proto` e rodar `make proto`.
- [ ] Adicionar `FlagURL` ao struct e ao `toNationalTeam` do repository.
- [ ] Adicionar `FlagURL` ao struct e ao mapeamento do service.
- [ ] Preencher `FlagUrl` no mapeamento do handler.
- [ ] Atualizar/criar testes (seção 10) e rodar `make test`.
- [ ] `make build-all` verde (prova de portabilidade sem CGO).

---

## 8. Arquivos Envolvidos

### 8.0 Visão em Árvore

```
proto/wc2026/nationalteam/v1/
└── national_team.proto                              [M]
internal/
├── db/
│   ├── migrations/
│   │   ├── 000002_seed_selecoes.up.sql              [R]
│   │   ├── 000005_add_bandeira_url_to_selecoes.up.sql   [N]
│   │   └── 000005_add_bandeira_url_to_selecoes.down.sql [N]
│   ├── queries/
│   │   └── selecoes.sql                             [M]
│   ├── sqlc/
│   │   ├── models.go                                [M] (gerado — make sqlc)
│   │   └── selecoes.sql.go                          [M] (gerado — make sqlc)
│   └── migrations/000001_create_selecoes.up.sql     [R]
├── pb/wc2026/nationalteam/v1/
│   ├── national_team.pb.go                          [M] (gerado — make proto)
│   └── national_team_grpc.pb.go                     [R]
└── nationalteam/
    ├── module.go                                    [R]
    ├── repository/national_team_repository.go       [M]
    ├── service/national_team_service.go             [M]
    └── handler/national_team_handler.go             [M]
sqlc.yaml                                            [M]
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

---

### 8.1 Arquivos Existentes (leitura/referencia)
- `internal/db/migrations/000001_create_selecoes.up.sql` — schema atual da tabela `selecoes`.
- `internal/db/migrations/000002_seed_selecoes.up.sql` — IDs canônicos das 16 seleções para o backfill.
- `internal/db/migrations/000004_create_usuario_selecoes.up.sql` — confirmar a FK `selecao_id` (motivo de não fazer rebuild).
- `internal/db/queries/selecoes.sql` — queries a estender.
- `sqlc.yaml` — bloco `rename` (ponte de idioma ADR-0004).
- `proto/wc2026/nationalteam/v1/national_team.proto` — mensagem `NationalTeam` a estender.
- `internal/nationalteam/repository/national_team_repository.go` — struct de domínio e `toNationalTeam`.
- `internal/nationalteam/service/national_team_service.go` — struct e mapeamento.
- `internal/nationalteam/handler/national_team_handler.go` — mapeamento proto.
- `internal/nationalteam/module.go` — confirmar que nenhum wiring novo é necessário.

### 8.2 Arquivos a Criar
- `internal/db/migrations/000005_add_bandeira_url_to_selecoes.up.sql` — ADD COLUMN `bandeira_url TEXT NOT NULL DEFAULT ''` + 16 UPDATEs com URLs flagcdn.
- `internal/db/migrations/000005_add_bandeira_url_to_selecoes.down.sql` — `ALTER TABLE selecoes DROP COLUMN bandeira_url;`.

### 8.3 Arquivos a Modificar
- `internal/db/queries/selecoes.sql` — incluir `bandeira_url` nos SELECTs.
- `sqlc.yaml` — `bandeira_url: "FlagURL"` no `rename`.
- `proto/wc2026/nationalteam/v1/national_team.proto` — `string flag_url = 3;`.
- `internal/nationalteam/repository/national_team_repository.go` — `FlagURL` no struct + `toNationalTeam`.
- `internal/nationalteam/service/national_team_service.go` — `FlagURL` no struct + mapeamento.
- `internal/nationalteam/handler/national_team_handler.go` — `FlagUrl: t.FlagURL` no mapeamento.
- `internal/db/sqlc/models.go` e `internal/db/sqlc/selecoes.sql.go` — **gerados** via `make sqlc` (não editar à mão).
- `internal/pb/wc2026/nationalteam/v1/national_team.pb.go` — **gerado** via `make proto` (não editar à mão).

---

## 9. Aceite Tecnico (criterios objetivos)
A task estara concluída quando:
- [ ] AC1 — A coluna `bandeira_url TEXT NOT NULL` existe em `selecoes` após aplicar as migrations; `make build-all` verde (sem CGO).
- [ ] AC2 — As 16 seleções do seed têm `bandeira_url` preenchida com a URL `https://flagcdn.com/w320/{codigo}.png` correspondente (mapeamento da §5).
- [ ] AC3 — `make sqlc` regenera `models.go`/`selecoes.sql.go` com o campo `FlagURL` (sem mapeamento manual de idioma; apenas via `rename`).
- [ ] AC4 — A mensagem proto `NationalTeam` expõe `flag_url` (campo 3) e `make proto` regenera o stub.
- [ ] AC5 — `ListNationalTeams` (handler→service→repository) retorna a `FlagURL`/`flag_url` de cada seleção.
- [ ] AC6 — Assinatura de `GetNationalTeamByID` inalterada; domínios `auth` e `usuario_selecoes` intactos.
- [ ] AC7 — `make test` verde para tudo que foi tocado.
- [ ] AC8 — Guardrails da seção 6 respeitados (ADR-0001/0002/0004; nenhum arquivo gerado editado à mão).

---

## 10. Testes

> Gerado pelo agente `agent-spec-qa-test-generator` em 2026-05-29. 14 casos de teste (2 unitários, 8 integração, 3 e2e + companions). Gates aplicados: invariant_first, owning_layer, real_execution, failure_means_fix_production, no_snapshot_without_contract, no_self_set_mock_assertion, negative_companion.

### 10.1 Testes Existentes a Modificar
- `internal/nationalteam/repository/national_team_repository_test.go` — estender suíte de integração (DB real via `testutil.TestNewDB`) com asserts do `bandeira_url`/`FlagURL`; declarar constante `seedBrasilFlagURL = "https://flagcdn.com/w320/br.png"` ao lado de `seedBrasilID`/`seedBrasilName`.
- `internal/nationalteam/service/national_team_service_test.go` — estender `TestListNationalTeams_ReturnsList` para verificar propagação de `FlagURL` pelo service (mock de repository com `FlagURL` preenchida).
- `test/e2e/national_team_e2e_test.go` — estender `TestE2E_ListNationalTeams_Public` (bufconn) verificando `flag_url` para as 16 seleções + URL exata do Brasil.

### 10.2 Testes a Criar

**Unitários**
- `internal/nationalteam/handler/national_team_handler_test.go` (package `handler_test`) — **novo arquivo** (não há suíte unitária de handler para o domínio). Testa o mapeamento `service.FlagURL → proto flag_url` (CT-010/CT-011). Confirmar criação com o orquestrador no Gate 2.

**Integração** (DB real efêmero via `testutil.TestNewDB`)
- Adicionados a `national_team_repository_test.go`: validação de schema/migration (`PRAGMA table_info`), backfill das 16 seleções, propagação de `FlagURL` em `ListNationalTeams`, ponte de idioma sqlc (CT-001 a CT-007, CT-014).

**E2E** (bufconn via `testutil.TestNewBufconnServer`)
- Adicionados a `national_team_e2e_test.go`: `flag_url` em `ListNationalTeams` e guarda de regressão de `GetNationalTeamByID` (CT-011 a CT-013).

### 10.3 Cenários Obrigatórios
- [ ] CT-001 — Coluna `bandeira_url` existe com `NOT NULL` após migrations (PRAGMA table_info).
- [ ] CT-002 — Down migration remove `bandeira_url` (DROP COLUMN suportado pelo modernc).
- [ ] CT-003 — 16 seleções com URL `https://flagcdn.com/w320/{codigo}.png`; Brasil=`br`, Inglaterra=`gb-eng`, Coreia do Sul=`kr`.
- [ ] CT-004 — Nenhuma seleção com `bandeira_url = ''` após backfill (companion de CT-003).
- [ ] CT-005 — `repository.ListNationalTeams` retorna `FlagURL` preenchida para cada seleção.
- [ ] CT-006 — Companion negativo: nenhuma `FlagURL` vazia no repository.
- [ ] CT-007 — `GetNationalTeamByID` mantém assinatura `(string, error)` sem vazar `FlagURL`.
- [ ] CT-008 — `service.ListNationalTeams` propaga `FlagURL` do repository (mock).
- [ ] CT-009 — Companion negativo: service não enriquece `FlagURL` vazia.
- [ ] CT-010 — Handler mapeia `FlagURL` → proto `flag_url`.
- [ ] CT-011 — E2E `ListNationalTeams` retorna `flag_url` nas 16 seleções; Brasil com URL exata.
- [ ] CT-012 — Companion negativo E2E: nenhuma `flag_url` vazia.
- [ ] CT-013 — E2E `GetNationalTeamByID` inalterado (sem `flag_url`).
- [ ] CT-014 — Ponte de idioma sqlc: `FlagURL` no tipo gerado via `rename`, sem alias SQL/conversão manual.

### 10.4 Padrões de Teste
- **Framework**: Go stdlib `testing` + `github.com/stretchr/testify/require`.
- **Convenção de nomes**: `TestIntegration_*` (integração com DB real), `TestE2E_*` (e2e bufconn), `TestXxx_Yyy` (unitário com mock).
- **Fixture/Setup**: SQLite efêmero via `internal/testutil.TestNewDB` (modernc, `CGO_ENABLED=0`); servidor gRPC in-process via `internal/testutil.TestNewBufconnServer`; constantes de seed (`seedBrasilID`/`seedBrasilName` → adicionar `seedBrasilFlagURL`).
- **Mocks**: mocks manuais por interface declarada no consumidor (ex.: `nationalTeamRepoMock` com `listFn`). Mock Budget respeitado — todo CT com mock tem companion de integração real (CT-005/CT-011).

### 10.5 Cenários de Erro

| Cenário                                   | Trigger                                         | Expected                                             | Código/Status    |
| ----------------------------------------- | ----------------------------------------------- | ---------------------------------------------------- | ---------------- |
| Down migration não remove coluna          | `ALTER TABLE selecoes DROP COLUMN bandeira_url` | `pragma_table_info` retorna count 0                  | — (SQL/DDL)      |
| Backfill incompleto                       | Seleção com `bandeira_url = ''` após migrations | `SELECT count(*) WHERE bandeira_url=''` == 0         | — (integridade)  |
| `FlagURL` vazia no repository             | `ListNationalTeams` retorna team sem URL        | `require.NotEmpty(team.FlagURL)`                     | — (regressão)    |
| Service enriquece `FlagURL` indevidamente | mock retorna `FlagURL=''`                       | service repassa `''` (mapper puro)                   | — (regressão)    |
| `flag_url` vazia no E2E                   | qualquer team da lista sem URL                  | `require.NotEmpty(team.GetFlagUrl())` com ID no erro | gRPC OK + assert |
| Vazamento em `GetNationalTeamByID`        | response com `flag_url`                         | tipo de retorno sem `flag_url` (type system)         | compilação       |

### 10.6 Rastreabilidade: Aceite Técnico -> Testes

| #   | Critério de Aceite (seção 9)                                     | Teste(s) Correspondente(s)                     | Tipo                        |
| --- | ---------------------------------------------------------------- | ---------------------------------------------- | --------------------------- |
| AC1 | Coluna `bandeira_url NOT NULL` + build sem CGO                   | CT-001, CT-002                                 | Integração                  |
| AC2 | 16 seleções com URL flagcdn correta                              | CT-003, CT-004, CT-006, CT-012                 | Integração / E2E            |
| AC3 | `make sqlc` gera `FlagURL` (sem mapeamento manual)               | CT-005, CT-006, CT-014                         | Integração                  |
| AC4 | Proto `NationalTeam.flag_url` (campo 3)                          | CT-010, CT-011                                 | Unitário / E2E              |
| AC5 | `ListNationalTeams` retorna `FlagURL`/`flag_url` ponta a ponta   | CT-005, CT-008, CT-009, CT-010, CT-011, CT-012 | Unitário / Integração / E2E |
| AC6 | `GetNationalTeamByID` inalterado; auth/usuario_selecoes intactos | CT-007, CT-013                                 | Integração / E2E            |
| AC7 | `make test` verde                                                | CT-011 (cadeia completa)                       | E2E                         |
| AC8 | Guardrails ADR-0001/0002/0004; gerados não editados à mão        | CT-001, CT-014                                 | Integração                  |

---

## 11. Notas / Observacoes
- **Trade-off `DEFAULT ''`**: a coluna é criada com `NOT NULL DEFAULT ''` para permitir o ADD COLUMN numa tabela já populada sem rebuild (que seria arriscado pela FK `usuario_selecoes.selecao_id`). O backfill imediato preenche as 16 linhas reais. Não há query de `INSERT` de seleção no código (apenas seed + leitura), então nenhum caminho de gravação depende do default vazio. Se futuramente surgir um RPC de criação de seleção, reavaliar a remoção do default.
- **model/risk**: classificados como `opus`/`high` pela regra de critical path (`db_migrations` + `api_contracts`). Se o time considerar a migration aditiva suficientemente trivial, pode rebaixar para `sonnet`/`medium` editando o frontmatter antes da execução.
- **flagcdn `gb-eng` e `kr`**: Inglaterra usa o código regional `gb-eng` (suportado pelo flagcdn) e Coreia do Sul usa `kr`.
- **Código gerado**: `internal/db/sqlc/**` e `internal/pb/**` são regenerados por `make sqlc`/`make proto` e devem ser commitados junto, nunca editados à mão.
