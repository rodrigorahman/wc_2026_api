# Backend Contract Handoff — URL da Bandeira da Seleção

> Gerado em: 2026-05-29
> Fonte primária: protobuf (`api/proto/wc2026/nationalteam/v1/national_team.proto`) + handler do domínio nationalteam
> Referências: taskcard `task-01-adicionar-bandeira-url-selecao.md`; ADR-0005 (schema/proto em inglês)

> **Delta handoff**: esta feature apenas **adiciona um campo** ao contrato existente de `ListNationalTeams`. O contrato completo do RPC vive em [`../../arquitetura-base/v1/handoff-frontend.md`](../../arquitetura-base/v1/handoff-frontend.md) §4.4.

---

## 1. Feature

Cada seleção passa a expor a **URL da bandeira** (`flag_url`), para o frontend renderizar o escudo/bandeira em qualquer lugar que liste seleções (cadastro, card de jogos, perfil).

## 2. Scope

- **Entra**: novo campo `flag_url` na mensagem `NationalTeam` retornada por `ListNationalTeams`.
- **Não entra**: RPC de escrita/edição de seleção (não existe); validação de acessibilidade da URL; upload de imagem.

## 3. Mudança de Contrato

Operação afetada: `wc2026.nationalteam.v1.NationalTeamService/ListNationalTeams` (pública, sem auth).

**Antes** → **Depois** (campo `flag_url = 3` adicionado) <!-- fonte: api/proto/wc2026/nationalteam/v1/national_team.proto:17-21 -->
```json
{
  "national_teams": [
    {
      "id": "uuid",
      "name": "string",
      "flag_url": "https url"        // ← NOVO (campo 3)
    }
  ]
}
```

- `flag_url` é **sempre preenchida** para as 16 seleções do seed (backfill na migração). Padrão: `https://flagcdn.com/w320/{codigo}.png`. <!-- fonte: taskcard §5 (tabela de códigos); migration 000005/000006 -->
- Compatibilidade: adição de campo proto3 — clientes antigos ignoram; clientes novos leem. Sem quebra. Mesma versão `v1`.
- `GetNationalTeamByID` (uso interno) **não** expõe `flag_url` — irrelevante ao frontend (não há RPC). <!-- fonte: taskcard §6.1 -->

**Mapeamento de bandeira (16 seleções do seed)** <!-- fonte: taskcard task-01 §5, tabela -->

| Seleção | code flagcdn | Seleção | code flagcdn |
|---|---|---|---|
| Brasil | `br` | Bélgica | `be` |
| Argentina | `ar` | Croácia | `hr` |
| França | `fr` | México | `mx` |
| Alemanha | `de` | Estados Unidos | `us` |
| Espanha | `es` | Japão | `jp` |
| Inglaterra | `gb-eng` | Coreia do Sul | `kr` |
| Portugal | `pt` | Itália | `it` |
| Países Baixos | `nl` | Uruguai | `uy` |

## 4. UI States / Error Mapping

Sem novos estados nem novos erros — herda `ListNationalTeams` (ver arquitetura-base §5/§6). `flag_url` é dado adicional na resposta `success`.

## 5. Fixtures

```json
{
  "name": "list-national-teams/success-with-flag",
  "response": { "code": "OK", "body": { "national_teams": [
    { "id": "a1f3c5e7-0001-4000-8000-000000000001", "name": "Brasil", "flag_url": "https://flagcdn.com/w320/br.png" },
    { "id": "a1f3c5e7-0006-4000-8000-000000000006", "name": "Inglaterra", "flag_url": "https://flagcdn.com/w320/gb-eng.png" },
    { "id": "a1f3c5e7-0016-4000-8000-000000000016", "name": "Coreia do Sul", "flag_url": "https://flagcdn.com/w320/kr.png" }
  ] } }
}
```

## 6. Frontend Implementation Notes

- Renderizar `<img src={flag_url}>` (origem externa `flagcdn.com`) — incluir `flagcdn.com` na CSP `img-src` se houver CSP. `[DÚVIDA]` CSP é responsabilidade do frontend; backend não define.
- `flag_url` nunca vem vazia para o seed; ainda assim, ter fallback de imagem é boa prática.
- Tamanho servido é `w320` (largura 320px) — adequado para thumbnails; para telas maiores avaliar outra largura do flagcdn no cliente.

## 7. Acceptance Criteria

- [ ] Cada item de `ListNationalTeams` renderiza a bandeira a partir de `flag_url`.
- [ ] `flag_url` ausente/erro de imagem cai em fallback visual (não quebra o layout).
- [ ] `flagcdn.com` permitido na CSP de imagens (se aplicável).

## 8. Minimum Tests

| # | Tipo | Comportamento |
|---|---|---|
| 1 | Component | Item de seleção renderiza `<img>` com `flag_url` recebida |
| 2 | Component | Falha de carregamento da imagem mostra fallback |

## 9. Open Questions

- [ ] `[HIPÓTESE]` `flag_url` sempre presente e válida (backfill garante para o seed). Confirmar comportamento caso futuramente exista seleção criada sem bandeira.
