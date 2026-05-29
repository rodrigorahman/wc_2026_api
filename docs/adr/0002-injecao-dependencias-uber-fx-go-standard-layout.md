---
id: 0002
title: Injeção de dependências com uber-fx e go-standard-layout
status: accepted
date: 2026-05-27
tags: [architecture, build, cross-cutting]
---

# 0002 - Injeção de dependências com uber-fx e go-standard-layout

## Context

A API cresce em número de módulos (handlers, services, repos) e precisa de wiring consistente, gerenciamento de lifecycle (start/stop de servidores, conexões) e ordem de inicialização previsível. Wiring manual tende a virar um construtor gigante e frágil à medida que as dependências crescem. Falta também uma convenção clara de organização de pastas.

## Decision

Adotar uber-fx como container de injeção de dependências e lifecycle, organizando o código segundo o go-standard-layout (`cmd/`, `internal/`, `pkg/`) com módulos fx por domínio.

## Consequences

**Pros:**
- Wiring declarativo e modular (`fx.Module` por domínio)
- Lifecycle gerenciado (`OnStart`/`OnStop`) para servidores e conexões
- Layout padrão reduz atrito de onboarding

**Cons:**
- Curva de aprendizado do fx; erros de DI só em runtime
- Toque de "magia"/reflection vs wiring explícito

**Neutros:**
- Construtores permanecem Go idiomático; fx só monta o grafo

## Alternatives considered

- **Wiring manual** — explícito, sem dependências externas. Motivo da rejeição: vira construtor gigante e frágil conforme o grafo cresce; sem gestão de lifecycle pronta.
- **google/wire** — DI por codegen, com erros em compile-time. Motivo da rejeição: não gerencia lifecycle e adiciona passo de geração extra no build.

## Applied in

-
