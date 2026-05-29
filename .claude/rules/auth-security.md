---
description: Autenticação e segurança da WC 2026 API — JWT HS256 com defesa alg-confusion, bcrypt cost 12, equalização de timing anti-enumeração, fail-fast de JWT_SECRET, sub via contexto e clock injetável. Carregada ao editar o domínio auth, config ou clock. Materializa a ADR-0003.
paths:
  - "internal/auth/**"
  - "internal/config/**"
  - "internal/clock/**"
  - "internal/server/**"
---

# WC 2026 API — Autenticação e Segurança

> ADR-0003 (JWT HS256 + bcrypt + TTL 1h sem refresh). Detalhe em [`docs/adr/0003-autenticacao-jwt-hs256-bcrypt-ttl-1h-sem-refresh.md`](../../docs/adr/0003-autenticacao-jwt-hs256-bcrypt-ttl-1h-sem-refresh.md).

## JWT

- HS256 com segredo compartilhado. A validação **sempre** usa `jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}))` — rejeita `alg=none` e RS256 (defesa contra alg-confusion). Não remova sob pretexto de simplificação.
- Erros de token são sentinelas do `golang-jwt` comparados com `errors.Is` (`jwt.ErrTokenExpired`, `jwt.ErrTokenSignatureInvalid`), nunca por string.
- A expiração usa o clock injetado via `jwt.WithTimeFunc(clock.Now)` para consistência entre emissão e validação.

## Senhas

- **bcrypt cost 12** (configurável no construtor; menor só em CI por performance). Senha plana **nunca** é persistida, retornada ou logada.
- Login equaliza timing: no caminho "e-mail inexistente", execute `bcrypt.CompareHashAndPassword` contra um **hash dummy fixo** antes de retornar `Unauthenticated` (anti-enumeração por timing). Mensagem genérica idêntica à de senha errada.

## Fail-fast de config

- `JWT_SECRET` ausente ou `< 32` bytes → a config retorna `error` (não `os.Exit`/`log.Fatal` interno). O composition root aborta **antes** de servir. Não suba servidor inseguro silenciosamente.

## Acesso protegido

- RPCs protegidos (ex.: `GetMe`) leem o `sub` do **contexto** injetado pelo interceptor JWT — nunca de input do cliente (evita IDOR).
- Token/credencial **nunca** aparecem em log.

## Clock determinístico

- Código que depende de tempo recebe `clock.Clock` injetado (`internal/clock`); **não** chame `time.Now()` direto em código testável.

> Cadeia de interceptors e public methods: [`grpc-layers.md`](grpc-layers.md). Como testar auth/clock: [`testing.md`](testing.md).
