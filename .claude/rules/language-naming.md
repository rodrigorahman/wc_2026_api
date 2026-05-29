---
description: Convenção de idioma da WC 2026 API — schema do banco, código Go e contratos proto todos em inglês, sem bridge de tradução. Carregada ao editar código Go, proto ou o sqlc.yaml. Materializa a ADR-0005 (que supersedeu a ADR-0004).
paths:
  - "internal/**"
  - "api/proto/**"
  - "sqlc.yaml"
---

# WC 2026 API — Idioma e Nomenclatura

> ADR-0005 (schema do banco em inglês, sem bridge). Detalhe em [`docs/adr/0005-schema-banco-ingles-sem-bridge-idioma.md`](../../docs/adr/0005-schema-banco-ingles-sem-bridge-idioma.md). Supersede a ADR-0004 (schema pt-BR + rename do sqlc).

## Regra

- **Schema do banco** (tabelas, colunas), **código Go, tipos de domínio e contratos proto** todos em **inglês**: `national_teams`, `users`, `user_national_teams`, `full_name`, `password_hash`, `national_team_id`, `created_at`, `flag_url`.
- **Sem bridge de idioma**: o sqlc gera os tipos diretamente dos nomes em inglês do schema. **Não** há bloco `rename` no `sqlc.yaml` (apenas `overrides` de tipo, ex.: `TIMESTAMP → time.Time`).
- **Dados de domínio** podem permanecer no idioma de origem quando fizer sentido (ex.: nomes de seleções no seed: `'Brasil'`, `'França'`). A regra é sobre **identificadores de schema**, não sobre conteúdo.
- **Mensagens voltadas ao usuário final** podem ser em pt-BR (ex.: `"e-mail ou senha inválidos"`). Identificadores Go, não.

## Naming

- Schema em `snake_case` inglês; tipos/campos Go em `CamelCase` inglês idiomático.
- Initialisms gerados pelo sqlc/protoc-gen-go usam `Url`/`Id` (ex.: `flag_url → FlagUrl`). Onde o estilo idiomático Go importar (`URL`, `ID`), use o nome idiomático no **struct de domínio** e mapeie a partir do tipo gerado no `toDomain`.
- **NÃO** use `SELECT coluna AS alias` nem structs-espelho manuais para "traduzir" idioma — não há idioma a traduzir.
- Ao adicionar coluna, nomeie-a em inglês e regenere (`make sqlc`).

> A mecânica de geração e o driver vivem em [`persistence-sqlite.md`](persistence-sqlite.md).
