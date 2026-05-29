---
description: Injeção de dependências e camadas da WC 2026 API — uber-fx por domínio, bind concreto→interface no módulo, interface declarada no consumidor, adapters de tradução e composition root compartilhado. Carregada ao editar fx.Module, o composition root ou wiring. Materializa a ADR-0002.
paths:
  - "internal/**/module.go"
  - "internal/server/**"
  - "cmd/**"
---

# WC 2026 API — DI (uber-fx) e Camadas

> ADR-0002 (uber-fx + go-standard-layout). Detalhe em [`docs/adr/0002-injecao-dependencias-uber-fx-go-standard-layout.md`](../../docs/adr/0002-injecao-dependencias-uber-fx-go-standard-layout.md).

## Módulos por domínio

- Wiring vive em `fx.Module` **por domínio** (`internal/domain/<dominio>/module.go`). Um módulo provê seus componentes (repository, service, handler, interceptors) e declara dependências cross-domain que **não** possui.
- O **bind concreto→interface** acontece dentro do `fx.Module` (via `fx.Provide` que retorna o tipo da interface), **nunca** no consumidor.
- Colisão de tipo no grafo (ex.: dois `grpc.UnaryServerInterceptor`) → desambigue com `fx.ResultTags`/`fx.ParamTags` por nome.

## Interface no consumidor

- Quem **consome** declara a interface, com os métodos mínimos que usa (ex.: `auth/service` declara `UserRepository`, `NationalTeamRepository`, `TokenManager`).
- O pacote **concreto** não conhece o consumidor e expõe seus próprios tipos/sentinelas.
- Quando tipos/sentinelas divergem entre concreto e interface, escreva um **adapter fino no `fx.Module`** que converte tipos e traduz sentinelas — em **um único ponto** (não duplique a tradução no service e no adapter). Não force a interface a vazar o tipo do concreto.

## Composition root

- `cmd/server/main.go` compõe os módulos por domínio + os providers compartilhados de `internal/server`.
- `internal/server` é pacote próprio (não `package main`) **de propósito**: produção (`cmd/server`) e E2E (`internal/testutil/bufconn.go`) montam a **mesma** cadeia e wiring. Não duplique a montagem em teste.
- Config NÃO é provida por `internal/server.Providers`: o binário a carrega via `ConfigProvider` (fail-fast); o helper de teste injeta uma config fixa.

> Erros/sentinelas e a cadeia gRPC: [`grpc-layers.md`](grpc-layers.md). Fail-fast de config e segurança: [`auth-security.md`](auth-security.md).
