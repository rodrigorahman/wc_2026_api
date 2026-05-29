# Backend Contract Handoff — Usuário com Múltiplas Seleções

> Gerado em: 2026-05-29
> Fonte primária: protobuf (`api/proto/wc2026/auth/v1/auth.proto`) + `AuthService` (Register/GetMe)
> Referências: taskcard `task-01-migrar-selecoes-usuario-nn.md`; ADR-0002

> **Delta handoff**: altera a cardinalidade de seleções favoritas de **1 → N (1 a 3)** em `Register` e `GetMe`. Contrato completo desses RPCs em [`../../arquitetura-base/v1/handoff-frontend.md`](../../arquitetura-base/v1/handoff-frontend.md) §4.1 e §4.3.

---

## 1. Feature

No cadastro, o usuário escolhe **de 1 a 3 seleções favoritas** (antes era exatamente 1). `GetMe` passa a devolver a lista de favoritas. É o que alimenta o filtro de "próximos jogos".

## 2. Scope

- **Entra**: campo `national_team_ids` (lista) em `RegisterRequest` e `GetMeResponse`.
- **Não entra**: RPC para **editar** as seleções de um usuário já cadastrado (só cadastro + leitura via GetMe); `Login`/JWT inalterados.

## 3. Mudança de Contrato

### 3.1 Register — `wc2026.auth.v1.AuthService/Register` (pública)

`national_team_id` (singular) → `national_team_ids` (lista). <!-- fonte: api/proto/wc2026/auth/v1/auth.proto:24-30 -->

**Request** — `RegisterRequest`
```json
{
  "full_name": "string (min 1)",
  "email": "string (formato e-mail)",
  "password": "string (min 8)",
  "national_team_ids": ["uuid"]   // ← LISTA: min 1, max 3 itens, cada item UUID
}
```

**Regras de validação** (camadas distintas — importante para mapear erros): <!-- fonte: taskcard §11; auth_service.go:159-172; auth.proto:24-30 -->

| Regra | Onde valida | Erro |
|---|---|---|
| 1 a 3 itens; cada item UUID sintático | protovalidate (contrato) | `InvalidArgument` (3) |
| cada id deve **existir** | service | `InvalidArgument` (3) — `seleção favorita inválida` |
| **sem duplicatas** na lista | service | `InvalidArgument` (3) — `seleção favorita duplicada` |

> Persistência é **atômica**: se qualquer seleção falhar, o usuário **não** é criado.

### 3.2 GetMe — `wc2026.auth.v1.AuthService/GetMe` (Bearer JWT)

`GetMeResponse.national_team_ids` agora é **lista** com as favoritas escolhidas no cadastro. <!-- fonte: auth.proto:51-56; auth_handler.go:89-94 -->
```json
{
  "user_id": "uuid",
  "full_name": "string",
  "email": "string",
  "national_team_ids": ["uuid"]   // ← LISTA (1 a 3)
}
```

## 4. UI States / Error Mapping (delta)

| Operação | Erro backend | Estado UI | Mensagem (i18n) |
|---|---|---|---|
| Register | `InvalidArgument` (protovalidate: <1 ou >3, UUID inválido) | `validation_error` no seletor de seleções | `errors.register.teams.cardinality` |
| Register | `InvalidArgument` (`seleção favorita inválida`) | `validation_error` no item inexistente | `errors.register.teams.unknown` |
| Register | `InvalidArgument` (`seleção favorita duplicada`) | `validation_error` (remover duplicata) | `errors.register.teams.duplicate` |

Demais erros de Register/GetMe: ver arquitetura-base §6.

## 5. Fixtures

```json
{ "name": "register/3-teams-success", "request": { "rpc": "AuthService/Register", "body": { "full_name": "João", "email": "joao@example.com", "password": "senha12345", "national_team_ids": ["a1f3c5e7-0001-4000-8000-000000000001","a1f3c5e7-0002-4000-8000-000000000002","a1f3c5e7-0003-4000-8000-000000000003"] } }, "response": { "code": "OK", "body": { "user_id": "..." } } }
```
```json
{ "name": "register/empty-teams-rejected", "request": { "body": { "national_team_ids": [] } }, "response": { "code": "InvalidArgument", "message": "(protovalidate: min_items 1)" } }
```
```json
{ "name": "register/too-many-teams-rejected", "request": { "body": { "national_team_ids": ["...1","...2","...3","...4"] } }, "response": { "code": "InvalidArgument", "message": "(protovalidate: max_items 3)" } }
```
```json
{ "name": "register/duplicate-team-rejected", "request": { "body": { "national_team_ids": ["a1f3c5e7-0001-...","a1f3c5e7-0001-..."] } }, "response": { "code": "InvalidArgument", "message": "seleção favorita duplicada" } }
```
```json
{ "name": "getme/multiple-teams", "response": { "code": "OK", "body": { "user_id": "...", "full_name": "João", "email": "joao@example.com", "national_team_ids": ["a1f3c5e7-0001-...","a1f3c5e7-0002-..."] } } }
```

## 6. Frontend Implementation Notes

- **Seletor multi**: 1 a 3 seleções, **sem repetição**. Replicar as 3 regras no cliente para feedback imediato, mas tratar o erro do backend como fonte de verdade.
- Popular opções via `ListNationalTeams` (id + name + flag_url).
- `national_team_ids` é sempre lista (mesmo com 1 item) — não tratar como escalar.
- Não há edição pós-cadastro: a UI de perfil exibe (GetMe) mas não altera favoritas nesta versão.

## 7. Acceptance Criteria

- [ ] Seletor permite escolher de 1 a 3 seleções e bloqueia a 4ª.
- [ ] Seletor impede selecionar a mesma seleção duas vezes (e o backend reforça: `seleção favorita duplicada`).
- [ ] Cadastro sem nenhuma seleção é bloqueado (cliente + `InvalidArgument`).
- [ ] `GetMe` renderiza a lista completa de favoritas (1 a 3).

## 8. Minimum Tests

| # | Tipo | Comportamento |
|---|---|---|
| 1 | Component | Seletor limita seleção a no máximo 3 e mínimo 1 |
| 2 | Component | Tentativa de duplicar seleção é impedida na UI |
| 3 | Integration | Register com 4 seleções → `InvalidArgument` tratado no seletor |
| 4 | Integration | GetMe renderiza N favoritas (lista, não escalar) |

## 9. Open Questions

- [ ] `[DÚVIDA]` Haverá RPC futuro para **editar** favoritas? Hoje só cadastro + leitura — a UI deve esconder/ desabilitar edição.
- [ ] `[HIPÓTESE]` Limite 1–3 é fixo no contrato (protovalidate). Mudança exigiria nova versão do proto.
