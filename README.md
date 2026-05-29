# WC 2026 API

API gRPC em Go para o **Álbum da Copa do Mundo 2026** — autenticação de usuários (cadastro/login) e catálogo de seleções nacionais. Construída como fundação arquitetural reproduzível: pure-Go (compila para Linux/macOS/Windows sem toolchain C), injeção de dependências com uber-fx e persistência SQLite embarcada.

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go 1.26.1 |
| Transporte | gRPC (`google.golang.org/grpc`) |
| Contratos | Protocol Buffers + [buf](https://buf.build) + validação declarativa [protovalidate](https://buf.build/go/protovalidate) |
| DI / Lifecycle | [uber-fx](https://github.com/uber-go/fx) |
| Banco | SQLite via [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite) (**pure-Go, sem CGO**) |
| Queries | [sqlc](https://sqlc.dev) (código tipado gerado) |
| Migrations | [golang-migrate](https://github.com/golang-migrate/migrate) (embarcadas via `embed.FS`) |
| Auth | JWT HS256 (`golang-jwt/v5`) + bcrypt (`golang.org/x/crypto`) |
| Config | [viper](https://github.com/spf13/viper) |
| Logs | [zap](https://github.com/uber-go/zap) (estruturado JSON) |
| Testes | [testify](https://github.com/stretchr/testify) + gRPC `bufconn` (E2E) |

> Decisões de arquitetura registradas em [`docs/adr/`](docs/adr/INDEX.md).

---

## Pré-requisitos

- **Go 1.26.1+** — obrigatório para build/test/run.
- **buf** — apenas para **regenerar stubs proto** (`make proto`). Não é necessário para rodar.
- **sqlc** — apenas para **regenerar queries** (`make sqlc`). Não é necessário para rodar.

As migrations são embarcadas no binário (`embed.FS`) e aplicadas no boot — **não há passo manual de migração** nem dependência de CLI de migração em runtime.

---

## Configuração

A API lê a configuração **de variáveis de ambiente** (viper `AutomaticEnv`). O arquivo `.env.example` é apenas um **template de referência** — a aplicação não o carrega automaticamente; exporte as variáveis no seu shell (ou use uma ferramenta como `direnv`/`dotenv`):

```bash
cp .env.example .env
# carregue no shell (exemplo):
export $(grep -v '^#' .env | xargs)
```

| Variável | Obrigatória | Default | Descrição |
|----------|:-----------:|---------|-----------|
| `DB_PATH` | não | `<dir do executável>/wc2026.db` | Caminho do arquivo SQLite. Quando ausente, o banco é criado na **mesma pasta do executável** (independe do diretório de trabalho). Defina para sobrescrever (ex.: `:memory:` em testes, caminho fixo em container) |
| `JWT_SECRET` | sim | — | Segredo HS256. **Mínimo 32 bytes** — o servidor **aborta no boot** se ausente/curto |
| `JWT_TTL` | não | `1h` | Tempo de vida do token (ex.: `1h`, `30m`) |
| `GRPC_PORT` | não | `50051` | Porta TCP do servidor gRPC |

> **Fail-fast de segurança**: um `JWT_SECRET` ausente ou com menos de 32 bytes interrompe a inicialização antes de o servidor abrir a porta. Isso é intencional (não suba um servidor inseguro silenciosamente).

---

## Como rodar

```bash
# 1. Exporte a configuração (ou use um .env carregado pelo seu shell)
export JWT_SECRET="$(openssl rand -base64 48)"   # >= 32 bytes
export GRPC_PORT=50051
# DB_PATH é opcional: sem ele, o banco nasce ao lado do executável.
# Em dev com `go run` o binário é efêmero (temp), então defina um caminho fixo:
export DB_PATH=./data/wc2026.db

# 2. Suba o servidor (migrations + seed de seleções aplicados no boot)
go run ./cmd/server
```

> No **binário compilado** (`make build`/`make build-all`) rodando nativamente, omitir `DB_PATH` cria `wc2026.db` na mesma pasta do executável.

No boot o servidor:
1. Valida a config (fail-fast de `JWT_SECRET`).
2. Abre o SQLite com os PRAGMAs corretos (`foreign_keys=ON`, `journal_mode=WAL`, `busy_timeout=5000`).
3. Aplica as migrations embarcadas (cria `national_teams`/`users` e popula o seed de seleções).
4. Monta a cadeia de interceptors e serve gRPC na `GRPC_PORT`.

Encerramento (`Ctrl+C`) faz **graceful shutdown** e fecha o banco.

### Localização do banco de dados

O caminho do arquivo SQLite é resolvido na inicialização nesta ordem:

1. **`DB_PATH` definido** → usa exatamente esse valor (arquivo relativo/absoluto, ou `:memory:` em testes). Tem precedência sobre tudo.
2. **`DB_PATH` ausente** → usa `wc2026.db` **no mesmo diretório do executável** (`os.Executable()`), independente do diretório de trabalho (CWD) de onde o binário foi chamado.

Implicações práticas:

- **Binário compilado** (`make build` / `make build-all`) rodando nativamente em Windows, Linux ou macOS: sem `DB_PATH`, o banco e seus arquivos WAL (`wc2026.db`, `wc2026.db-wal`, `wc2026.db-shm`) nascem **ao lado do executável**. Não importa de onde você o executou (terminal, duplo-clique, atalho) — o destino é estável.
- O diretório do executável sempre existe, então **não é necessário criar pasta** nem definir variável para o primeiro run: o SQLite cria o arquivo e as migrations embarcadas aplicam schema + seed automaticamente.
- **Desenvolvimento com `go run`**: o Go compila para um diretório temporário efêmero, então o default cairia lá (e seria descartado). Em dev, **defina `DB_PATH`** para um caminho fixo (ex.: `./data/wc2026.db`).
- Para um caminho fixo em produção/container (ex.: volume montado), **defina `DB_PATH`** explicitamente.

> O `.gitignore` ignora `wc2026.db` (e os arquivos `-wal`/`-shm`) além de `./data/`; o banco é um artefato de runtime, nunca versionado.

---

## Build

```bash
make build        # binário para o SO atual (CGO_ENABLED=0)
make build-all    # cross-compile: linux/amd64, linux/arm64, darwin/arm64, windows/amd64
```

Todos os targets compilam com `CGO_ENABLED=0` — prova de portabilidade garantida pelo driver pure-Go (ADR-0001). Binários em `bin/`.

---

## Testes

```bash
make test         # go test ./...  (unit + integração + E2E)
```

- **Unitários**: lógica de domínio com mocks (`internal/domain/auth/service`, `internal/domain/auth/token`, ...).
- **Integração**: contra SQLite real efêmero via `internal/testutil.TestNewDB` (repositórios, migrations, FK/UNIQUE).
- **E2E**: servidor completo em memória via `bufconn` com a cadeia de interceptors **real** (`test/e2e/`, `internal/testutil.TestNewBufconnServer`).

---

## API (resumo dos RPCs)

### `wc2026.auth.v1.AuthService`
| RPC | Acesso | Descrição |
|-----|--------|-----------|
| `Register` | público | Cadastra usuário (valida seleção, e-mail único, hash bcrypt cost 12, UUID v4) |
| `Login` | público | Autentica e emite JWT (`access_token`, `expires_at`) |
| `GetMe` | **protegido** | Retorna dados do usuário autenticado (lê `sub` do token via interceptor) |

### `wc2026.nationalteam.v1.NationalTeamService`
| RPC | Acesso | Descrição |
|-----|--------|-----------|
| `ListNationalTeams` | público | Lista as seleções da Copa (seed) |

Validação de entrada é declarativa (`buf.validate` nos `.proto`) e aplicada pelo interceptor protovalidate antes do handler.

---

## Arquitetura

### Composition root e cadeia de interceptors

O `cmd/server/main.go` monta o grafo fx a partir dos módulos por domínio (`db`, `auth`, `nationalteam`) mais os providers compartilhados de `internal/server`. A cadeia unária de interceptors é montada **em ordem fixa** (decisão arquitetural — não reordenar):

```
recovery → logging → protovalidate → auth JWT → handler
```

`internal/server` existe como pacote próprio (não dentro de `cmd/server`, que é `package main` e não é importável) justamente para que os testes E2E exercitem **a mesma cadeia e wiring** da produção.

### Estrutura de pastas (go-standard-layout)

```
cmd/server/            # composition root (bootstrap fx)
api/proto/wc2026/      # contratos .proto (versionados por domínio)
gen/wc2026/            # stubs proto GERADOS (buf) — não editar à mão
internal/
  domain/              # features (domínio de negócio)
    auth/              # domínio de autenticação
      token/           # TokenManager JWT HS256
      service/         # AuthService (bcrypt, anti-timing) + interfaces consumidas
      repository/      # UserRepository (sqlc)
      handler/         # AuthHandler (mapper proto↔domínio)
      interceptor/     # protovalidate + auth JWT
      module.go        # fx.Module do domínio auth
    nationalteam/      # domínio NationalTeam (módulo de referência read-only)
  infra/               # infraestrutura técnica (cross-cutting)
    config/            # carga de config (viper) + fail-fast JWT_SECRET
    logger/            # zap + interceptor de logging
    clock/             # interface Clock injetável (determinismo de tempo)
    db/                # conexão SQLite, migrations, sqlc, fx.Module
      migrations/      # *.up.sql / *.down.sql (embarcadas)
      queries/         # *.sql (fonte do sqlc)
      sqlc/            # código GERADO (não editar à mão)
  server/              # glue: cadeia de interceptors + binds cross-domain
  testutil/            # helpers de teste (TestNewDB, TestNewBufconnServer)
test/e2e/              # testes ponta a ponta (bufconn)
docs/                  # ADRs e specs (SDD)
```

### Convenções-chave

- **Idioma (ADR-0005, supersede ADR-0004)**: schema do banco, código Go e contratos proto **todos em inglês** (`national_teams`, `users`, `password_hash` → `NationalTeam`, `User`, `PasswordHash`). **Sem bridge de tradução**: o sqlc gera direto do schema (sem bloco `rename` no `sqlc.yaml`; apenas `overrides` de tipo).
- **Interface no consumidor**: interfaces de dependência são declaradas no pacote que as consome (ex.: `AuthService` declara `UserRepository`); o bind concreto→interface acontece no `fx.Module`.
- **Sem CGO (ADR-0001)**: sempre o driver `sqlite` (modernc), **nunca** `sqlite3`/mattn.
- **Erros como `status.Error(codes.X)`**: o service mapeia erro→código gRPC; o handler é mapper puro (não retraduz).

---

## Regeneração de código

```bash
make proto        # regenera stubs em gen/ a partir de api/proto/
make sqlc         # regenera internal/infra/db/sqlc/ a partir de queries/ + migrations/
```

Rode após alterar `.proto` ou `.sql`. O código gerado é versionado (não é regenerado no build).

---

## Documentação adicional

- **ADRs**: [`docs/adr/INDEX.md`](docs/adr/INDEX.md) — decisões de arquitetura.
- **Specs (SDD)**: [`docs/specs/features/arquitetura-base/v1/`](docs/specs/features/arquitetura-base/v1/) — PRD, tech spec, task plan.
- **Regras de desenvolvimento**: [`CLAUDE.md`](CLAUDE.md) e [`.claude/rules/`](.claude/rules/).
