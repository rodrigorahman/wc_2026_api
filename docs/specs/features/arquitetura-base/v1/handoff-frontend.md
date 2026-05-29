# Backend Contract Handoff — Arquitetura Base (Auth + Seleções)

> Gerado em: 2026-05-29
> Fonte primária: protobuf (`api/proto/wc2026/**`) + código backend (handlers/services)
> Referências: tech_spec.md (arquitetura-base/v1); ADR-0002 (uber-fx), ADR-0003 (JWT HS256/bcrypt/TTL 1h)

---

## 1. Feature

Fundação da API: cadastro de usuário, autenticação (login/JWT), dados do usuário autenticado e listagem das seleções da Copa. É o contrato base sobre o qual as demais features acrescentam campos/RPCs.

## 2. Scope

- **Entra**: `AuthService` (Register, Login, GetMe) e `NationalTeamService` (ListNationalTeams).
- **Não entra**: edição/exclusão de usuário; refresh token; escrita de seleções; partidas (ver handoff `proximos-jogos-selecoes`).
- **Versão da API**: `wc2026.*.v1` (gRPC).

## 3. Transporte e Auth (comum a todas as operações)

- **Protocolo**: gRPC sobre HTTP/2. Porta default `:50051`. <!-- fonte: cmd/server/main.go:41; internal/infra/config/config.go:59 -->
- `[DÚVIDA]` Não há grpc-web/gateway HTTP configurado no backend. Frontend **web/browser** precisará de um proxy (Envoy/grpc-web) ou gateway REST — confirmar com infra. Clientes nativos (mobile/Go/Node gRPC) conectam direto.
- **Reflection**: habilitada apenas quando `APP_ENV` é dev. <!-- fonte: internal/server/server.go:156 -->
- **Auth**: JWT HS256 no metadata gRPC `authorization: Bearer <access_token>`. TTL 1h, **sem refresh** (relogar ao expirar). <!-- fonte: .claude/rules/auth-security.md; ADR-0003 -->
- **RPCs públicos** (sem token): `Register`, `Login`, `ListNationalTeams`. **Protegidos**: `GetMe`. <!-- fonte: internal/server/server.go:85-90 -->
- **Validação de entrada**: declarativa via protovalidate. Violação de formato/obrigatoriedade → `InvalidArgument` (code 3) **antes** do handler. <!-- fonte: .claude/rules/grpc-layers.md -->
- **Formato de erro**: gRPC status (`code` numérico + `message` em pt-BR). **Não há** envelope JSON de erro.

## 4. Backend Entry Points

| # | Operação | Transporte | Método/Ação | Path/Nome | Auth |
|---|---|---|---|---|---|
| 1 | Cadastrar usuário | gRPC | RPC | `wc2026.auth.v1.AuthService/Register` | pública |
| 2 | Login | gRPC | RPC | `wc2026.auth.v1.AuthService/Login` | pública |
| 3 | Dados do usuário logado | gRPC | RPC | `wc2026.auth.v1.AuthService/GetMe` | Bearer JWT |
| 4 | Listar seleções | gRPC | RPC | `wc2026.nationalteam.v1.NationalTeamService/ListNationalTeams` | pública |

---

## 4. Contracts

### 4.1 Register

- **Tipo**: gRPC RPC · **Auth**: nenhuma · **Permissões**: pública · **Idempotência**: não (cada chamada tenta criar; e-mail repetido → `AlreadyExists`) · **Cache**: não

**Request** — `RegisterRequest` <!-- fonte: api/proto/wc2026/auth/v1/auth.proto:16 -->
```json
{
  "full_name": "string (min 1)",
  "email": "string (formato e-mail)",
  "password": "string (min 8)",
  "national_team_ids": ["uuid (1 a 3 itens, sem duplicatas)"]
}
```

**Response (OK)** — `RegisterResponse`
```json
{ "user_id": "uuid" }
```

**Erros possíveis** <!-- fonte: internal/domain/auth/service/auth_service.go:158-202 -->

| gRPC code | Quando ocorre | message |
|---|---|---|
| `InvalidArgument` (3) | protovalidate: full_name vazio, e-mail inválido, senha < 8, lista vazia/>3/UUID inválido | (detalhe do protovalidate) |
| `InvalidArgument` (3) | id de seleção inexistente | `seleção favorita inválida` |
| `InvalidArgument` (3) | id de seleção duplicado na lista | `seleção favorita duplicada` |
| `AlreadyExists` (6) | e-mail já cadastrado | `e-mail já está em uso` |
| `Internal` (13) | falha de persistência/hash | `erro interno` |

**Side effects**: cria usuário + vínculos de seleção (atômico). Senha é hasheada com bcrypt; **nunca** retornada/logada. <!-- fonte: auth_service.go:183-199 -->

### 4.2 Login

- **Tipo**: gRPC RPC · **Auth**: nenhuma · **Permissões**: pública · **Idempotência**: sim (consulta) · **Cache**: não

**Request** — `LoginRequest`
```json
{ "email": "string (formato e-mail)", "password": "string (min 1)" }
```

**Response (OK)** — `LoginResponse` <!-- fonte: api/proto/wc2026/auth/v1/auth.proto:44 -->
```json
{ "access_token": "jwt string", "expires_at": "RFC3339 timestamp (UTC)" }
```

**Erros possíveis** <!-- fonte: auth_service.go:223-245 -->

| gRPC code | Quando ocorre | message |
|---|---|---|
| `InvalidArgument` (3) | protovalidate (e-mail inválido / senha vazia) | (detalhe do protovalidate) |
| `Unauthenticated` (16) | e-mail inexistente **ou** senha errada (mensagem idêntica — anti-enumeração) | `e-mail ou senha inválidos` |
| `Internal` (13) | falha ao gerar token | `erro interno` |

**Observações**: a resposta não diferencia "e-mail não existe" de "senha errada" (proposital). Guarde `access_token` + `expires_at`; ao expirar (sem refresh) → refazer Login. <!-- fonte: auth_service.go:226-231 -->

### 4.3 GetMe

- **Tipo**: gRPC RPC · **Auth**: **obrigatória** (Bearer JWT) · **Permissões**: o usuário só lê os próprios dados — id vem do `sub` do token, nunca de input (anti-IDOR) · **Cache**: curta, invalidar no logout

**Request** — `GetMeRequest` (vazio) — identidade vem do token. <!-- fonte: api/proto/wc2026/auth/v1/auth.proto:49 -->

**Response (OK)** — `GetMeResponse`
```json
{
  "user_id": "uuid",
  "full_name": "string",
  "email": "string",
  "national_team_ids": ["uuid"]
}
```

**Erros possíveis** <!-- fonte: internal/domain/auth/handler/auth_handler.go:78-87; auth_service.go:207-217 -->

| gRPC code | Quando ocorre | message |
|---|---|---|
| `Unauthenticated` (16) | token ausente, inválido, expirado ou alg incorreto (barrado no interceptor) | `não autenticado` / erro do interceptor |
| `NotFound` (5) | `sub` do token sem usuário correspondente | `usuário não encontrado` |
| `Internal` (13) | falha de leitura | `erro interno` |

### 4.4 ListNationalTeams

- **Tipo**: gRPC RPC · **Auth**: nenhuma · **Permissões**: pública · **Cache**: alta (dataset fixo — 16 seleções do seed); pode cachear agressivamente. `[HIPÓTESE]` Lista é estável (sem RPC de escrita).

**Request** — `ListNationalTeamsRequest` (vazio).

**Response (OK)** — `ListNationalTeamsResponse` <!-- fonte: api/proto/wc2026/nationalteam/v1/national_team.proto:13; handler national_team_handler.go:43 -->
```json
{
  "national_teams": [
    { "id": "uuid", "name": "string", "flag_url": "https url" }
  ]
}
```

**Erros possíveis**

| gRPC code | Quando ocorre | message |
|---|---|---|
| `Internal` (13) | falha de leitura | `erro interno` |

**Observações**: retorna 16 seleções (seed). Nunca vazia em operação normal; trate `empty` defensivamente. `flag_url` documentado no handoff `national-team-flag-url`.

---

## 5. UI States Required

| Operação | loading | success | empty | validation_error | unauthorized | not_found | conflict | unexpected_error |
|---|---|---|---|---|---|---|---|---|
| Register | ✓ | ✓ | — | ✓ | — | — | ✓ (e-mail/seleção) | ✓ |
| Login | ✓ | ✓ | — | ✓ | ✓ (credencial) | — | — | ✓ |
| GetMe | ✓ | ✓ | — | — | ✓ | ✓ | — | ✓ |
| ListNationalTeams | ✓ | ✓ | ✓ | — | — | — | — | ✓ |

> gRPC não tem 403/forbidden aqui (sem RBAC); `unauthorized` = `Unauthenticated`.

## 6. Error Mapping

| Operação | Erro backend | Estado UI | Mensagem (i18n) | Retry | Invalida cache |
|---|---|---|---|---|---|
| Register | `InvalidArgument` | inline por campo | `errors.register.{campo}` | não | não |
| Register | `AlreadyExists` | inline no campo e-mail | `errors.register.email_taken` | não | não |
| Login | `Unauthenticated` | erro genérico no form | `errors.login.invalid_credentials` | sim (nova tentativa) | não |
| GetMe / protegido | `Unauthenticated` | limpar token + redirect login | — | — | sim (limpar tudo) |
| GetMe | `NotFound` | logout (sessão órfã) | `errors.session.invalid` | não | sim |
| qualquer | `Internal` | `unexpected_error` + telemetria | `errors.unexpected` | sim (backoff) | não |

## 7. Fixtures

```json
{ "name": "register/success", "request": { "rpc": "AuthService/Register", "body": { "full_name": "Maria Silva", "email": "maria@example.com", "password": "senha12345", "national_team_ids": ["a1f3c5e7-0001-4000-8000-000000000001"] } }, "response": { "code": "OK", "body": { "user_id": "9b2e..." } } }
```
```json
{ "name": "register/email-taken", "response": { "code": "AlreadyExists", "message": "e-mail já está em uso" } }
```
```json
{ "name": "login/success", "response": { "code": "OK", "body": { "access_token": "eyJ...", "expires_at": "2026-05-29T13:00:00Z" } } }
```
```json
{ "name": "login/invalid", "response": { "code": "Unauthenticated", "message": "e-mail ou senha inválidos" } }
```
```json
{ "name": "getme/success", "response": { "code": "OK", "body": { "user_id": "9b2e...", "full_name": "Maria Silva", "email": "maria@example.com", "national_team_ids": ["a1f3c5e7-0001-4000-8000-000000000001"] } } }
```
```json
{ "name": "list-national-teams/success", "response": { "code": "OK", "body": { "national_teams": [ { "id": "a1f3c5e7-0001-4000-8000-000000000001", "name": "Brasil", "flag_url": "https://flagcdn.com/w320/br.png" } ] } } }
```

## 8. Frontend Implementation Notes

- **Token**: armazenar `access_token` + `expires_at`; anexar `authorization: Bearer` em todo RPC protegido. Sem refresh → ao detectar `expires_at` passado ou `Unauthenticated`, forçar relogin.
- **Anti-enumeração**: não tente inferir existência de e-mail pela resposta do Login — é proposital.
- **Seleções no cadastro**: 1 a 3 ids, sem duplicatas (ver handoff `usuario-multiplas-selecoes`). Popular o seletor via `ListNationalTeams`.
- **ListNationalTeams**: cachear (dataset fixo). Usar `id` como valor e `flag_url`/`name` para render.

## 9. Acceptance Criteria

- [ ] Form de cadastro valida localmente full_name (≥1), e-mail, senha (≥8), seleções (1–3, sem duplicata) espelhando o protovalidate.
- [ ] `AlreadyExists` no Register aparece inline no campo e-mail.
- [ ] Login com credencial errada mostra mensagem genérica (sem revelar e-mail).
- [ ] `access_token` é anexado em GetMe; ausência/expiração → redirect login.
- [ ] `Unauthenticated` em qualquer protegido limpa o token e redireciona.
- [ ] Seletor de seleções é populado por `ListNationalTeams` e exibe a bandeira.

## 10. Minimum Tests

| # | Tipo | Comportamento |
|---|---|---|
| 1 | Integration | Register com payload válido → guarda user_id; e-mail repetido → erro inline |
| 2 | Integration | Login sucesso guarda token+expiry; credencial inválida → mensagem genérica |
| 3 | Integration | GetMe sem token → redirect login; com token → renderiza dados |
| 4 | Component | Lista de seleções renderiza 16 itens com bandeira; estado loading/empty |

## 11. Open Questions

- [ ] `[DÚVIDA]` Há proxy grpc-web/gateway REST para o frontend web, ou o cliente é nativo gRPC? Backend só expõe gRPC puro em `:50051`.
- [ ] `[DÚVIDA]` Política de armazenamento do token no web (cookie httpOnly vs storage)? Não definida no backend.
- [ ] `[HIPÓTESE]` `ListNationalTeams` é estável (16 seleções, sem escrita) — seguro cachear longo. Confirmar se haverá atualização de seleções.
