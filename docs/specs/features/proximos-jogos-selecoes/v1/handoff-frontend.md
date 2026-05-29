# Backend Contract Handoff — Próximos Jogos das Seleções

> Gerado em: 2026-05-29
> Fonte primária: protobuf (`api/proto/wc2026/match/v1/match.proto`) + handler/service do domínio match
> Referências: scope.md, intent.md (proximos-jogos-selecoes/v1); ADR-0003 (anti-IDOR / corte por clock)

---

## 1. Feature

Card da home que lista os **próximos jogos** das seleções favoritas do usuário autenticado, em ordem cronológica. As favoritas são inferidas do token (o usuário não envia filtro).

## 2. Scope

- **Entra**: `MatchService.ListUpcomingMatches` (1 RPC protegido).
- **Não entra**: cadastro/edição de partidas (não há RPC de escrita); placar/status ao vivo; paginação; escolha de favoritas (ver `usuario-multiplas-selecoes`).
- **Versão da API**: `wc2026.match.v1` (gRPC).

## 3. Transporte e Auth

- **Protocolo**: gRPC sobre HTTP/2, porta default `:50051`. <!-- fonte: cmd/server/main.go:41 -->
- **Auth**: **obrigatória** — Bearer JWT no metadata `authorization`. RPC **protegido** (não está na allowlist pública). <!-- fonte: internal/server/server.go:85-90 (ausente da lista) -->
- **Anti-IDOR**: o `user_id` usado para filtrar favoritas vem **exclusivamente** do `sub` do token; o request é vazio. Frontend não envia (nem consegue enviar) id de usuário. <!-- fonte: internal/domain/match/handler/match_handler.go:47-52 -->
- **Formato de erro**: gRPC status (code + message pt-BR).

## 4. Backend Entry Points

| # | Operação | Transporte | Método/Ação | Path/Nome | Auth |
|---|---|---|---|---|---|
| 1 | Próximos jogos das favoritas | gRPC | RPC | `wc2026.match.v1.MatchService/ListUpcomingMatches` | Bearer JWT |

---

## 4. Contracts

### 4.1 ListUpcomingMatches

- **Tipo**: gRPC RPC · **Auth**: obrigatória (Bearer JWT) · **Permissões**: usuário autenticado; resultado escopado às favoritas do próprio usuário (sub do token) · **Idempotência**: sim (leitura) · **Cache**: curta — o conjunto muda com o tempo (corte = agora) e com as favoritas. `[HIPÓTESE]` TTL curto (ex. 60s) é seguro; sem header de cache no backend.

**Request** — `ListUpcomingMatchesRequest` (vazio). <!-- fonte: api/proto/wc2026/match/v1/match.proto:13 -->

**Response (OK)** — `ListUpcomingMatchesResponse` <!-- fonte: match.proto:15-34; handler match_handler.go:57-80 -->
```json
{
  "matches": [
    {
      "id": "uuid",
      "kickoff": "RFC3339 timestamp (UTC)",
      "home_team": { "id": "uuid", "name": "string", "flag_url": "https url", "code": "FIFA 3-letras" },
      "away_team": { "id": "uuid", "name": "string", "flag_url": "https url", "code": "FIFA 3-letras" },
      "stadium": "string",
      "city": "string",
      "stage": "string"
    }
  ]
}
```

- **Ordenação**: `kickoff` ascendente (cronológica). <!-- fonte: internal/infra/db/queries/matches.sql (ORDER BY m.kickoff ASC) -->
- **Filtro**: apenas jogos **futuros** (`kickoff > agora`, corte estritamente maior). <!-- fonte: query ListUpcomingMatchesByUser; service usa clock.Now() -->
- **Dedup**: uma partida entre **duas** seleções favoritas aparece **uma única vez**. <!-- fonte: scope.md §3.2; T3 dedup test -->
- **Vazia**: usuário sem favoritas, ou favoritas sem jogos futuros → `matches: []` (lista vazia, **não** é erro). <!-- fonte: T3/T7 tests -->

**Erros possíveis** <!-- fonte: match_handler.go:47-55 -->

| gRPC code | Quando ocorre | message |
|---|---|---|
| `Unauthenticated` (16) | token ausente/inválido/expirado (barrado no interceptor) ou sub ausente no handler | `não autenticado` / erro do interceptor |
| `Internal` (13) | qualquer falha do service (ex. erro de banco) — handler colapsa em Internal genérico | `erro interno` |

**Observações**: o handler é mapper puro — **não** repassa códigos específicos do service; qualquer erro de negócio/infra vira `Internal`. `code` é a sigla FIFA (ex. `BRA`, `KOR`) para exibição compacta no card.

---

## 5. UI States Required

| Operação | loading | success | empty | unauthorized | unexpected_error |
|---|---|---|---|---|---|
| ListUpcomingMatches | ✓ | ✓ | ✓ | ✓ | ✓ |

> Sem `validation_error` (request vazio), sem `not_found` (listagem), sem `forbidden` (sem RBAC).

## 6. Error Mapping

| Operação | Erro backend | Estado UI | Mensagem (i18n) | Retry | Invalida cache |
|---|---|---|---|---|---|
| ListUpcomingMatches | `OK` + `matches: []` | `empty` (card "sem jogos das suas seleções") | `matches.empty` | — | — |
| ListUpcomingMatches | `Unauthenticated` | limpar token + redirect login | — | — | sim |
| ListUpcomingMatches | `Internal` | `unexpected_error` + telemetria | `errors.unexpected` | sim (backoff) | não |

## 7. Fixtures

```json
{
  "name": "list-upcoming-matches/success",
  "request": { "rpc": "MatchService/ListUpcomingMatches", "metadata": { "authorization": "Bearer <jwt>" }, "body": {} },
  "response": {
    "code": "OK",
    "body": {
      "matches": [
        {
          "id": "f0a1...-001",
          "kickoff": "2026-06-12T20:00:00Z",
          "home_team": { "id": "a1f3c5e7-0001-4000-8000-000000000001", "name": "Brasil", "flag_url": "https://flagcdn.com/w320/br.png", "code": "BRA" },
          "away_team": { "id": "a1f3c5e7-0002-4000-8000-000000000002", "name": "Argentina", "flag_url": "https://flagcdn.com/w320/ar.png", "code": "ARG" },
          "stadium": "Maracanã", "city": "Rio de Janeiro", "stage": "Fase de Grupos"
        }
      ]
    }
  }
}
```
```json
{ "name": "list-upcoming-matches/empty", "response": { "code": "OK", "body": { "matches": [] } } }
```
```json
{ "name": "list-upcoming-matches/unauthorized", "response": { "code": "Unauthenticated", "message": "não autenticado" } }
```

## 8. Frontend Implementation Notes

- **Ordenação já vem pronta** (kickoff ASC) — não reordenar no cliente.
- **Card**: usar `code` (sigla FIFA) + `flag_url` para layout compacto; `kickoff` é UTC — converter ao fuso do usuário.
- **Empty ≠ erro**: tratar `matches: []` como estado vazio válido (sem favoritas ou sem jogos futuros).
- **Sem paginação**: a lista vem completa (dataset pequeno). Não implementar scroll infinito.
- **Cache**: TTL curto; invalidar ao mudar favoritas (quando essa feature existir) e ao relogar.

## 9. Acceptance Criteria

- [ ] Card mostra `loading` enquanto a request está pendente.
- [ ] Jogos renderizam em ordem cronológica (sem reordenar no cliente).
- [ ] Lista vazia renderiza estado `empty` (não erro).
- [ ] Partida com duas favoritas aparece uma única vez (sem duplicar).
- [ ] `kickoff` exibido no fuso local do usuário.
- [ ] `Unauthenticated` limpa token e redireciona para login.

## 10. Minimum Tests

| # | Tipo | Comportamento |
|---|---|---|
| 1 | Component | Renderiza skeleton durante request pendente |
| 2 | Component | `matches: []` renderiza estado vazio, não erro |
| 3 | Component | Lista mantém ordem cronológica recebida |
| 4 | Integration | Sem token → não chama / trata `Unauthenticated` com redirect |
| 5 | Integration | `Internal` mostra erro genérico com opção de retry |

## 11. Open Questions

- [ ] `[DÚVIDA]` Proxy grpc-web/gateway para web? Backend expõe só gRPC puro.
- [ ] `[HIPÓTESE]` TTL de cache curto (~60s) aceitável — não há header de cache no backend; confirmar com produto a frescor desejada.
- [ ] `[HIPÓTESE]` `stage` é texto livre (ex. "Fase de Grupos", "Oitavas") — confirmar enum/valores canônicos se a UI precisar de tradução/ícone por fase.
