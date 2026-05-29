---
id: 0001
title: Uso do driver modernc.org/sqlite (pure-Go) para portabilidade cross-platform
status: accepted
date: 2026-05-27
tags: [data, build, architecture]
---

# 0001 - Uso do driver modernc.org/sqlite (pure-Go) para portabilidade cross-platform

## Context

A API precisa de SQLite embarcado e roda/compila em múltiplos SOs (macOS, Linux, Windows) e ambientes de CI/containers. O driver mais popular (mattn/go-sqlite3) depende de CGO, exigindo toolchain C em cada plataforma de build, o que complica cross-compile, imagens Docker enxutas e pipelines de CI.

## Decision

Adotar o driver modernc.org/sqlite (implementação pure-Go, sem CGO) como driver SQLite padrão da API.

## Consequences

**Pros:**
- Build 100% Go: cross-compile e `CGO_ENABLED=0` sem toolchain C
- Imagens Docker menores (sem libs C / scratch-friendly)
- CI mais simples e rápido em todas as plataformas

**Cons:**
- Performance levemente inferior ao CGO em cargas pesadas
- Menos "battle-tested" que mattn/go-sqlite3

**Neutros:**
- Mesma interface `database/sql`; troca futura é viável

## Alternatives considered

- **mattn/go-sqlite3 (CGO)** — driver SQLite mais popular do ecossistema Go. Motivo da rejeição: exige toolchain C, dificultando cross-compile, imagens Docker enxutas e pipelines de CI multiplataforma.

## Applied in

-
