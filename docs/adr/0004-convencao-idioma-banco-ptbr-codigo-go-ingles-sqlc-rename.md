---
id: 0004
title: Convenção de idioma — schema em pt-BR e código Go/proto em inglês com bridge via sqlc rename
status: superseded
superseded_by: 0005
date: 2026-05-28
tags: [data, cross-cutting, architecture]
---

# 0004 - Convenção de idioma — schema em pt-BR e código Go/proto em inglês com bridge via sqlc rename

> **SUPERSEDIDA pela [ADR-0005](0005-schema-banco-ingles-sem-bridge-idioma.md) em 2026-05-29.** O bloco `rename` do sqlc cresceria a cada termo novo do domínio e exigia sincronização manual; optou-se por padronizar o schema também em inglês, eliminando a ponte de tradução. O conteúdo abaixo é mantido como registro histórico.

## Context

O domínio do projeto (Copa do Mundo 2026) e os stakeholders/DBA operam em pt-BR, então o schema do banco usa nomes em português para legibilidade de negócio. Porém o código Go e os contratos proto seguem a convenção idiomática em inglês. Sem uma ponte explícita, haveria mistura de idiomas no mesmo código ou queries manuais propensas a erro.

## Decision

O schema do banco (tabelas e colunas) é nomeado em português brasileiro; o código Go e os contratos proto usam nomes em inglês idiomático. A ponte entre os dois é feita exclusivamente via a diretiva `rename` do sqlc, sem mapeamento manual.

## Consequences

**Pros:**
- Schema legível para stakeholders/DBA pt-BR do domínio; código Go/proto idiomático em inglês.
- Bridge centralizada e declarativa no sqlc (`rename`) — sem `SELECT ... AS` nem structs manuais.

**Cons:**
- Exige manter o bloco de `rename` do sqlc sincronizado a cada nova coluna/tabela.
- Dois idiomas no projeto: troca de contexto mental entre a camada de dados (pt-BR) e o código (inglês).

**Neutros:**
- Convenção puramente de nomenclatura; não afeta runtime nem performance.

## Alternatives considered

- **Tudo em inglês (banco incluso)** — padronizar o banco também em inglês. Motivo da rejeição: perde legibilidade de negócio para stakeholders/DBA pt-BR do domínio.
- **Tudo em pt-BR (código incluso)** — usar pt-BR também no Go/proto. Motivo da rejeição: foge da convenção idiomática Go e polui APIs/contratos públicos.
- **Mapeamento manual por query** — escrever `SELECT ... AS` / structs manuais. Motivo da rejeição: verboso, repetitivo e propenso a erro; o `rename` do sqlc automatiza a ponte.

## Applied in

-
