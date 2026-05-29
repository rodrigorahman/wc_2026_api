---
description: Contratos gRPC e camadas da WC 2026 API — proto versionado + protovalidate, stubs gerados, separação handler/service/repository, ordem da cadeia de interceptors e tratamento de erros com códigos gRPC e sentinelas. Carregada ao editar proto, handlers, services, interceptors ou o servidor.
paths:
  - "proto/**"
  - "internal/**/handler/**"
  - "internal/**/service/**"
  - "internal/**/interceptor/**"
  - "internal/server/**"
  - "buf.yaml"
  - "buf.gen.yaml"
---

# WC 2026 API — gRPC, Camadas e Erros

## Contratos proto

- Em `proto/wc2026/<dominio>/v1/` com pacote `wc2026.<dominio>.v1`. Campos em inglês.
- Validação de entrada é **declarativa** via opções `buf.validate.*` (protovalidate — module path `buf.build/go/protovalidate`), não validação manual no handler.
- Stubs em `internal/pb/wc2026/**` gerados via `make proto`. **Não edite** código gerado.

## Camadas por domínio

- `handler` → `service` → `repository`. Responsabilidades:
  - **handler**: mapper proto↔domínio **puro**. Converte request↔domínio e **propaga** o erro do service sem retraduzir o código gRPC. Sem regra de negócio.
  - **service**: regra de negócio; retorna `status.Error(codes.X)` com o código correto.
  - **repository**: thin wrapper sobre sqlc, queries parametrizadas. Sem regra de negócio.

## Cadeia de interceptors

- Ordem **fixa** (decisão arquitetural — não reordenar): `recovery → logging → protovalidate → auth JWT`.
- Montada uma única vez em `internal/server` (compartilhada por produção e E2E).
- Métodos públicos vs protegidos: use as **constantes geradas** `…_FullMethodName` (nunca string literal) — typo em literal abriria um RPC protegido (fail-open).

## Tratamento de erros

- Service mapeia erro de domínio em `status.Error(codes.X)` (`InvalidArgument`, `AlreadyExists`, `Unauthenticated`, `Internal`, ...). Mensagens de credencial genéricas e idênticas (anti-enumeração).
- Sentinelas (`ErrUserNotFound`, `ErrNationalTeamNotFound`, ...) comparados com `errors.Is` — **nunca** por substring da mensagem.
- Tradução de sentinela entre camadas em **um único ponto** (o adapter do `fx.Module` — ver [`di-layers.md`](di-layers.md)).

> Idioma dos campos: [`language-naming.md`](language-naming.md). Auth/JWT do interceptor: [`auth-security.md`](auth-security.md).
