---
id: 0005
title: Schema do banco em inglês, sem bridge de idioma
status: accepted
supersedes: 0004
date: 2026-05-29
tags: [data, cross-cutting, architecture]
---

# 0005 - Schema do banco em inglês, sem bridge de idioma

## Context

A ADR-0004 estabeleceu o schema do banco em pt-BR e o código Go/proto em inglês, com a tradução feita exclusivamente pelo bloco `rename` do `sqlc.yaml`. Na prática, esse bloco cresce a cada novo termo distinto do domínio (tabela ou coluna) e precisa ser mantido em sincronia manual — toda coluna nova exige uma entrada para que o tipo gerado fique idiomático em Go. Além disso, termos sem entrada de `rename` vazavam para o código com nomes mistos (ex.: `Nome`, `UsuarioSeleco`), e o singularizador do sqlc produzia identificadores ruins a partir dos nomes pt-BR.

O projeto é greenfield: não há banco em produção nem migrations já aplicadas em ambientes — dev e testes recriam o schema do zero a cada execução. Isso torna viável reescrever as migrations existentes em vez de adicionar migrations de rename.

## Decision

O schema do banco (tabelas e colunas) passa a ser nomeado em **inglês**, igual ao código Go e aos contratos proto. Não há mais bridge de idioma: o bloco `rename` do `sqlc.yaml` é removido e o sqlc gera os tipos diretamente a partir dos nomes em inglês do schema.

- Tabelas/colunas em inglês idiomático: `national_teams`, `users`, `user_national_teams`, `full_name`, `password_hash`, `national_team_id`, `created_at`, `flag_url`, etc.
- As migrations originais (000001-000005) foram reescritas para nascer em inglês (sem migrations de rename), possível por ser greenfield.
- **Dados** de domínio permanecem no idioma de origem quando fizer sentido (ex.: nomes de seleções: `'Brasil'`, `'França'`) — a decisão é sobre **identificadores de schema**, não sobre conteúdo.
- Nomes Go derivados sem `rename`: `flag_url → FlagUrl` (initialism `Url`, não `URL`); structs de domínio podem manter `FlagURL` idiomático e mapear no `toDomain`.

## Consequences

**Pros:**
- `sqlc.yaml` não cresce mais por idioma; zero sincronização manual de tradução.
- Um único idioma no schema e no código — sem troca de contexto mental data↔código.
- Singularização do sqlc produz nomes corretos (`user_national_teams → UserNationalTeam`), eliminando entradas de `rename` corretivas.

**Cons:**
- Perde a legibilidade pt-BR do schema para stakeholders/DBA do domínio (o benefício que motivava a ADR-0004).
- Identificadores gerados usam `Url`/`Id` (initialism do protoc-gen-go/sqlc) em vez de `URL`/`ID` nos tipos gerados; structs de domínio compensam onde o estilo idiomático importa.

**Neutros:**
- Convenção de nomenclatura; não afeta runtime nem performance.

## Alternatives considered

- **Manter o `rename` do sqlc (ADR-0004)** — bridge centralizada e schema legível em pt-BR. Rejeitada: o bloco cresce a cada termo novo e exige manutenção manual contínua; termos sem entrada vazam nomes mistos.
- **`SELECT ... AS alias` por query** — mover a tradução para os `.sql`. Rejeitada: duplica o alias em cada query que toca a coluna, não controla o initialism do tipo gerado e era explicitamente o anti-padrão rejeitado pela ADR-0004.
- **Migrations aditivas de rename (manter 000001-000005 em pt-BR + ALTER RENAME)** — preservaria histórico. Rejeitada: desnecessária num projeto greenfield (sem produção) e deixaria o schema meio pt-BR meio inglês, com `ALTER TABLE RENAME` frágil sob FKs no SQLite.

## Applied in

- `internal/db/migrations/000001-000005` (reescritas em inglês)
- `internal/db/queries/national_teams.sql`, `internal/db/queries/users.sql`
- `sqlc.yaml` (bloco `rename` removido)
- `internal/db/sqlc/**` (regenerado)
