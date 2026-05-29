---
id: 0006
title: Recuperação de senha via senha temporária por e-mail (Resend)
status: accepted
date: 2026-05-29
tags: [auth, security, cross-cutting]
---

# 0006 - Recuperação de senha via senha temporária por e-mail (Resend)

## Context

O domínio `auth` oferece apenas Register/Login/GetMe — um usuário que esquece a
senha não tem como recuperar o acesso. Precisamos de um fluxo de recuperação que
funcione sobre a API gRPC (sem página web hospedada para um link de reset) e que
introduza, pela primeira vez no projeto, o envio de e-mail transacional. A
abordagem precisa respeitar a postura anti-enumeração já adotada no Login e
permanecer compatível com a arquitetura stateless sem refresh (ADR-0003).

## Decision

Adotamos recuperação por **senha temporária**: o backend gera uma senha aleatória,
persiste seu hash bcrypt, a envia em texto por e-mail via **Resend** (provider
acessado por uma interface `EmailSender` declarada no consumidor) e força a **troca
obrigatória no primeiro login** com essa senha. A temporária **expira em 15 minutos**
e o pedido de reset sempre retorna resposta genérica (falha de envio apenas logada).

## Consequences

**Pros:**
- Reusa bcrypt e Login existentes; sem tabela nova (apenas flag de troca + controle de expiração).
- Resposta genérica preserva a defesa anti-enumeração já vigente no Login.
- `EmailSender` no consumidor mantém o padrão fx do projeto e permite fake/no-op em testes.

**Cons:**
- A senha em texto trafega e **persiste no inbox** do usuário até a troca/expiração — superfície de exposição inerente.
- Janela de 15 min é apertada para o ciclo receber→logar→trocar; atraso de entrega pode expirar antes do uso.
- Dependência externa nova (Resend): indisponibilidade do provedor afeta o envio.

**Neutros:**
- A troca de senha não invalida JWTs já emitidos (stateless sem refresh, ADR-0003); expiram em até 1h.
- Rate limit no pedido de reset fica para v2 (sem mecanismo pronto hoje).

## Alternatives considered

- **Token/código de reset de uso único** — e-mail leva código/link e o usuário define a senha definitiva direto, sem senha válida no inbox. Motivo da rejeição: o usuário optou pela senha temporária; o token exigiria tabela + RPC dedicados.
- **OTP numérico de 6 dígitos** — variante do token com código curto (UX mobile). Motivo da rejeição: mesma do token (variante de uma abordagem já preterida).
- **Login restrito por escopo de token enquanto a troca é obrigatória** — token que só autoriza ChangePassword. Motivo da rejeição: over-engineering; flag `must_change_password` + RPC dedicado bastam.
- **Fail-closed no envio de e-mail** (retornar erro se o e-mail não sair). Motivo da rejeição: vazaria a existência do e-mail, quebrando a postura anti-enumeração.

## Applied in

-
