---
id: 0003
title: Autenticação via JWT HS256 com bcrypt e TTL de 1h sem refresh
status: accepted
date: 2026-05-28
tags: [auth, security, cross-cutting]
---

# 0003 - Autenticação via JWT HS256 com bcrypt e TTL de 1h sem refresh

## Context

A API é stateless e roda como serviço único, sem necessidade de compartilhar
chaves de verificação entre múltiplos serviços. Precisamos de um mecanismo de
autenticação que valide tokens localmente, com baixo overhead operacional e
sem infraestrutura adicional de store de sessão. As senhas precisam de hashing
resistente e os tokens de um tempo de vida controlado.

## Decision

Adotamos JWT assinado com HS256 (segredo compartilhado), senhas com hash bcrypt
e access token com TTL de 1 hora, sem refresh token — a renovação exige novo login.

## Consequences

**Pros:**
- Stateless e simples: validação local do token, sem store de sessão, menos infra e código.

**Cons:**
- Sem revogação imediata: o token é válido até expirar; logout/revogação antecipada não é possível sem blocklist.
- Re-login a cada 1h: o TTL curto sem refresh obriga novo login na expiração, impactando a UX.
- Segredo HS256 crítico: vazamento do segredo compartilhado compromete todos os tokens; exige rotação e guarda cuidadosa.

**Neutros:**
- TTL de 1h é configurável e pode ser ajustado sem mudança estrutural.

## Alternatives considered

- **JWT RS256 (assimétrico)** — par de chaves pública/privada em vez de segredo compartilhado. Motivo da rejeição: complexidade de gestão de chaves desnecessária para um único serviço.
- **Sessão server-side (cookie + store)** — sessões com estado em Redis/DB. Motivo da rejeição: exige infraestrutura de store e quebra a statelessness da API.
- **JWT com refresh token** — access + refresh token rotativo. Motivo da rejeição: TTL curto de 1h sem refresh foi preferido pela simplicidade nesta fase.
- **OAuth2 / provedor externo** — delegar auth a Auth0/Keycloak/Cognito. Motivo da rejeição: overhead de integração e dependência externa não justificados.

## Applied in

-
