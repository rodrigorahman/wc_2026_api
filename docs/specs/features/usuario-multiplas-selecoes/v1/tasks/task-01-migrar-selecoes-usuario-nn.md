# TASKCARD - Execucao Rapida (com Guardrails LLM)

## 1. Identificacao
- **ID**: TC-001
- **Nome da Task**: Migrar cadastro de usuário de 1 seleção (1:1) para múltiplas seleções (N:N)
- **model**: opus                <!-- migration de schema + domínio auth + cross-module (≥3 pacotes) -->
- **risk**: high                 <!-- db_migrations + auth -->
- **gates**: [qa, tech_review]   <!-- tipo=db_migrations/auth -->
- **Variante**: backend
- **Responsavel**: agente executor (go-backend-implementer)
- **Data**: 2026-05-29
- **Status**: A Fazer
- **Dependencias**: (nenhuma)
- **Relacionados**: ADR-0001, ADR-0002, ADR-0004; feature `usuario-multiplas-selecoes/v1`
- **source**: no_discovery   <!-- não havia pre-refinement.md para esta feature -->

---

## 2. Contexto
O cadastro de usuário hoje vincula **exatamente uma** seleção favorita ao usuário (modelo 1:1): coluna `selecao_id TEXT NOT NULL` em `usuarios`, campo singular `national_team_id` no `RegisterRequest`/`GetMeResponse` e validação singular em `AuthService.Register`. O produto passou a exigir que o usuário escolha **mais de uma** seleção no cadastro. Esta task migra o modelo para **N:N** (mínimo 1, máximo 3 seleções por usuário), atravessando schema, contrato gRPC e as três camadas do domínio auth.

---

## 3. Objetivo da Task
Permitir que o cadastro (`Register`) receba uma lista de seleções (1 a 3) e que o `GetMe` retorne essa lista. Ao final: tabela de junção `usuario_selecoes` substituindo a coluna `selecao_id` de `usuarios`; contrato `repeated national_team_ids` validado; persistência atômica das seleções por usuário; e a suíte (`make test`) verde para o domínio auth e e2e.

---

## 4. Escopo
### 4.1 Inclui
- [ ] Migration `000004` que cria `usuario_selecoes` (PK composta `usuario_id` + `selecao_id`, FKs) e **remove** a coluna `selecao_id` de `usuarios`.
- [ ] Atualização do `sqlc.yaml` (renames pt-BR→inglês da nova tabela/coluna) e das queries em `usuarios.sql`; regeneração via `make sqlc`.
- [ ] `RegisterRequest.national_team_id` (singular) → `repeated national_team_ids` com validação `min_items: 1, max_items: 3`, cada item UUID; idem `GetMeResponse`; regeneração via `make proto`.
- [ ] `AuthService.Register`: validar **existência de cada** seleção e **rejeitar duplicatas** na lista; persistir o usuário + N seleções de forma **atômica**.
- [ ] `AuthService.GetUser`/`GetMe`: carregar e retornar a lista de seleções do usuário.
- [ ] Ajuste do `userRepositoryAdapter` (em `internal/auth/module.go`) e do `UserRepository` para o novo shape de lista.

### 4.2 Fora do escopo
- [ ] Endpoint para **editar** as seleções de um usuário já cadastrado (só cadastro + leitura via GetMe).
- [ ] Migração de dados de usuários preexistentes (greenfield — banco sem usuários em produção; a migration apenas dropa a coluna).
- [ ] Versionamento do pacote proto para `v2` (a mudança é in-place em `v1`, contrato ainda não publicado).
- [ ] Qualquer alteração no fluxo de `Login`, JWT, interceptors ou na tabela `selecoes` (read-only, permanece intacta).

---

## 5. Descricao de Execucao (COMO fazer)

**Ordem sugerida: schema → sqlc → proto → repository → service → handler → wiring.**

### Schema (migration 000004)
Criar par `up`/`down` seguindo o padrão de `000003`:
- **up**: `CREATE TABLE usuario_selecoes (usuario_id TEXT NOT NULL REFERENCES usuarios(id), selecao_id TEXT NOT NULL REFERENCES selecoes(id), PRIMARY KEY (usuario_id, selecao_id));` + índice em `selecao_id` (`idx_usuario_selecoes_selecao_id`). Em seguida remover o índice `idx_usuarios_selecao_id` e a coluna: `DROP INDEX idx_usuarios_selecao_id;` + `ALTER TABLE usuarios DROP COLUMN selecao_id;` (suportado pelo SQLite ≥3.35 do `modernc.org/sqlite`).
- **down**: reverso — recriar a coluna `selecao_id`/índice em `usuarios` e `DROP TABLE usuario_selecoes`.
- Tabela e colunas em **pt-BR** (ADR-0004).

### sqlc (`sqlc.yaml` + `usuarios.sql`)
- Em `sqlc.yaml`, adicionar renames para a nova tabela/coluna mantendo a ponte de idioma: `usuario_selecoes: "UserNationalTeam"` e `usuario_id: "UserID"` (a coluna `selecao_id` já mapeia para `NationalTeamID`).
- Em `usuarios.sql`: remover `selecao_id` do `INSERT`/`RETURNING` de `CreateUser` e dos `SELECT` de `GetUserByEmail`/`GetUserByID`. Adicionar queries para a junção: `AddUserNationalTeam` (`INSERT INTO usuario_selecoes ...`) e `ListUserNationalTeams` (`SELECT selecao_id FROM usuario_selecoes WHERE usuario_id = ?`).
- Rodar `make sqlc` (não editar `internal/db/sqlc/**` à mão).

### proto (`auth.proto`)
- `RegisterRequest`: trocar `string national_team_id = 4` por `repeated string national_team_ids = 4` com `(buf.validate.field).repeated = { min_items: 1, max_items: 3, items: { string: { uuid: true } } }`.
- `GetMeResponse`: trocar `string national_team_id = 4` por `repeated string national_team_ids = 4`.
- Rodar `make proto` (não editar `internal/pb/**` à mão).

### repository (`user_repository.go`)
- `User.NationalTeamID string` → `NationalTeamIDs []string`.
- `CreateUser`: passar a abrir uma **transação** (`db.BeginTx` + `q.WithTx(tx)`): inserir a linha de `usuarios` e, no mesmo `tx`, uma linha em `usuario_selecoes` por seleção; `Commit`/`Rollback`. Isso exige injetar `*sql.DB` em `NewUserRepository` (hoje recebe apenas `*sqlc.Queries`); o `*sql.DB` já é provido pelo `internal/db` module.
- `GetUserByID`/`GetUserByEmail`: após carregar a linha do usuário, popular `NationalTeamIDs` via `ListUserNationalTeams`.
- Ajustar `toUser` para o novo shape.

### service (`auth_service.go`)
- `RegisterParams.NationalTeamID` → `NationalTeamIDs []string`; `User.NationalTeamID` → `NationalTeamIDs []string`.
- `Register`: para cada id na lista, validar existência via `nationalTeams.GetNationalTeamByID` (id inexistente → `codes.InvalidArgument`, `msgInvalidNationalTeam`); **rejeitar duplicatas** na lista (→ `codes.InvalidArgument`). A cardinalidade 1..3 é garantida no contrato (protovalidate) — **não** reimplementar a checagem de tamanho no service.
- Montar `User{..., NationalTeamIDs: ...}` e delegar ao `users.CreateUser` (atomicidade vive no repository).

### handler (`auth_handler.go`)
- `Register`: `NationalTeamIDs: req.GetNationalTeamIds()`.
- `GetMe`: `NationalTeamIds: user.NationalTeamIDs`.

### wiring (`internal/auth/module.go`)
- `userRepositoryAdapter` (3 métodos): trocar `NationalTeamID` por `NationalTeamIDs` no mapeamento `service.User`↔`repository.User`. As assert(`_ service.UserRepository = ...`) devem continuar compilando.

### 5.1 Exemplo de Payload
N/A — sem payload parcial (não há `PUT`/`PATCH`; `Register` continua com todos os campos obrigatórios, agora a lista de seleções com 1..3 itens).

---

## 6. Guardrails de Execucao (LLM) - DEVE / NAO DEVE
> Quebrar qualquer item aqui **invalida a task**.

### 6.1 DEVE
- **Obedecer ADR-0001**: driver SQLite é `modernc.org/sqlite` (pure-Go); migration e queries devem rodar com `CGO_ENABLED=0`. Não introduzir SQL específico de outro driver.
- **Obedecer ADR-0004**: schema (tabela `usuario_selecoes`, colunas `usuario_id`/`selecao_id`) em **pt-BR**; código Go e proto em **inglês**; a ponte é o `rename` do `sqlc.yaml`, **nunca** mapeamento manual.
- **Obedecer ADR-0002**: manter o wiring no `fx.Module` do domínio; a interface continua declarada no consumidor (`service.UserRepository`), o bind concreto→interface fica no `module.go`.
- Persistir usuário + seleções de forma **atômica** (transação): falha ao inserir qualquer seleção desfaz o usuário.
- Regenerar código via `make proto` e `make sqlc`; tratar `internal/pb/**` e `internal/db/sqlc/**` como gerados.
- Senha e token **nunca** logados (ADR-0003 permanece válido para o domínio).
- Alterar apenas arquivos listados na seção 8.
- `make test` verde para o que foi tocado antes de reportar conclusão.

### 6.2 NAO DEVE
- Não editar manualmente `internal/pb/**` nem `internal/db/sqlc/**`.
- Não criar endpoint de edição de seleções, nem mexer em `Login`/JWT/interceptors/tabela `selecoes`.
- Não reimplementar a validação de cardinalidade (1..3) no service — ela vive no contrato (protovalidate). Evitar validação defensiva duplicada.
- Não criar abstrações genéricas (camada de "associações" genérica, repositório de junção separado) — manter no `UserRepository` existente.
- Não versionar o pacote proto para `v2`; a mudança é in-place em `v1`.
- Não migrar dados preexistentes nem adicionar lógica de backfill (greenfield).

---

## 7. Passos Sugeridos (checklist executavel)
- [ ] Criar migration `000004` up/down (`usuario_selecoes` + drop coluna `selecao_id`).
- [ ] Atualizar `sqlc.yaml` (renames) e `usuarios.sql` (ajustar CreateUser/Gets + `AddUserNationalTeam` + `ListUserNationalTeams`); rodar `make sqlc`.
- [ ] Atualizar `auth.proto` (`repeated national_team_ids` + validação 1..3 em Register, lista em GetMe); rodar `make proto`.
- [ ] Atualizar `user_repository.go`: shape de lista + `CreateUser` transacional + leitura das seleções; injetar `*sql.DB` em `NewUserRepository`.
- [ ] Atualizar `auth_service.go`: lista, validação de existência por id, rejeição de duplicatas, novo shape do `User`.
- [ ] Atualizar `auth_handler.go`: mapear `repeated` em Register e GetMe.
- [ ] Atualizar `userRepositoryAdapter` em `module.go` para o shape de lista.
- [ ] Rodar `make test`; corrigir testes existentes impactados (ver seção 10).

---

## 8. Arquivos Envolvidos

### 8.0 Visão em Árvore

```
proto/wc2026/auth/v1/
└── auth.proto                                      [M]
internal/
├── db/
│   ├── migrations/
│   │   ├── 000004_create_usuario_selecoes.up.sql   [N]
│   │   ├── 000004_create_usuario_selecoes.down.sql [N]
│   │   ├── 000001_create_selecoes.up.sql           [R]
│   │   └── 000003_create_usuarios.up.sql           [R]
│   ├── queries/
│   │   └── usuarios.sql                            [M]
│   ├── sqlc/
│   │   ├── db.go (WithTx)                          [R]
│   │   ├── usuarios.sql.go                         [R] (regenerado por make sqlc)
│   │   └── models.go                               [R] (regenerado por make sqlc)
│   └── module.go (provê *sql.DB / *sqlc.Queries)   [R]
└── auth/
    ├── service/auth_service.go                     [M]
    ├── repository/user_repository.go               [M]
    ├── handler/auth_handler.go                     [M]
    └── module.go (userRepositoryAdapter / wiring)  [M]
sqlc.yaml                                           [M]
internal/pb/wc2026/auth/v1/**                       [R] (regenerado por make proto)
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

---

### 8.1 Arquivos Existentes (leitura/referencia)
- `internal/db/migrations/000003_create_usuarios.up.sql` / `.down.sql` — padrão de migration (índice + tabela) a seguir no `000004`.
- `internal/db/migrations/000001_create_selecoes.up.sql` — alvo da FK `selecao_id`.
- `internal/db/sqlc/db.go` — confirma `Queries.WithTx(tx *sql.Tx)` disponível para a transação.
- `internal/db/module.go` — mostra que `*sql.DB` e `*sqlc.Queries` já são providos por fx (a injeção do `*sql.DB` no repository resolve automaticamente).
- `internal/auth/module.go` — `userRepositoryAdapter` e binds concreto→interface; precisa acompanhar o novo shape.
- `internal/auth/service/auth_service.go`, `internal/auth/repository/user_repository.go`, `internal/auth/handler/auth_handler.go` — estado atual 1:1 a migrar.
- `sqlc.yaml` — ponte de idioma (renames) a estender.
- `CLAUDE.md`, ADR-0001/0002/0003/0004 — regras de stack invioláveis.
- Testes existentes a observar: `test/e2e/auth_e2e_test.go`, `internal/auth/service/auth_service_test.go`, `internal/auth/repository/user_repository_test.go`, `internal/auth/handler/auth_handler_test.go`, `internal/auth/wiring_test.go`.

### 8.2 Arquivos a Criar
- `internal/db/migrations/000004_create_usuario_selecoes.up.sql` — cria `usuario_selecoes` (PK composta + FKs + índice) e remove `selecao_id`/índice de `usuarios`.
- `internal/db/migrations/000004_create_usuario_selecoes.down.sql` — reverso: recria coluna/índice em `usuarios` e dropa `usuario_selecoes`.

### 8.3 Arquivos a Modificar
- `proto/wc2026/auth/v1/auth.proto` — `repeated national_team_ids` (validação 1..3, UUID) em `RegisterRequest` e `GetMeResponse`.
- `sqlc.yaml` — renames da nova tabela/coluna (`usuario_selecoes`→`UserNationalTeam`, `usuario_id`→`UserID`).
- `internal/db/queries/usuarios.sql` — ajustar `CreateUser`/`GetUserByEmail`/`GetUserByID` (sem `selecao_id`) + `AddUserNationalTeam` + `ListUserNationalTeams`.
- `internal/auth/service/auth_service.go` — `NationalTeamIDs []string`, validação de existência + duplicatas.
- `internal/auth/repository/user_repository.go` — shape de lista, `CreateUser` transacional, leitura das seleções, `*sql.DB` no construtor.
- `internal/auth/handler/auth_handler.go` — mapear `repeated` em Register/GetMe.
- `internal/auth/module.go` — `userRepositoryAdapter` para o novo shape.

---

## 9. Aceite Tecnico (criterios objetivos)
A task estará concluída quando:
- [ ] **AC1** — A migration `000004` aplica (up) e reverte (down) sem erro; após `up`, `usuarios` não tem mais `selecao_id` e existe `usuario_selecoes` com PK composta e FKs para `usuarios`/`selecoes`.
- [ ] **AC2** — `RegisterRequest` e `GetMeResponse` expõem `repeated national_team_ids`; protovalidate rejeita lista vazia e lista com >3 itens (`codes.InvalidArgument`), e aceita 1..3 UUIDs válidos.
- [ ] **AC3** — `Register` com 1..3 seleções válidas persiste o usuário e **todas** as seleções atomicamente e retorna `user_id` não vazio.
- [ ] **AC4** — `Register` com qualquer seleção inexistente retorna `codes.InvalidArgument` (`seleção favorita inválida`) e **não** cria o usuário (rollback).
- [ ] **AC5** — `Register` com ids duplicados na lista retorna `codes.InvalidArgument`.
- [ ] **AC6** — `GetMe` (autenticado) retorna exatamente as seleções escolhidas no cadastro.
- [ ] **AC7** — Guardrails da seção 6 respeitados (idioma pt-BR/EN via sqlc rename, sem CGO, código gerado não editado à mão).
- [ ] **AC8** — `make test` verde para o domínio auth, repository e e2e; nenhum fluxo existente (Login, interceptors) quebrado.

---

## 10. Testes

> Gerado pelo agente `agent-spec-qa-test-generator` em 2026-05-29. 28 casos de teste (unit: 8, integração: 10, e2e: 8, segurança: 2).

### 10.1 Testes Existentes a Modificar
Suítes que já existem e precisam ser atualizadas para o novo shape de lista (`NationalTeamIDs` / `national_team_ids`):
- `internal/auth/repository/user_repository_test.go` — atualizar para lista de seleções; casos: CT-048, CT-052, CT-053.
- `internal/auth/service/auth_service_test.go` — atualizar para lista de seleções; casos: CT-036, CT-039, CT-041, CT-046.
- `internal/auth/wiring_test.go` — atualizar para lista de seleções; casos: CT-054.
- `test/e2e/auth_e2e_test.go` — atualizar para lista de seleções; casos: CT-056, CT-057.

### 10.2 Testes a Criar
Novos casos a cobrir, organizados por camada:

**Unitários**
- `CT-036` (internal/auth/service/auth_service_test.go) — Register com 1 seleção válida persiste usuário e seleção e retorna user_id não vazio
- `CT-037` (internal/auth/service/auth_service_test.go) — Register com 3 seleções válidas distintas persiste todas e retorna user_id
- `CT-038` (internal/auth/service/auth_service_test.go) — Register com 2 seleções válidas persiste ambas atomicamente (table-driven: 1, 2 e 3 seleções)
- `CT-039` (internal/auth/service/auth_service_test.go) — Register com seleção única — GetNationalTeamByID chamado com o ID correto
- `CT-040` (internal/auth/service/auth_service_test.go) — Register com 3 seleções — GetNationalTeamByID chamado 3 vezes, uma por ID
- `CT-041` (internal/auth/service/auth_service_test.go) — Register com seleção inexistente retorna InvalidArgument e não persiste
- `CT-042` (internal/auth/service/auth_service_test.go) — Register com lista de seleções onde a 2ª é inexistente retorna InvalidArgument e não persiste (table-driven: posições 1, 2, 3)
- `CT-043` (internal/auth/service/auth_service_test.go) — Register com IDs duplicados na lista retorna InvalidArgument e não persiste
- `CT-044` (internal/auth/service/auth_service_test.go) — Register com e-mail duplicado retorna AlreadyExists com lista de seleções (migração de CT-004)
- `CT-045` (internal/auth/service/auth_service_test.go) — GetUser retorna NationalTeamIDs com a lista completa de seleções do usuário
- `CT-046` (internal/auth/service/auth_service_test.go) — GetUser com ID desconhecido retorna codes.NotFound

**Integração**
- `CT-047` (internal/auth/repository/user_repository_test.go) — Repository.CreateUser persiste usuário e retorna lista de seleções via ListUserNationalTeams (integração real com SQLite)
- `CT-048` (internal/auth/repository/user_repository_test.go) — Repository.CreateUser com 3 seleções válidas persiste as 3 na tabela usuario_selecoes
- `CT-049` (internal/auth/repository/user_repository_test.go) — Repository.GetUserByID retorna lista NationalTeamIDs corretamente populada
- `CT-050` (internal/auth/repository/user_repository_test.go) — Repository.CreateUser com seleção inexistente faz rollback: usuário e seleções não persistidos
- `CT-051` (internal/auth/repository/user_repository_test.go) — Repository.CreateUser com 3 seleções onde a 3ª tem FK inválida faz rollback de todas
- `CT-052` (internal/auth/repository/user_repository_test.go) — Migration 000004 up: tabela usuario_selecoes existe com PK composta e selecao_id não existe em usuarios
- `CT-053` (internal/auth/repository/user_repository_test.go) — Migration 000004 down reverte para schema 1:1: selecao_id retorna em usuarios, usuario_selecoes removida
- `CT-054` (internal/auth/wiring_test.go) — Wiring: Register com lista de seleções funciona pelo grafo fx real (migração de CT-022)
- `CT-055` (internal/auth/wiring_test.go) — Wiring: Register com seleção inexistente retorna InvalidArgument pelo grafo fx real (migração de TestIntegration_Register_UnknownNationalTeam)
- `CT-064` (internal/auth/wiring_test.go) — Wiring smoke: grafo fx compila com novo shape de UserRepository (NationalTeamIDs lista) — migração de CT-035

**E2E**
- `CT-056` (test/e2e/auth_e2e_test.go) — E2E: Register com lista de 1 seleção válida retorna user_id não vazio (migração de CT-027)
- `CT-057` (test/e2e/auth_e2e_test.go) — E2E: Register → Login → GetMe retorna lista national_team_ids com a seleção escolhida (migração de CT-029)
- `CT-058` (test/e2e/auth_e2e_test.go) — E2E: Register com 3 seleções válidas distintas — GetMe retorna as 3 seleções
- `CT-059` (test/e2e/auth_e2e_test.go) — E2E: Register com national_team_ids vazio é rejeitado por protovalidate com InvalidArgument
- `CT-060` (test/e2e/auth_e2e_test.go) — E2E: Register com 4 seleções é rejeitado por protovalidate com InvalidArgument
- `CT-061` (test/e2e/auth_e2e_test.go) — E2E: Register com seleção inexistente retorna InvalidArgument e não cria usuário (migração de CT-028 contextual)
- `CT-062` (test/e2e/auth_e2e_test.go) — E2E: Register com national_team_ids contendo UUID sintático inválido é rejeitado por protovalidate
- `CT-063` (test/e2e/auth_e2e_test.go) — E2E: CT-031 (token expirado) continua funcionando após a migração N:N

### 10.3 Cenários Obrigatórios
- [ ] CT-036 — Register com 1 seleção válida persiste usuário e seleção e retorna user_id não vazio
- [ ] CT-037 — Register com 3 seleções válidas distintas persiste todas e retorna user_id
- [ ] CT-038 — Register com 2 seleções válidas persiste ambas atomicamente (table-driven: 1, 2 e 3 seleções)
- [ ] CT-039 — Register com seleção única — GetNationalTeamByID chamado com o ID correto
- [ ] CT-040 — Register com 3 seleções — GetNationalTeamByID chamado 3 vezes, uma por ID
- [ ] CT-041 — Register com seleção inexistente retorna InvalidArgument e não persiste
- [ ] CT-042 — Register com lista de seleções onde a 2ª é inexistente retorna InvalidArgument e não persiste (table-driven: posições 1, 2, 3)
- [ ] CT-043 — Register com IDs duplicados na lista retorna InvalidArgument e não persiste
- [ ] CT-044 — Register com e-mail duplicado retorna AlreadyExists com lista de seleções (migração de CT-004)
- [ ] CT-045 — GetUser retorna NationalTeamIDs com a lista completa de seleções do usuário
- [ ] CT-046 — GetUser com ID desconhecido retorna codes.NotFound
- [ ] CT-047 — Repository.CreateUser persiste usuário e retorna lista de seleções via ListUserNationalTeams (integração real com SQLite)
- [ ] CT-048 — Repository.CreateUser com 3 seleções válidas persiste as 3 na tabela usuario_selecoes
- [ ] CT-049 — Repository.GetUserByID retorna lista NationalTeamIDs corretamente populada
- [ ] CT-050 — Repository.CreateUser com seleção inexistente faz rollback: usuário e seleções não persistidos
- [ ] CT-051 — Repository.CreateUser com 3 seleções onde a 3ª tem FK inválida faz rollback de todas
- [ ] CT-052 — Migration 000004 up: tabela usuario_selecoes existe com PK composta e selecao_id não existe em usuarios
- [ ] CT-053 — Migration 000004 down reverte para schema 1:1: selecao_id retorna em usuarios, usuario_selecoes removida
- [ ] CT-054 — Wiring: Register com lista de seleções funciona pelo grafo fx real (migração de CT-022)
- [ ] CT-055 — Wiring: Register com seleção inexistente retorna InvalidArgument pelo grafo fx real (migração de TestIntegration_Register_UnknownNationalTeam)
- [ ] CT-056 — E2E: Register com lista de 1 seleção válida retorna user_id não vazio (migração de CT-027)
- [ ] CT-057 — E2E: Register → Login → GetMe retorna lista national_team_ids com a seleção escolhida (migração de CT-029)
- [ ] CT-058 — E2E: Register com 3 seleções válidas distintas — GetMe retorna as 3 seleções
- [ ] CT-059 — E2E: Register com national_team_ids vazio é rejeitado por protovalidate com InvalidArgument
- [ ] CT-060 — E2E: Register com 4 seleções é rejeitado por protovalidate com InvalidArgument
- [ ] CT-061 — E2E: Register com seleção inexistente retorna InvalidArgument e não cria usuário (migração de CT-028 contextual)
- [ ] CT-062 — E2E: Register com national_team_ids contendo UUID sintático inválido é rejeitado por protovalidate
- [ ] CT-063 — E2E: CT-031 (token expirado) continua funcionando após a migração N:N
- [ ] CT-064 — Wiring smoke: grafo fx compila com novo shape de UserRepository (NationalTeamIDs lista) — migração de CT-035

### 10.4 Padrões de Teste
- **Framework**: go test + testify/require
- **Convenção de nomes**: Test<Camada>_<Comportamento>_<CenarioOuResultado> — ex.: TestRegister_Success, TestIntegration_Login_WrongPassword, TestE2E_Register_Valid
- **Fixture/Setup**: helpers locais (baseUser(), teamFound(), hashOf(), newRepo(), startWiredApp()); constantes de seed via migration 000002 (validNationalTeamID / seededBrasilID)
- **Mocks**: structs manuais que implementam a interface (userRepoMock, nationalTeamRepoMock, tokenManagerMock) — sem biblioteca de mocking de terceiros
- **Boundary real**: SQLite efêmero via testutil.TestNewDB (migrations + seed + foreign_keys=ON) nos testes de repository e handler-integration; bufconn in-process via testutil.TestNewBufconnServer nos E2E
- **Clock**: fixedClock local (implementa clock.Clock) injetado; testutil.NewFixedClock(*fixedClock) com método Advance nos E2E

### 10.5 Cenários de Erro
| Cenário | Trigger | Expected | Código/Status |
|---------|---------|----------|---------------|
| CT-041 Register com seleção inexistente retorna InvalidArgumen | NationalTeamIDs com 1 ID que não existe | codes.InvalidArgument | codes.InvalidArgument |
| CT-042 Register com lista de seleções onde a 2ª é inexistente  | Table-driven: 3 subcasos variando qual posição da lista é inválida | Todos os 3 subcasos retornam InvalidArgument sem persistir. | codes.InvalidArgument |
| CT-043 Register com IDs duplicados na lista retorna InvalidArg | Table-driven: 2 casos — [id, id] e [id1, id2, id1] | codes.InvalidArgument | codes.InvalidArgument |
| CT-044 Register com e-mail duplicado retorna AlreadyExists com | RegisterParams com e-mail já registrado e NationalTeamIDs válida | codes.AlreadyExists | codes.AlreadyExists |
| CT-046 GetUser com ID desconhecido retorna codes.NotFound | userID não presente no repositório | codes.NotFound | codes.NotFound |
| CT-050 Repository.CreateUser com seleção inexistente faz rollb | User com NationalTeamIDs contendo ID inexistente | CreateUser retorna erro FK | codes.NotFound |
| CT-055 Wiring: Register com seleção inexistente retorna Invali | NationalTeamIDs = [UUID inexistente] | codes.InvalidArgument | codes.InvalidArgument |
| CT-059 E2E: Register com national_team_ids vazio é rejeitado p | RegisterRequest sem national_team_ids (campo omitido ou lista vazia) | codes.InvalidArgument retornado pelo interceptor protovalidate. | codes.InvalidArgument |
| CT-061 E2E: Register com seleção inexistente retorna InvalidAr | RegisterRequest com national_team_ids = [UUID inexistente] | codes.InvalidArgument | codes.InvalidArgument |

### 10.6 Rastreabilidade: Aceite Técnico → Testes
| Critério (seção 9) | Teste(s) | Tipo |
|---|---|---|
| AC1 | CT-052, CT-053 | INTEGRACAO |
| AC2 | CT-059, CT-060, CT-062 | E2E |
| AC3 | CT-036, CT-037, CT-038, CT-039, CT-040, CT-047, CT-048, CT-054, CT-056, CT-058 | E2E, INTEGRACAO, UNITARIO |
| AC4 | CT-040, CT-041, CT-042, CT-050, CT-051, CT-055, CT-061 | E2E, INTEGRACAO, UNITARIO |
| AC5 | CT-043 | UNITARIO |
| AC6 | CT-045, CT-047, CT-049, CT-054, CT-057, CT-058 | E2E, INTEGRACAO, UNITARIO |
| AC7 | CT-054, CT-064 | INTEGRACAO |
| AC8 | CT-044, CT-046, CT-056, CT-057, CT-063, CT-064 | E2E, INTEGRACAO, UNITARIO |

### 10.7 Gates de Teste Aplicados
- invariant_first
- owning_layer
- real_execution
- failure_means_fix_production
- no_snapshot_without_contract
- no_self_set_mock_assertion
- negative_companion

---

## 11. Notas / Observacoes
- **Divisão de validação (decisão fechada)**: cardinalidade (≥1, ≤3) é responsabilidade do **contrato** (protovalidate no proto, aplicado pelo interceptor); **existência de cada seleção** e **rejeição de duplicatas** são responsabilidade do **service**. Evita validação defensiva duplicada (YAGNI) e mantém o erro de tamanho no boundary de entrada.
- **Atomicidade**: a inserção de usuário + seleções passa a exigir transação no repository — daí a injeção de `*sql.DB` em `NewUserRepository`. É a única mudança de assinatura de construtor; fx resolve o `*sql.DB` automaticamente (já provido pelo `internal/db` module).
- **Greenfield**: a remoção da coluna `selecao_id` não migra dados (sem usuários em produção). Se isso mudar antes de rodar, revisar a opção "migrar dados existentes".
- **Trade-off para o Tech Review**: manter a checagem de cardinalidade apenas no proto significa que um chamador direto do service (fora do gRPC) não teria o limite 1..3 aplicado. Decisão deliberada: o único ponto de entrada é o gRPC com protovalidate; não há caso de uso de chamada direta ao service em produção.
