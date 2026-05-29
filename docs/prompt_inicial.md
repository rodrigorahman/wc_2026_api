# Prompt Inicial — WC 2026 API

> Prompt único consolidado. Reúne, num só ponto de partida, tudo que originalmente
> foi entregue em 3 ciclos (1 SDD de arquitetura-base + 2 TaskCards: `flag_url` e
> `usuário com múltiplas seleções`). Digite/cole este prompt para gerar a base já no
> estado final, sem precisar repetir as 3 alterações.

---

Vamos iniciar a construção da API do projeto de **Álbum da Copa do Mundo 2026**, onde o usuário poderá cadastrar as figurinhas de sua coleção e organizar as repetidas para possíveis trocas.

## Requisitos de stack
- **gRPC** como protocolo de comunicação entre frontend e backend.
- **SQLite** como banco de dados.
- **buf.validate / protovalidate** para validação declarativa de dados (sem validação manual no handler).
- Autenticação com **golang-jwt** (HS256) e middleware **go-grpc-middleware/auth**.
- **viper** para configuração via env.
- **zap** para logging (token/senha **nunca** logados).
- **testify** para testes.
- **sqlc** para queries tipadas.
- **golang-migrate** para migrations.
- **uber-fx** para Dependency Injection (wiring em `fx.Module` por domínio).
- **protoc-gen-go** e **protoc-gen-go-grpc** (via buf) para os stubs gRPC.
- Compilável e executável em **Windows, Linux e macOS**, com **CGO desabilitado** (`CGO_ENABLED=0`) — driver SQLite pure-Go (`modernc.org/sqlite`).
- IDs de todas as tabelas em **UUID v4**.

## Estrutura de pastas (go-standard-layout)
- `.proto` em **`api/proto/wc2026/<dominio>/v1/`**.
- Código gerado (proto/gRPC) em **`gen/`** na raiz (regenerado por `make proto`).
- Código da aplicação em **`internal/`**, separando domínio de infraestrutura:
  - **`internal/domain/<feature>/`** — cada feature com camadas `handler → service → repository`.
  - **`internal/infra/`** — `config`, `logger`, `clock`, `db` (migrations + queries + sqlc).
  - **`internal/server`** — composition root (cadeia de interceptors montada uma única vez, reusada por produção e E2E).
  - **`internal/testutil`** — helpers de teste (SQLite efêmero, bufconn).
- **Idioma**: schema do banco, código Go e contratos proto **em inglês**, sem bridge de tradução (sqlc gera direto do schema). Dados de domínio (ex.: nomes de seleções) e mensagens ao usuário podem ser pt-BR.

## Domínio: Seleção (NationalTeam)
Campos:
- `id` (UUID v4)
- `name` (nome da seleção, ex.: "Brasil")
- `flag_url` (URL completa da bandeira, ex.: `https://flagcdn.com/w320/br.png`)

Popular via seed as 16 seleções participantes já com suas respectivas URLs de bandeira.

API:
- `ListNationalTeams` → lista pública de seleções com `id`, `name` e `flag_url`.

## Domínio: Autenticação / Usuário
O usuário possui:
- Nome completo (`full_name`)
- E-mail (`email`, único)
- Senha (`password`, armazenada com **bcrypt** — nunca em texto plano)
- **Uma ou mais seleções** que acompanha — relação **N:N** (um usuário pode ter várias seleções; uma seleção pode ser acompanhada por vários usuários).

API:
- `Register(full_name, email, password, national_team_ids[])` → cria o usuário associado a uma ou mais seleções.
- `Login(email, password)` → retorna `access_token` (JWT).
- `GetMe()` (protegido por JWT) → retorna `user_id`, `full_name`, `email` e a lista de `national_team_ids` do usuário autenticado (lendo a identidade do contexto, nunca de input do cliente).
