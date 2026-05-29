# TECH_SPEC -- Especificação Técnica (Backend)

## 1. Identificação
- **Feature/Projeto**: Arquitetura Base — API do Álbum da Copa do Mundo
- **Variante**: backend
- **Stack**: Go (1.26.1) — gRPC, SQLite (modernc.org/sqlite), uber-fx, sqlc, golang-migrate
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-28
- **Versão**: v1
- **Status**: Draft
- **PRD Relacionado**: `docs/prds/features/arquitetura-base/v1/prd.md`

---

## 2. Resumo Técnico da Solução

Entregar a fundação de uma API **gRPC** em Go seguindo go-standard-layout (`cmd/`, `internal/`; `pkg/` reservado para quando houver código realmente importável por terceiros — não usado na v1) com wiring por **uber-fx** (`fx.Module` por domínio) e lifecycle gerenciado. A persistência usa **SQLite** via driver pure-Go `modernc.org/sqlite` (sem CGO, portável nos três SOs), com queries tipadas geradas por **sqlc** e schema versionado por **golang-migrate**. O contrato é definido em `.proto` versionado (`wc2026.<dominio>.v1`), com validação declarativa via **protovalidate** aplicada por interceptor. O módulo de **autenticação** expõe `Register` e `Login`: senha protegida com **bcrypt** (cost 12) e sessão via **JWT HS256** (TTL 1h, segredo em env/viper) validado por interceptor `go-grpc-middleware/auth` nos RPCs protegidos. O módulo **Seleção Favorita** (read-only, populado por seed) serve de referência arquitetural para os módulos futuros (figurinhas/trocas).

---

## 3. Arquitetura da Solução

### 3.1 Visão Geral

Servidor gRPC único, montado e gerenciado por uber-fx. As requisições atravessam uma cadeia de interceptors (recovery → logging zap → protovalidate → auth JWT) antes de chegar ao handler. Cada domínio é um `fx.Module` que provê seu handler, service e repository.

```
gRPC Client (frontend, fora de escopo)
        │  (HTTP/2 + protobuf)
        ▼
┌─────────────────────────────────────────────┐
│ gRPC Server (cmd/server, fx lifecycle)        │
│  Interceptors (chain, unary):                 │
│   recovery → zap-logging → protovalidate →    │
│   auth(JWT HS256, só RPCs protegidos)         │
└─────────────────────────────────────────────┘
        │
        ▼ (dispatch por serviço)
┌──────────────────┐   ┌──────────────────────────┐
│ AuthHandler       │   │ NationalTeamHandler       │
│  Register / Login │   │  ListNationalTeams        │
└──────┬───────────┘   └──────────┬───────────────┘
       ▼                          ▼
┌──────────────────┐   ┌──────────────────────────┐
│ AuthService       │   │ NationalTeamService       │
│  bcrypt / jwt     │   │  (read-only)              │
└──────┬───────────┘   └──────────┬───────────────┘
       ▼                          ▼
┌──────────────────┐   ┌──────────────────────────┐
│ UserRepository    │   │ NationalTeamRepository    │
│  (sqlc)           │   │  (sqlc)                   │
└──────┬───────────┘   └──────────┬───────────────┘
       └───────────┬──────────────┘
                   ▼
        SQLite (modernc.org/sqlite, arquivo embarcado)
        DB em pt-BR: usuarios, selecoes
        schema: golang-migrate | queries: sqlc (rename → Go en)
```

### 3.2 Componentes / Módulos

| Componente | Responsabilidade | Camada |
|------------|------------------|--------|
| `cmd/server/main.go` | Bootstrap fx: monta server gRPC, interceptors, lifecycle | Composition root |
| `internal/config` | Carregar env via viper (DB path, JWT secret, JWT TTL, porta) | Infra/config |
| `internal/logger` | Logger zap estruturado + interceptor de logging | Observabilidade |
| `internal/db` | Abrir conexão SQLite (modernc), aplicar migrations, prover `*sql.DB`/sqlc `Queries` | Infra/persistência |
| `internal/auth/handler` | `AuthHandler` gRPC: traduz proto ↔ domínio para `Register`/`Login` | Apresentação |
| `internal/auth/service` | `AuthService`: regras de cadastro/login, bcrypt, emissão JWT | Domínio/serviço |
| `internal/auth/repository` | `UserRepository` (sqlc): `CreateUser`, `GetUserByEmail` | Dados |
| `internal/auth/interceptor` | Interceptor de autenticação JWT (valida Bearer, injeta claims no ctx) | Cross-cutting/segurança |
| `internal/auth/token` | Emissão/validação de JWT HS256 (`TokenManager`) | Segurança |
| `internal/nationalteam/handler` | `NationalTeamHandler` gRPC: `ListNationalTeams` | Apresentação |
| `internal/nationalteam/service` | `NationalTeamService`: listagem read-only | Domínio/serviço |
| `internal/nationalteam/repository` | `NationalTeamRepository` (sqlc): `ListNationalTeams`, `GetNationalTeamByID` | Dados |
| `internal/pb/...` | Stubs gerados (protoc-gen-go / protoc-gen-go-grpc) — `internal` pois só este módulo os consome na v1 | Contrato |

> **Convenção de idioma (regra do projeto)**: identificadores de **código Go e proto** ficam **100% em inglês** (`User`, `NationalTeam`, `NationalTeamService`, `ErrDuplicateEmail`…); identificadores do **banco de dados** (tabelas/colunas) ficam **100% em pt-BR** (`usuarios`, `selecoes`, `nome_completo`, `senha_hash`, `selecao_id`, `criado_em`). A ponte é o **sqlc** (`sqlc.yaml` com `rename`/`overrides`), que mapeia colunas pt-BR → campos Go em inglês. Ver §21 (candidato a ADR).

### 3.3 Camadas e Fronteiras

Estilo **Layered** (handler → service → repository), com regra de dependência unidirecional (apresentação depende de serviço, serviço depende de repositório; nunca o inverso). Organização **feature-first**: cada domínio é isolado em seu pacote `internal/<dominio>/`, com sub-pacotes por camada (`handler/`, `service/`, `repository/`, …) e um `fx.Module` que provê o grafo do domínio (decisão registrada — ADR-0002 "fx.Module por domínio").

> **Placement de interfaces (idiomático Go, obrigatório com sub-pacotes por camada)**: a interface do repositório é **declarada no pacote consumidor** (`service`), não exposta pelo `repository`. Assim `service` nunca importa `repository` (direção de dependência aponta para dentro, sem ciclo); o pacote `repository` apenas fornece um tipo concreto que satisfaz a interface, e o `fx` faz o bind concreto→interface no wiring. Isso também viabiliza mock em testes unitários (gate "No Self-Set Mock"). O composition root (`cmd/server`) é o único ponto que conhece todos os módulos. O modelo de `User` é coeso e isolado; a relação futura com figurinhas/trocas **não é antecipada** nesta versão (apenas não inviabilizada — `usuarios.id` é UUID estável referenciável).

---

## 4. Contratos de API

> Comunicação exclusivamente **gRPC** (HTTP/2 + protobuf). Não há REST/gRPC-Gateway na v1.

### 4.1 Endpoints Expostos (RPCs)

| Ação | RPC | Serviço (proto) | Request | Response | Status gRPC | Auth |
|------|-----|-----------------|---------|----------|-------------|------|
| Cadastrar usuário | `Register` | `wc2026.auth.v1.AuthService` | `RegisterRequest` | `RegisterResponse` | OK / InvalidArgument / AlreadyExists | Não |
| Autenticar | `Login` | `wc2026.auth.v1.AuthService` | `LoginRequest` | `LoginResponse` | OK / InvalidArgument / Unauthenticated | Não |
| Listar seleções | `ListNationalTeams` | `wc2026.nationalteam.v1.NationalTeamService` | `ListNationalTeamsRequest` (vazio) | `ListNationalTeamsResponse` | OK | Não |

> **RPC protegido de referência**: para validar o interceptor de auth (CA-07), o `AuthService` expõe um RPC protegido de referência `GetMe` (`GetMeRequest` → `GetMeResponse` com dados do usuário do token). É o exemplo canônico de RPC que **exige** Bearer JWT — usado pelos testes E2E CT-029/CT-030/CT-031. `ListNationalTeams` permanece **público** (sem auth) por ser consumido no fluxo de cadastro (CA-10).

### 4.1.1 Exemplo de Payload por Endpoint

> N/A — não há verbos de atualização parcial (`PUT`/`PATCH`) nesta feature. Todos os RPCs de escrita (`Register`) exigem o conjunto completo de campos obrigatórios, validados por protovalidate. Sem semântica de "campos opcionais ignorados".

### 4.2 Schemas / DTOs

| Schema | Origem | Campos principais | Versão |
|--------|--------|-------------------|--------|
| `RegisterRequest` | proto + buf.validate | `full_name` (string, required, min_len), `email` (string, required, email), `password` (string, required, min_len 8), `national_team_id` (string, required, uuid) | v1 |
| `RegisterResponse` | proto | `user_id` (string, uuid) | v1 |
| `LoginRequest` | proto + buf.validate | `email` (string, required, email), `password` (string, required) | v1 |
| `LoginResponse` | proto | `access_token` (string, JWT), `expires_at` (google.protobuf.Timestamp) | v1 |
| `ListNationalTeamsRequest` | proto | (vazio) | v1 |
| `ListNationalTeamsResponse` | proto | `national_teams` (repeated `NationalTeam`) | v1 |
| `NationalTeam` | proto | `id` (string, uuid), `name` (string) | v1 |
| `GetMeResponse` | proto | `user_id`, `full_name`, `email`, `national_team_id` | v1 |

> Campos proto em **inglês** (`national_team_id`); o mapeamento para a coluna `selecao_id` da tabela `usuarios` (pt-BR) ocorre na camada repository/sqlc.

### 4.3 Eventos Publicados / Consumidos

N/A — não há mensageria/eventos nesta versão (pre-refinement §10: mensageria N/A).

---

## 5. Fluxos de Negócio

### 5.1 Fluxo Principal

**Cadastro (`Register`)**:
1. Cliente envia `RegisterRequest`.
2. Interceptor **protovalidate** valida campos (obrigatórios, e-mail, uuid, min_len) → falha ⇒ `InvalidArgument` antes do handler.
3. `AuthHandler.Register` mapeia request → service.
4. `AuthService` verifica seleção existente (`NationalTeamRepository.GetNationalTeamByID`) → inexistente ⇒ `InvalidArgument` (RN4/CA-04). *(Decisão fixada no challenge: `InvalidArgument` único — o `national_team_id` é um argumento semanticamente inválido; alinha com RN3/RN4 em §6.3 e elimina a ambiguidade `InvalidArgument/FailedPrecondition`.)*
5. `AuthService` verifica e-mail livre (`UserRepository.GetUserByEmail`) → existente ⇒ `AlreadyExists` (RN1/CA-02). *(A constraint UNIQUE no banco é a garantia final em caso de corrida.)*
6. `AuthService` gera hash bcrypt (cost 12) da senha (RN2/CA-08).
7. `AuthService` gera UUID v4 e chama `UserRepository.CreateUser`.
8. Handler retorna `RegisterResponse{user_id}`.

**Login (`Login`)**:
1. Cliente envia `LoginRequest`; protovalidate valida formato.
2. `AuthService` busca usuário por e-mail (`GetUserByEmail`).
3. Usuário inexistente OU senha inválida (`bcrypt.CompareHashAndPassword`) ⇒ `Unauthenticated` com **mensagem genérica idêntica** (RN5/CA-06 — não revela qual campo falhou).
   - **Equalização de timing (anti-enumeração)**: quando `GetUserByEmail` retorna *não encontrado*, o service **ainda executa** `bcrypt.CompareHashAndPassword` contra um **hash dummy fixo** (gerado no boot) antes de retornar `Unauthenticated`. Sem isso, o caminho "e-mail inexistente" retornaria muito mais rápido que "senha errada", abrindo canal de *timing* para enumeração de e-mails que a mensagem genérica sozinha não fecha (RN5).
4. Credenciais válidas ⇒ `TokenManager` emite JWT HS256 (claims: `sub`=user_id, `exp`=agora+1h) (CA-05).
5. Handler retorna `LoginResponse{access_token, expires_at}`.

**Acesso protegido (`GetMe`)**:
1. Cliente envia metadata `authorization: Bearer <jwt>`.
2. Interceptor **auth** extrai e valida o token (assinatura HS256, `exp`, alg) → inválido/ausente/expirado ⇒ `Unauthenticated` sem invocar handler (CA-07).
3. Válido ⇒ injeta claims no `context`; handler lê `sub` e retorna dados.

### 5.2 Fluxos Alternativos

- `ListNationalTeams` é público e sem parâmetros: retorna sempre a lista do seed (CA-10).
- Campos ausentes/inválidos são barrados no interceptor protovalidate (não chegam ao service) — fluxo unificado para `Register` e `Login`.
- Token com algoritmo `none` ou assimétrico (RS256) é rejeitado mesmo com payload bem formado (defesa contra alg-confusion).

### 5.3 Mapeamento de User Stories

| User Story (PRD) | Fluxo / Endpoint | Componentes Envolvidos |
|------------------|------------------|------------------------|
| US-01 (criar conta) | `Register` | AuthHandler, AuthService, UserRepository, NationalTeamRepository, protovalidate, bcrypt |
| US-02 (login) | `Login` | AuthHandler, AuthService, UserRepository, TokenManager (JWT) |
| US-03 (escolher seleção) | `ListNationalTeams` + `Register` (national_team_id) | NationalTeamHandler/Service/Repository, AuthService (validação FK) |
| US-04 (senha segura) | `Register` (hash) | AuthService (bcrypt), UserRepository (senha_hash) |
| US-05 (fundação 3 SOs) | bootstrap/build | cmd/server, fx, config (viper), db (modernc), build cross-platform |
| US-06 (módulo de referência) | `ListNationalTeams` | módulo `internal/nationalteam` (handler+service+repository+seed) como template |

---

## 6. Regras de Processamento

### 6.1 Validações de Input

| Regra | Onde Aplica | Comportamento em Falha |
|-------|-------------|------------------------|
| `full_name` obrigatório, não vazio | protovalidate (`.proto`) | `InvalidArgument` |
| `email` obrigatório + formato e-mail | protovalidate | `InvalidArgument` |
| `password` obrigatório, min_len 8 | protovalidate | `InvalidArgument` |
| `national_team_id` obrigatório, formato uuid | protovalidate | `InvalidArgument` |
| seleção existe na tabela | AuthService (`GetNationalTeamByID`) | `InvalidArgument` |
| e-mail único | AuthService (`GetUserByEmail`) + UNIQUE no DB | `AlreadyExists` |
| credenciais corretas | AuthService (`bcrypt.CompareHashAndPassword`) | `Unauthenticated` (genérico) |
| JWT válido/não expirado | interceptor auth | `Unauthenticated` |

### 6.2 Transformações de Dados

- proto Request ↔ entidade de domínio (mapper no handler).
- senha plana → hash bcrypt (nunca persistir/loggar a senha plana).
- `time.Time` (`expires_at`) ↔ `google.protobuf.Timestamp` na resposta.
- `national_team_id`/`user_id` como `string` no proto ↔ UUID v4 no domínio ↔ coluna `selecao_id` no banco (sqlc).

### 6.3 Regras de Domínio

| Regra | Descrição | Erro de Domínio Associado |
|-------|-----------|---------------------------|
| RN1 | E-mail é identificador único | `ErrDuplicateEmail` → `AlreadyExists` |
| RN2 | Senha sempre via hash bcrypt, nunca texto plano | (invariante; sem erro de runtime) |
| RN3/RN4 | Seleção favorita deve existir na lista pré-cadastrada | `ErrInvalidNationalTeam` → `InvalidArgument` |
| RN5 | Login só com e-mail+senha correspondentes; resposta de falha **indistinguível** por mensagem **e por tempo** (e-mail inexistente força comparação bcrypt dummy) | `ErrInvalidCredentials` → `Unauthenticated` |
| RN6 | Acesso autenticado tem validade temporária (1h) | `ErrInvalidToken`/`ErrTokenExpired` → `Unauthenticated` |
| RN7 | Validar dados antes de processar | protovalidate → `InvalidArgument` |

---

## 7. Persistência de Dados

### 7.1 Banco de Dados Principal

**Relacional** — SQLite embarcado (arquivo local), driver **`modernc.org/sqlite`** (pure-Go, sem CGO — ADR-0001). Acesso via `database/sql` + queries tipadas geradas por **sqlc**.

**PRAGMAs/configuração obrigatórios na abertura da conexão** (definidos no challenge para evitar erros silenciosos):
- `PRAGMA foreign_keys = ON` — enforcement da FK `selecao_id` (sem isto, CT-019 passa falsamente).
- `PRAGMA journal_mode = WAL` — permite leituras concorrentes com a escrita serializada (melhora `ListNationalTeams`/`GetMe` sob carga).
- `PRAGMA busy_timeout = <ms>` (ex.: 5000) — evita `SQLITE_BUSY`/"database is locked" sob acesso concorrente, fazendo o driver aguardar o lock em vez de falhar imediatamente.
- `db.SetMaxOpenConns(1)` para o pool de escrita (ou pool dedicado de escrita) — alinha o `database/sql` ao modelo single-writer do SQLite e elimina locks de escrita concorrente no MVP. Ver §7.4.

### 7.2 Tabelas / Coleções

| Nome | Colunas / Campos | Tipos | Constraints | Índices |
|------|------------------|-------|-------------|---------|
| `selecoes` | `id` | TEXT (UUID v4) | PK NOT NULL | PK |
| | `nome` | TEXT | NOT NULL, UNIQUE | UNIQUE(nome) |
| `usuarios` | `id` | TEXT (UUID v4) | PK NOT NULL | PK |
| | `nome_completo` | TEXT | NOT NULL | |
| | `email` | TEXT | NOT NULL, UNIQUE | UNIQUE(email) |
| | `senha_hash` | TEXT | NOT NULL | |
| | `selecao_id` | TEXT | NOT NULL, FK→`selecoes(id)` | idx(selecao_id) |
| | `criado_em` | TIMESTAMP | NOT NULL DEFAULT CURRENT_TIMESTAMP | |

> **Identificadores do banco em pt-BR** (regra do projeto: DB 100% pt-BR). O mapeamento para os campos Go em inglês (ex.: `nome_completo`→`FullName`, `senha_hash`→`PasswordHash`, `selecao_id`→`NationalTeamID`, `criado_em`→`CreatedAt`, struct `usuarios`→`User`) é feito via `rename`/`overrides` no `sqlc.yaml` — ver §21 (candidato a ADR).

### 7.3 Migrações

| Versão | Arquivo | Operação |
|--------|---------|----------|
| 000001 | `000001_create_selecoes.up.sql` / `.down.sql` | cria tabela `selecoes` / drop |
| 000002 | `000002_seed_selecoes.up.sql` / `.down.sql` | popula seed de seleções (lista fixa de seleções da Copa) / delete |
| 000003 | `000003_create_usuarios.up.sql` / `.down.sql` | cria tabela `usuarios` (+FK, +índices) / drop |

> Migrations aplicadas no boot via golang-migrate (lifecycle fx `OnStart`). Seed da v1 é fixo (sem CRUD administrativo — fora de escopo).
>
> **⚠️ Driver pure-Go obrigatório (challenge)**: usar o driver de banco **`sqlite`** do golang-migrate (baseado em `modernc.org/sqlite`) — **NUNCA** o driver `sqlite3` (mattn/CGO). O driver CGO reintroduziria a dependência de toolchain C e **quebraria o build `CGO_ENABLED=0`** (§16.2), violando a ADR-0001. Recomenda-se aplicar as migrations sobre o **mesmo `*sql.DB`** já aberto (`migrate.WithInstance` + driver `sqlite`), reaproveitando a conexão modernc e os PRAGMAs de §7.1.

### 7.4 Estratégia de Transação e Consistência

- SQLite com **single-writer** (lock file-level). Operações de `Register` são simples (1 insert) — sem necessidade de transação multi-statement na v1.
- **Pool alinhado ao single-writer**: `SetMaxOpenConns(1)` + `busy_timeout` (§7.1) garantem que escritas concorrentes sejam serializadas pelo `database/sql` em vez de colidirem em "database is locked". WAL (§7.1) preserva concorrência de leitura.
- A unicidade de e-mail é garantida em duas camadas: checagem prévia no service (UX/mensagem clara) **e** constraint `UNIQUE` no banco (garantia real contra corrida).
- Sem isolation level customizado; default do SQLite (`SERIALIZABLE` efetivo no single-writer).

### 7.5 Política de Retenção / Archival

N/A — sem soft-delete, TTL ou particionamento na v1. Dados de usuário/seleção são permanentes.

---

## 8. Integração com APIs Externas

N/A — não há integrações externas, webhooks ou SDKs de terceiros nesta versão (pre-refinement §10).

---

## 9. Sincronização de Dados

N/A — sem filas, eventos ou consistência distribuída na v1.

### 9.1 Eventos / Filas
N/A.

### 9.2 Idempotência
N/A — `Register` não é idempotente por design (segunda chamada com mesmo e-mail ⇒ `AlreadyExists`, comportamento esperado).

### 9.3 Outbox / Saga
N/A.

---

## 10. Gerenciamento de Erros

### 10.1 Mapeamento Erro de Negócio → Status gRPC

| Erro | Código gRPC | Mensagem | Camada de Origem |
|------|-------------|----------|------------------|
| Campo inválido/ausente | `InvalidArgument` | detalhe do protovalidate | interceptor protovalidate |
| Seleção inexistente | `InvalidArgument` | "seleção favorita inválida" | service |
| E-mail duplicado | `AlreadyExists` | "e-mail já está em uso" | service (+UNIQUE) |
| Credenciais inválidas | `Unauthenticated` | "e-mail ou senha inválidos" (genérico) | service |
| Token ausente/inválido/expirado | `Unauthenticated` | "não autenticado" | interceptor auth |
| Erro interno (DB, etc.) | `Internal` | "erro interno" (sem vazar detalhe) | qualquer |

### 10.2 Resiliência

- Interceptor de **recovery** converte panic em `Internal` (não derruba o servidor).
- Sem retry/circuit-breaker (sem dependências externas).
- Lifecycle fx garante shutdown gracioso (fecha conexão DB no `OnStop`).

### 10.3 Estratégia de Logging de Erros

- Logs estruturados zap. **Nunca** logar senha plana, `senha_hash` ou o JWT completo.
- Erros de domínio: nível `warn`; erros internos: `error` com stack.

---

## 11. Segurança

### 11.1 Autenticação

JWT **HS256**, validado por interceptor `go-grpc-middleware/auth` (extrai `authorization: Bearer <token>` da metadata). Claims: `sub` (user_id), `exp` (1h). Validação inclui assinatura, expiração e **algoritmo esperado** (rejeita `none`/RS256 — defesa contra alg-confusion). Login emite o token; RPCs protegidos (ex.: `GetMe`) exigem-no.

### 11.2 Autorização

N/A formal — não há papéis/RBAC na v1. A única regra é "autenticado vs não autenticado" (RPCs públicos: `Register`, `Login`, `ListNationalTeams`; protegidos: `GetMe`).

### 11.3 Criptografia

- Senhas: **bcrypt** cost 12 (`golang.org/x/crypto/bcrypt`). Nunca em texto legível (CA-08).
- **Anti-enumeração por timing no Login**: e-mail inexistente força um `bcrypt.CompareHashAndPassword` contra hash dummy fixo, equalizando o tempo de resposta com o caminho "senha errada" (ver §5.1; coberto por CT-008/CT-009).
- JWT: assinatura HMAC-SHA256 com segredo.
- TLS: terminação fora do escopo deste backend (responsabilidade do ambiente de deploy/proxy na v1).

### 11.4 Sanitização e Validação

- SQL injection: **impossível por design** — sqlc gera queries parametrizadas (prepared statements).
- Validação de schema de entrada: protovalidate no interceptor.

### 11.5 Rate Limiting / Anti-abuse

N/A na v1 (sem proteção anti-bruteforce de login). Registrado como risco (§20).

### 11.6 Secrets Management

- `JWT_SECRET`, `JWT_TTL`, `DB_PATH`, `GRPC_PORT` lidos via **viper** (env/arquivo). Segredo **nunca** versionado nem logado. Em produção, injetado por variável de ambiente.
- **Fail-fast no boot (challenge)**: `internal/config` valida o `JWT_SECRET` na inicialização e **aborta o start** (erro no `fx OnStart`) se estiver **ausente, vazio ou com menos de 32 bytes**. HS256 com segredo curto/fraco é vulnerável a brute-force offline; falhar cedo evita subir um servidor inseguro silenciosamente.

---

## 12. Performance

### 12.1 Metas

- Latência p95/p99: **N/A formal na v1** (fundação; sem SLA definido — PRD §9 "sem prazo/janela").
- Throughput: não dimensionado (MVP).

### 12.2 Estratégias

- Índices UNIQUE em `usuarios.email` e `selecoes.nome`; índice em `usuarios.selecao_id`.
- Connection pooling padrão do `database/sql` (SQLite single-writer limita escrita concorrente — aceitável no MVP).
- bcrypt cost 12 é intencionalmente custoso (segurança > latência no login).

### 12.3 Limites Conhecidos

- SQLite single-writer: gargalo de escrita sob alta concorrência (mitigação futura: migração a Postgres, viável pela interface `database/sql` — ADR-0001).
- modernc.org/sqlite levemente mais lento que CGO sob carga pesada (trade-off aceito — ADR-0001).

---

## 13. Logs e Observabilidade

### 13.1 Logs Estruturados

| Evento | Nível | Campos Chave | Sensibilidade |
|--------|-------|--------------|---------------|
| RPC recebido/respondido | info | rpc, code, duration_ms | — |
| Cadastro criado | info | user_id, selecao_id | **sem** senha/hash |
| Login bem-sucedido | info | user_id | **sem** token |
| Credenciais inválidas | warn | email_hash (ou omitido) | mascarar PII |
| Erro interno | error | rpc, error, stack | sem dados sensíveis |

Padrão: JSON estruturado via **zap**, emitido por interceptor de logging.

### 13.2 Métricas

N/A na v1 (sem Prometheus/OTel). Registrado como evolução futura.

### 13.3 Tracing

N/A na v1.

### 13.4 Alertas

N/A na v1.

---

## 14. Feature Flags

N/A — sem solução de feature flags nesta versão. Comportamento fixo.

---

## 15. Versionamento de API

### 15.1 Estratégia

Versão no **pacote proto e no path**: `wc2026.<dominio>.v1` em `proto/wc2026/<dominio>/v1/`. Padrão Buf/Google API (versão no namespace, não em header/URL — gRPC não usa path REST).

### 15.2 Compatibilidade

- Mudanças aditivas (novos campos/RPCs) mantêm `v1`. Breaking changes ⇒ novo pacote `v2` coexistindo.
- Sem janela de descontinuação formal definida (fundação inicial).

### 15.3 Schemas / Contratos

- `.proto` é a fonte de verdade; stubs gerados por protoc-gen-go/protoc-gen-go-grpc.
- Validação declarada via opções `buf.validate.*` (protovalidate) no próprio `.proto`.

---

## 16. Deploy e Infraestrutura

### 16.1 Pipeline

CI/CD formal **fora de escopo da v1** (PRD §4.2). Recomendação registrada (§19/§21): build matrix multiplataforma.

### 16.2 Empacotamento

- Binário único Go compilado com `CGO_ENABLED=0` (pure-Go, graças a modernc.org/sqlite — ADR-0001).
- Cross-compile para `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64` sem toolchain C.

### 16.3 Infraestrutura como Código

N/A na v1 (sem Terraform/Helm). Binário executado diretamente como serviço.

### 16.4 Estratégia de Rollout

N/A — execução local/manual do binário na v1.

### 16.5 Escalabilidade

Vertical apenas (instância única + SQLite embarcado). Escala horizontal exigiria banco compartilhado (futuro).

### 16.6 Rollback

N/A formal — rollback = subir binário anterior; migrations possuem `.down.sql` para reverter schema.

---

## 17. Mapeamento de User Stories para Definições Técnicas

| User Story (PRD) | Definição Técnica | Componentes Envolvidos |
|------------------|-------------------|------------------------|
| US-01 | RPC `Register` + validação + hash + persistência | §4.1, §5.1, §6, §7.2; AuthHandler/Service, UserRepository |
| US-02 | RPC `Login` + emissão JWT | §4.1, §5.1, §11.1; AuthService, TokenManager |
| US-03 | RPC `ListNationalTeams` + FK `selecao_id` | §4.1, §6.1, §7.2; NationalTeamHandler/Service/Repository |
| US-04 | bcrypt cost 12 + coluna `senha_hash` | §11.3, §7.2; AuthService |
| US-05 | Fundação fx + modernc + build CGO_ENABLED=0 | §3, §7.1, §16.2; cmd/server, config, db |
| US-06 | Módulo `internal/nationalteam` como template arquitetural | §3.2; pacote nationalteam completo |

---

## 18. Dependências Externas

| Tipo | Nome | Versão | Motivo |
|------|------|--------|--------|
| Linguagem | Go | 1.26.1 | Stack fixada |
| DI / lifecycle | go.uber.org/fx | latest | Injeção de dependências (ADR-0002) |
| Driver DB | modernc.org/sqlite | latest | SQLite pure-Go cross-platform (ADR-0001) |
| Queries | sqlc | latest (tool) | Geração de queries tipadas |
| Migrations | golang-migrate/migrate | latest | Versionamento de schema + seed |
| gRPC | google.golang.org/grpc | latest | Protocolo de comunicação |
| Protobuf | protoc-gen-go / protoc-gen-go-grpc | latest (tools) | Geração de stubs |
| Validação | github.com/bufbuild/protovalidate-go | latest | Validação declarativa no .proto |
| Auth middleware | go-grpc-middleware/auth | v2 | Interceptor de autenticação |
| JWT | github.com/golang-jwt/jwt | v5 | Emissão/validação HS256 |
| Crypto | golang.org/x/crypto/bcrypt | latest | Hash de senha |
| Config | github.com/spf13/viper | latest | Configuração via env |
| Logging | go.uber.org/zap | latest | Logs estruturados |
| Testes | github.com/stretchr/testify | latest | Asserts/mock/suite |
| UUID | github.com/google/uuid | latest | Geração UUID v4 (crypto/rand) |

---

## 19. Estratégia de Testes

> **Resumo**: 35 casos de teste | Unitários: 17 | Integração: 11 | E2E: 6 | Build cross-platform: 1 (CT-033)
> **Padrão**: `go test` + `testify` (assert/require/suite/mock). Integração com SQLite in-memory (`modernc.org/sqlite`) + golang-migrate. E2E via `bufconn` com interceptors reais (protovalidate + auth JWT). Gerado pelo `qa-test-generator` aplicando os 7 gates de `testing-best-practices` (mock_budget observado; gates: invariant_first, owning_layer, real_execution, failure_means_fix_production, no_snapshot_without_contract, no_self_set_mock, negative_companion).

### Rastreabilidade: Critérios de Aceite → Testes

| CA (PRD) | Descrição Resumida | Testes |
|----------|--------------------|--------|
| CA-01 | Cadastro com dados+seleção válida cria conta | CT-001, CT-018, CT-022, CT-027, CT-029, CT-034 |
| CA-02 | E-mail duplicado é impedido | CT-004, CT-018 |
| CA-03 | Campo ausente/e-mail inválido é rejeitado | CT-003, CT-005, CT-028 |
| CA-04 | Seleção inexistente rejeita cadastro | CT-006, CT-019, CT-025 |
| CA-05 | Login correto concede JWT temporário | CT-007, CT-010, CT-016, CT-022, CT-029 |
| CA-06 | Credenciais incorretas negam sem revelar campo | CT-008, CT-009, CT-017, CT-023 |
| CA-07 | Sessão expirada nega acesso protegido | CT-011-interceptor, CT-012, CT-013, CT-014, CT-030, CT-031 |
| CA-08 | Senha nunca em texto legível | CT-002, CT-026 |
| CA-09 | Compila cross-platform (4 targets); auth sobe via wiring | CT-033 (build), CT-035 (wiring) |
| CA-10 | Lista seleções pré-cadastradas | CT-015, CT-020, CT-021, CT-024, CT-032 |

> **Nota de cobertura**: todos os 10 CA-XX possuem ≥1 teste. CT-010 valida tanto o caminho de token válido (CA-05) quanto a fronteira de expiração (CA-07).

### 19.1 Testes Unitários

#### Service: AuthService (`internal/auth/service/auth_service_test.go`)

Mock: `UserRepository`, `NationalTeamRepository`, `TokenManager`, `Clock` (injetado para determinismo).

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-001 | TestRegister_Success | CA-01, CA-08 | Cadastro válido cria usuário; `CreateUser` recebe hash ≠ senha plana | dados válidos + seleção existente | sucesso, user_id | repo GetUserByEmail=nil, CreateUser=nil, GetNationalTeamByID=ok |
| CT-002 | TestRegister_PasswordNeverPlaintext | CA-08 | `PasswordHash` ≠ senha e `bcrypt.CompareHashAndPassword` valida | senha "segredo123" | hash bcrypt verificável | captura arg de CreateUser |
| CT-004 | TestRegister_DuplicateEmail | CA-02 | E-mail existente ⇒ `AlreadyExists` sem vazar info | e-mail já usado | codes.AlreadyExists | GetUserByEmail retorna user |
| CT-006 | TestRegister_NonexistentNationalTeam | CA-04 | national_team_id inexistente ⇒ `codes.InvalidArgument` | national_team_id inválido | codes.InvalidArgument | GetNationalTeamByID=not found |
| CT-007 | TestLogin_Success_JWT_TTL1h | CA-05 | Credenciais corretas ⇒ JWT com exp = now+1h | e-mail+senha corretos | token, expires_at=+1h | Clock fixo, repo retorna user |
| CT-008 | TestLogin_WrongPassword | CA-06 | Senha errada ⇒ Unauthenticated genérico | senha errada | codes.Unauthenticated | repo retorna user |
| CT-009 | TestLogin_NonexistentEmail_SameMessage | CA-06 | E-mail inexistente ⇒ mensagem idêntica à de senha errada **e** comparação bcrypt dummy é executada (equalização de timing, RN5) | e-mail inexistente | Unauthenticated, msg == CT-008; `bcrypt.CompareHashAndPassword` invocado mesmo sem usuário | GetUserByEmail=nil; espião no comparador bcrypt |
| CT-016 | TestGetUserByEmail_Found | CA-05 | Repositório retorna usuário existente (via interface) | e-mail válido | user | repo mock |
| CT-017 | TestGetUserByEmail_NotFound | CA-06 | E-mail inexistente ⇒ not-found | e-mail ausente | ErrNotFound | repo mock |

#### Apresentação/Cross-cutting: Interceptor de validação (`internal/auth/interceptor/protovalidate_test.go`)

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-003 | TestRegister_MissingRequiredField | CA-03 | Campo obrigatório ausente ⇒ InvalidArgument antes do service | request sem full_name | codes.InvalidArgument, service não chamado | — (table-driven recomendado) |
| CT-005 | TestRegister_InvalidEmailFormat | CA-03 | E-mail mal formado ⇒ InvalidArgument via protovalidate | email="abc" | codes.InvalidArgument | — |

#### Cross-cutting: Interceptor JWT (`internal/auth/interceptor/auth_jwt_test.go`)

Mock: `Clock` injetado; `TokenManager` real (HS256 com secret de teste).

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-010 | TestJWT_ValidToken_AllowsAccess | CA-05, CA-07 | Token válido/não expirado ⇒ handler executa, claims no ctx | Bearer válido | handler chamado, sub no ctx | Clock fixo |
| CT-011-interceptor | TestJWT_ExpiredToken | CA-07 | Token expirado ⇒ Unauthenticated, handler não chamado | exp no passado | codes.Unauthenticated | Clock avançado |
| CT-012 | TestJWT_InvalidSignature | CA-07 | Secret errado ⇒ Unauthenticated | token assinado c/ outro secret | codes.Unauthenticated | — |
| CT-013 | TestJWT_MissingToken | CA-07 | Sem metadata authorization ⇒ Unauthenticated | sem Bearer | codes.Unauthenticated | — |
| CT-014 | TestJWT_AlgNoneOrRS256_Rejected | CA-07 | Algoritmo none/RS256 rejeitado (alg-confusion) | token alg=none | codes.Unauthenticated | — |

#### Service: NationalTeamService (`internal/nationalteam/service/national_team_service_test.go`)

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-015 | TestListNationalTeams_ReturnsList | CA-10 | Retorna lista não-vazia mockada | — | lista de seleções | NationalTeamRepository mock |

> **Recomendação do QA**: consolidar CT-003/CT-005 em um teste table-driven (`register_validation_test.go`) e CT-011-interceptor/CT-012/CT-013 em `interceptor_rejects_invalid_tokens_test.go`, mantendo a cobertura dos cenários.

### 19.2 Testes de Integração

#### Integração: Service + Repository + SQLite in-memory

- **Setup**: helper `TestNewDB(t)` abre SQLite in-memory (`modernc.org/sqlite`), aplica migrations via golang-migrate, habilita `PRAGMA foreign_keys = ON`, registra `t.Cleanup`. Reutilizado por CT-018..CT-026, CT-034, CT-035.
- **Cenários**:

| CT | Teste | CA | Fluxo | Validação |
|----|-------|----|-------|-----------|
| CT-018 | TestIntegration_Register_UniqueEmail | CA-01, CA-02 | Register OK; segundo Register mesmo e-mail | 1º cria; 2º ⇒ AlreadyExists (UNIQUE real) |
| CT-019 | TestIntegration_FK_NonexistentNationalTeam | CA-04 | Insert com selecao_id ausente no DB | FK rejeita (foreign_keys ON) |
| CT-020 | TestIntegration_SeedNationalTeams_List | CA-10 | Migrations aplicam seed; ListNationalTeams | retorna seleções do seed |
| CT-021 | TestIntegration_NoMigrations_Error | CA-10 | Consultar selecoes sem migrar | erro de tabela inexistente |
| CT-022 | TestIntegration_Register_Login_JWT | CA-01, CA-05, CA-08 | Register → Login sequencial | JWT verificável; hash no DB |
| CT-023 | TestIntegration_Login_WrongPassword | CA-06 | Register depois Login senha errada | codes.Unauthenticated |
| CT-024 | TestIntegration_GetNationalTeamByID_Seed | CA-10 | GetNationalTeamByID de id do seed | seleção correta |
| CT-025 | TestIntegration_GetNationalTeamByID_NotFound | CA-04 | GetNationalTeamByID id inexistente | not-found |
| CT-026 | TestIntegration_PasswordHash_RealDB | CA-08 | Ler `senha_hash` persistido | hash ≠ senha original (DB real) |
| CT-034 | TestIntegration_UUIDv4_Unique | CA-01 | 2 inserções consecutivas | usuarios.id válido UUID v4 e únicos |
| CT-035 | TestIntegration_FxWiring_Smoke | CA-09 | Sobe AuthService via fx + DB in-memory | grafo fx monta sem erro (smoke wiring) |

### 19.3 Testes End-to-End (E2E)

- **Framework**: gRPC black-box via `bufconn` (`google.golang.org/grpc/test/bufconn`), servidor montado com interceptors reais (protovalidate + auth JWT). Helper `TestNewBufconnServer(t)` retorna conn + teardown. Reutilizado por CT-027..CT-032.

#### Fluxo: Register E2E (CT-027)
- **CA**: CA-01, CA-09 — **Pré-condições**: server bufconn + seed aplicado.
- **Passos**: 1) cliente chama `Register` com dados válidos; 2) recebe `user_id`.
- **Validações**: status OK, user_id não vazio.

#### Fluxo: Register validação E2E (CT-028)
- **CA**: CA-03 — campo ausente ⇒ `InvalidArgument` via interceptor protovalidate real (não chega ao handler).

#### Fluxo: Register → Login → RPC protegido (CT-029)
- **CA**: CA-01, CA-05, CA-07 — **Pré-condições**: server bufconn.
- **Passos**: 1) `Register`; 2) `Login` → token; 3) chamar `GetMe` com `Bearer <token>`.
- **Validações**: GetMe retorna dados do usuário; fluxo completo OK.

#### Fluxo: RPC protegido sem token (CT-030)
- **CA**: CA-07 — chamar `GetMe` sem metadata ⇒ `Unauthenticated`.

#### Fluxo: RPC protegido com token expirado (CT-031)
- **CA**: CA-07 — token com exp no passado (Clock injetado) ⇒ `Unauthenticated`.

#### Fluxo: ListNationalTeams público (CT-032)
- **CA**: CA-10 — `ListNationalTeams` **sem** token retorna seleções do seed (RPC público).

### 19.4 Cenários de Erro

| Cenário | CA | Trigger | Comportamento Esperado | Status gRPC | Log/Observabilidade |
|---------|----|---------|------------------------|-------------|----------------------|
| E-mail duplicado | CA-02 | 2º Register mesmo e-mail | impede criação | AlreadyExists | warn "email já em uso" |
| Campo ausente / e-mail inválido | CA-03 | request incompleto | barra no interceptor | InvalidArgument | detalhe protovalidate |
| Seleção inexistente | CA-04 | national_team_id inválido | rejeita cadastro | InvalidArgument | warn |
| Credenciais inválidas | CA-06 | senha/e-mail errado | nega sem revelar campo | Unauthenticated (msg genérica) | warn (sem PII) |
| Token expirado/ausente/inválido | CA-07 | RPC protegido | nega acesso | Unauthenticated | warn |
| Alg-confusion (none/RS256) | CA-07 | token forjado | rejeita | Unauthenticated | warn segurança |

> **Cenários não cobertos (justificados)**: refresh/rotação de token (fora v1 — TTL fixo 1h); recuperação de senha/verificação de e-mail (fora v1); testes de carga/throughput (recomendar ghz/k6 pós-fundação); race de inserção concorrente do mesmo e-mail (SQLite single-writer + UNIQUE cobre; risco real só em Postgres futuro); SQL injection (impossível por design com sqlc parametrizado); CRUD admin de seleções (fora escopo); validação de logging zap (baixo valor de teste, alto acoplamento).
>
> **Recomendações técnicas do QA** (acionáveis na execução): (1) helpers `TestNewDB` e `TestNewBufconnServer` reutilizáveis; (2) **injetar `Clock` via interface** no AuthService e interceptor JWT para determinismo (mandatório p/ CT-007, CT-010, CT-011, CT-031); (3) `PRAGMA foreign_keys = ON` no setup (senão CT-019 passa silenciosamente); (4) garantir UUID v4 com `crypto/rand` (não `math/rand`); (5) build matrix CI (linux/darwin/windows, `CGO_ENABLED=0`) para CA-09; (6) bcrypt cost configurável (10 em CI, 12 em prod) se hash >500ms no CI.

### 19.5 Verificação de Build Cross-Platform (CT-033)

> **Definido no challenge** — antes, CT-033 era referenciado (CA-09) mas não especificado, e a contagem de testes não fechava.

| CT | Verificação | CA | Mecanismo | Expected |
|----|-------------|----|-----------|----------|
| CT-033 | Cross-compile dos 4 targets | CA-09 | `go build` / alvo `make build-all` com `CGO_ENABLED=0` para `linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64` | os 4 binários compilam sem toolchain C (prova de portabilidade — ADR-0001) |

> **Escopo honesto de CA-09 na v1**: CT-033 garante **compilação cross-platform** (build, não runtime) e CT-035 garante que o **grafo fx monta** (wiring smoke num SO). A **execução runtime efetiva nos três SOs** é validada **manualmente** na v1 — a CI matrix automatizada (rodar a suíte em Windows/Linux/macOS) é **evolução futura** (CI/CD fora de escopo, §16.1 / PRD §4.2). Não há, portanto, prova automatizada de "roda nos 3 SOs em runtime" nesta versão; o risco residual é baixo porque o binário é pure-Go (`CGO_ENABLED=0`) sem dependências de plataforma.

---

## 20. Riscos Técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|---------------|---------|-----------|
| SQLite single-writer vira gargalo de escrita | Média (em escala) | Médio | Interface `database/sql` permite migrar a Postgres sem reescrever camada de dados (ADR-0001) |
| Sem rate-limiting no Login (bruteforce) | Média | Médio | bcrypt cost 12 desacelera tentativas; adicionar rate-limit em versão futura (§11.5) |
| FK não enforced sem `PRAGMA foreign_keys=ON` | Alta se esquecido | Alto (CA-04 falha silenciosa) | Habilitar pragma na abertura da conexão; CT-019 valida |
| Determinismo de testes com `time.Now()` | Alta | Médio | Injetar `Clock` via interface (recomendação QA) |
| `JWT_SECRET` fraco/versionado | Baixa | Alto | Segredo só via env (viper); nunca em repo; **fail-fast no boot se ausente/vazio/<32 bytes** (§11.6) |
| Enumeração de e-mails por timing no Login | Média | Médio | **Mitigado**: comparação bcrypt dummy no caminho "e-mail inexistente" equaliza o tempo (§5.1/§11.3, CT-009) |
| Reintrodução de CGO via driver `sqlite3` do golang-migrate | Média se esquecido | Alto (quebra build `CGO_ENABLED=0`, viola ADR-0001) | Usar driver `sqlite` (pure-Go/modernc) do golang-migrate; aplicar via `WithInstance` no mesmo `*sql.DB` (§7.3) |
| TTL 1h sem refresh frustra sessões longas | Baixa (v1) | Baixo | Aceito na v1; refresh token planejado para versão futura |
| Erro de DI só em runtime (fx) | Média | Médio | CT-035 (smoke de wiring) detecta no boot/teste |

---

## 21. Observações Técnicas

### ADRs Aplicáveis nesta Feature (inventário FASE 4.6)

> `docs/adr/` possui 2 ADRs `accepted` (não há `INDEX.md` ainda — recomenda-se rodar `/adr-reindex`).

| ADR | Classificação | Como a feature obedece |
|-----|---------------|------------------------|
| **ADR-0001** — driver modernc.org/sqlite (pure-Go) | **APLICÁVEL** | §7.1 usa `modernc.org/sqlite`; §16.2 compila com `CGO_ENABLED=0`; CT-033/CT-035 validam build/wiring cross-platform (CA-09). Interface `database/sql` preservada (§12.3 prevê migração futura). |
| **ADR-0002** — uber-fx + go-standard-layout | **APLICÁVEL** | §3.2/§3.3 organizam o código em `cmd/` + `internal/` (com `pkg/` reservado, não usado na v1) com um `fx.Module` por domínio; lifecycle fx aplica migrations no `OnStart` e fecha conexão no `OnStop` (§7.3, §10.2); CT-035 é o smoke de wiring fx. |

### Candidatos a ADR (hook FASE 4.5)

> Avaliação dos 5 critérios canônicos (C1 transversal, C2 tag, C3 reversão cara, C4 surpreendente, C5 trade-off real). **Não criar ADR automaticamente** — usuário decide via `/adr-create`.

- **Candidato a ADR confirmado (5/5) — Estratégia de autenticação: JWT HS256 + bcrypt + TTL 1h sem refresh**
  - tag: `auth` / `security`.
  - C1: transversal — todo RPC protegido futuro dependerá deste padrão. ✓
  - C2: cai em `auth`. ✓
  - C3: reverter (HS256→RS256, ou trocar algoritmo de hash) implica refactor em emissão/validação e migração de hashes. ✓
  - C4: surpreendente — futuro leitor questionaria "por que HS256 e não RS256? por que sem refresh?". ✓
  - C5: trade-off real — HS256 (simples, mono-serviço) rejeitou RS256/EdDSA (validação distribuída); bcrypt rejeitou argon2id; sem-refresh aceito por escopo. ✓
  - **Recomendação**: rodar `/adr-create "estrategia de autenticacao JWT HS256 com bcrypt e TTL de 1h sem refresh"`.

- **Candidato a ADR parcial (4/5) — Versionamento de contrato proto via pacote `wc2026.<dominio>.v1`**
  - tag: `architecture` / `api_contracts`.
  - Falha em C4 (não surpreendente — é o padrão Buf/Google amplamente conhecido). C1/C2/C3/C5 passam (transversal a todos os módulos; reverter renomeia pacotes e quebra clientes; trade-off contra pacote sem versão). Registrar como decisão técnica (§15) é suficiente; ADR opcional.

- **Candidato a ADR confirmado (5/5) — Convenção de idioma: banco em pt-BR, código Go/proto em inglês, ponte via sqlc rename**
  - tag: `data` / `cross-cutting`.
  - C1: transversal — toda tabela, query, struct e RPC de qualquer módulo futuro segue esta regra. ✓
  - C2: cai em `data`/`cross-cutting`. ✓
  - C3: reverter (ex.: alinhar tudo a um único idioma) implica renomear tabelas/colunas + migrations + reconfigurar `sqlc.yaml` + ajustar todo o código gerado. ✓
  - C4: surpreendente — um dev novo estranharia "por que a coluna é `senha_hash` mas o campo Go é `PasswordHash`?" sem o registro. ✓
  - C5: trade-off real — manter DB em pt-BR (legibilidade do dado de negócio em português) vs código 100% inglês (idiomático Go/comunidade), conciliados pelo `rename` do sqlc; alternativa rejeitada = idioma único. ✓
  - **Recomendação**: rodar `/adr-create "convencao de idioma: banco de dados em pt-BR e codigo Go/proto em ingles com bridge via sqlc rename"`.

- **Candidato a ADR parcial (4/5) — Versionamento de contrato proto via pacote `wc2026.<dominio>.v1`**
  - tag: `architecture` / `api_contracts`.
  - Falha em C4 (não surpreendente — é o padrão Buf/Google amplamente conhecido). C1/C2/C3/C5 passam. Registrar como decisão técnica (§15) é suficiente; ADR opcional.

- **Candidato a ADR parcial (3/5) — sqlc + golang-migrate como padrão de acesso a dados**
  - tag: `data`. Passa C1 (todos os módulos), C3 (trocar gerador implica reescrever camada de dados), C5 (vs ORM/query manual). Falha C4 (esperado no ecossistema Go) e parcialmente C2 (coberto por ADR-0001 que já cita `data`/`build`). Registrar em §7 é suficiente.

### Notas adicionais

- **Convenção de idioma (regra do projeto)**: banco de dados 100% em pt-BR (`usuarios`, `selecoes`, `nome_completo`, `senha_hash`, `selecao_id`, `criado_em`); código Go e proto 100% em inglês (`User`, `NationalTeam`, `NationalTeamService`, `ErrDuplicateEmail`). Ponte feita por `rename`/`overrides` no `sqlc.yaml`. Promovida a candidato a ADR confirmado (acima).
- **Glossário de domínio criado no challenge**: `/docs/specs/domain-glossary.md` (GLOBAL) com `User` (`usuarios`), `NationalTeam`/Seleção (`selecoes`) e `Sessão`/JWT — entidades de negócio cross-feature (figurinhas/trocas futuras se relacionam a `User`). Registrada a ambiguidade resolvida: **"Seleção Favorita" não é entidade separada** — é a `NationalTeam` escolhida por um `User` (atributo de relacionamento via `selecao_id`). Registrado também que "seleção" (futebol) ↔ "national team" em inglês (mapeamento semântico, **não** "selection").
- **Decisão de escopo — `GetMe` (challenge)**: `GetMe` **não mapeia uma US do PRD**; é uma **decisão técnica deliberada** para expor um RPC protegido de referência que habilita a verificação ponta-a-ponta de CA-07 (CT-029/030/031) e serve de exemplo canônico de RPC autenticado para módulos futuros. Não é overengineering — sem ele, CA-07 não teria alvo real. `ListNationalTeams` é **público**; `GetMe` é **protegido** (resolve a ambiguidade do CT-032 — recomendação QA).

---

## 22. Arquivos Envolvidos e Ações

### 22.0 Visão em Árvore

```
wc_2026_api/
├── cmd/
│   └── server/
│       └── main.go                                      [N]
├── proto/
│   └── wc2026/
│       ├── auth/v1/
│       │   └── auth.proto                                [N]
│       └── nationalteam/v1/
│           └── national_team.proto                       [N]
├── internal/
│   ├── pb/
│   │   └── wc2026/                                       [N] (gerado por buf/protoc)
│   │       ├── auth/v1/{auth.pb.go, auth_grpc.pb.go}
│   │       └── nationalteam/v1/{national_team.pb.go, national_team_grpc.pb.go}
│   ├── config/
│   │   └── config.go                                     [N]
│   ├── logger/
│   │   ├── logger.go                                     [N]
│   │   └── interceptor.go                                [N]
│   ├── db/
│   │   ├── db.go                                         [N]
│   │   ├── module.go                                     [N]
│   │   ├── migrations/
│   │   │   ├── 000001_create_selecoes.up.sql             [N]
│   │   │   ├── 000001_create_selecoes.down.sql           [N]
│   │   │   ├── 000002_seed_selecoes.up.sql               [N]
│   │   │   ├── 000002_seed_selecoes.down.sql             [N]
│   │   │   ├── 000003_create_usuarios.up.sql             [N]
│   │   │   └── 000003_create_usuarios.down.sql           [N]
│   │   ├── queries/
│   │   │   ├── usuarios.sql                               [N]
│   │   │   └── selecoes.sql                               [N]
│   │   └── sqlc/                                          [N] (gerado por sqlc)
│   ├── auth/
│   │   ├── module.go                                     [N]
│   │   ├── handler/auth_handler.go                       [N]
│   │   ├── handler/auth_handler_test.go                  [N]
│   │   ├── service/auth_service.go                       [N]
│   │   ├── service/auth_service_test.go                  [N]
│   │   ├── repository/user_repository.go                 [N]
│   │   ├── token/token_manager.go                        [N]
│   │   ├── interceptor/auth_jwt.go                       [N]
│   │   ├── interceptor/auth_jwt_test.go                  [N]
│   │   ├── interceptor/protovalidate.go                  [N]
│   │   └── interceptor/protovalidate_test.go             [N]
│   ├── nationalteam/
│   │   ├── module.go                                     [N]
│   │   ├── handler/national_team_handler.go              [N]
│   │   ├── service/national_team_service.go              [N]
│   │   ├── service/national_team_service_test.go         [N]
│   │   └── repository/national_team_repository.go        [N]
│   └── testutil/
│       ├── db.go         (TestNewDB)                     [N]
│       └── bufconn.go    (TestNewBufconnServer)          [N]
├── test/
│   └── e2e/
│       ├── auth_e2e_test.go                              [N]
│       └── national_team_e2e_test.go                     [N]
├── buf.yaml / buf.gen.yaml                                [N]
├── sqlc.yaml                                              [N]
├── Makefile                                               [N]
├── .env.example                                           [N]
├── go.mod                                                 [M]
├── go.sum                                                 [M]
└── docs/adr/INDEX.md                                      [N] (via /adr-reindex)

Legenda: [N] Novo  [M] Modificado  [R] Referência
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

### 22.1 Arquivos a Criar

| Arquivo | Descrição | Camada |
|---------|-----------|--------|
| `cmd/server/main.go` | Bootstrap fx (server gRPC, interceptors, lifecycle) | Composition root |
| `proto/wc2026/auth/v1/auth.proto` | Contrato AuthService + buf.validate | Contrato |
| `proto/wc2026/nationalteam/v1/national_team.proto` | Contrato NationalTeamService | Contrato |
| `internal/pb/wc2026/**` | Stubs gerados (protoc-gen-go/-grpc); `buf.gen.yaml` com `out: internal/pb` | Contrato (gerado) |
| `internal/config/config.go` | Config viper (DB_PATH, JWT_SECRET, JWT_TTL, GRPC_PORT) | Infra |
| `internal/logger/logger.go` + `interceptor.go` | zap + interceptor de logging | Observabilidade |
| `internal/db/db.go` + `module.go` | Conexão SQLite modernc + migrations + fx module | Infra/dados |
| `internal/db/migrations/0000{1,2,3}_*.sql` | Schema selecoes/seed/usuarios (up+down) | Migrations |
| `internal/db/queries/{usuarios,selecoes}.sql` | Queries sqlc | Dados |
| `internal/db/sqlc/**` | Código gerado por sqlc | Dados (gerado) |
| `internal/auth/module.go` | fx.Module do domínio auth | Wiring |
| `internal/auth/handler/auth_handler.go` (+ `_test`) | AuthHandler gRPC (Register/Login/GetMe) | Apresentação |
| `internal/auth/service/auth_service.go` (+ `_test`) | Regras auth, bcrypt | Domínio |
| `internal/auth/repository/user_repository.go` | UserRepository (sqlc) | Dados |
| `internal/auth/token/token_manager.go` | Emissão/validação JWT HS256 | Segurança |
| `internal/auth/interceptor/auth_jwt.go` (+ `_test`) | Interceptor de auth JWT | Cross-cutting |
| `internal/auth/interceptor/protovalidate.go` (+ `_test`) | Interceptor de validação | Cross-cutting |
| `internal/nationalteam/module.go` | fx.Module do domínio nationalteam | Wiring |
| `internal/nationalteam/handler/national_team_handler.go` | NationalTeamHandler (ListNationalTeams) | Apresentação |
| `internal/nationalteam/service/national_team_service.go` (+ `_test`) | NationalTeamService read-only | Domínio |
| `internal/nationalteam/repository/national_team_repository.go` | NationalTeamRepository (sqlc) | Dados |
| `internal/testutil/db.go` + `bufconn.go` | Helpers de teste (TestNewDB, TestNewBufconnServer) | Testes |
| `test/e2e/{auth,national_team}_e2e_test.go` | Testes E2E via bufconn | Testes |
| `buf.yaml`, `buf.gen.yaml`, `sqlc.yaml`, `Makefile`, `.env.example` | Tooling/config de build | Build/config |

### 22.2 Arquivos a Modificar

| Arquivo | Modificação | Motivo |
|---------|-------------|--------|
| `go.mod` | Adicionar dependências (grpc, fx, modernc, jwt, bcrypt, viper, zap, testify, protovalidate, uuid, golang-migrate) | Stack |
| `go.sum` | Checksums das deps | Stack |

### 22.3 Arquivos de Referência (somente leitura)

| Arquivo | Motivo da Consulta |
|---------|--------------------|
| `docs/prds/features/arquitetura-base/v1/prd.md` | CA-XX, regras de negócio |
| `docs/specs/features/arquitetura-base/v1/pre-refinement.md` | Stack fixada, dúvidas resolvidas |
| `docs/adr/0001-driver-modernc-sqlite-pure-go-portabilidade.md` | Conformidade driver SQLite |
| `docs/adr/0002-injecao-dependencias-uber-fx-go-standard-layout.md` | Conformidade DI/layout |

---

## 23. Checklist Final

- [x] Variante registrada (backend) na seção 1
- [x] Stack identificada
- [x] TECH_SPEC cobre todo o PRD (todas as US-XX mapeadas em 17)
- [x] Resumo técnico claro e objetivo (seção 2)
- [x] Arquitetura definida com componentes e camadas (seção 3)
- [x] Contratos de API definidos com payloads, status codes e schemas (seção 4)
- [x] Fluxos de negócio descritos (seção 5)
- [x] Regras de processamento e validações (seção 6)
- [x] Persistência: tabelas, índices, migrações, transação (seção 7)
- [x] Integrações externas mapeadas (seção 8 — N/A)
- [x] Sincronização: eventos, idempotência (seção 9 — N/A)
- [x] Gerenciamento de erros e resiliência (seção 10)
- [x] Segurança: auth, autorização, criptografia, sanitização (seção 11)
- [x] Performance: metas, estratégias, limites (seção 12)
- [x] Logs, métricas, tracing e alertas (seção 13)
- [x] Feature flags listadas (seção 14 — N/A)
- [x] Versionamento de API definido (seção 15)
- [x] Deploy e infraestrutura: pipeline, empacotamento, IaC, rollout (seção 16)
- [x] Dependências externas listadas (seção 18)
- [x] Estratégia de testes via `qa-test-generator` integrada (seção 19, com rastreabilidade CA→CT)
- [x] Riscos técnicos identificados (seção 20)
- [x] Observações técnicas registradas (seção 21 — inventário ADR + candidatos)
- [x] Arquivos envolvidos listados — criar, modificar, referência (seção 22)
- [x] Pronto para geração das TASKS
