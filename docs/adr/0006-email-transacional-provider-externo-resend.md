---
id: 0006
title: Envio de e-mail transacional via provider externo (Resend) com interface no consumidor
status: accepted
date: 2026-05-29
tags: [architecture, http, cross-cutting]
---

# 0006 - Envio de e-mail transacional via provider externo (Resend) com interface no consumidor

> **Refocada em 2026-05-31**: esta ADR foi originalmente redigida em torno da feature "recuperação de senha", o que misturava produto com arquitetura. Reescrita para registrar a decisão **transversal** que de fato motiva uma ADR — como o projeto envia e-mail transacional. O design **feature-scoped** de recuperação por senha temporária (escolha sobre token/OTP, coexistência de credenciais, e-mail de notificação de troca) vive no `tech-alignment.md` da feature `esqueci-a-senha`, que referencia esta ADR para a parte de e-mail.

## Context

O projeto não possui nenhuma capacidade de envio de e-mail. A primeira feature a
exigi-la é a recuperação de senha, mas o envio transacional é uma capacidade de
infraestrutura que outras features tendem a reusar (confirmação de cadastro,
notificações). Precisamos de uma forma consistente de enviar e-mail que: respeite
a restrição pure-Go/sem-CGO (ADR-0001); se encaixe no padrão de DI por uber-fx com
interface declarada no consumidor (ADR-0002); seja fakeável em testes e local; e
tenha uma postura de falha definida — o envio transacional é um efeito colateral
que, falhando, **não deve** necessariamente quebrar o fluxo chamador.

## Decision

Adotamos uma interface **`EmailSender` declarada no pacote consumidor**, com uma
implementação concreta apoiada no **Resend** (API HTTP, pure-Go, sem CGO). O fx faz
o bind concreto→interface no módulo, permitindo um fake/no-op em testes e em
desenvolvimento.

A **postura de falha padrão é best-effort**: a falha de envio é **registrada (log),
não propagada** como erro fatal ao fluxo chamador; token/credencial e o conteúdo
sensível do e-mail nunca são logados. Cada consumidor decide se precisa de garantia
mais forte, mas o default da capacidade é best-effort.

A **credencial** (`RESEND_API_KEY`) segue a política de config do projeto:
**fail-fast obrigatória em produção** (ausência aborta o boot, como `JWT_SECRET`),
com um **sender no-op em desenvolvimento** (loga o e-mail em vez de enviar) quando a
chave está ausente, reusando `Config.IsDevelopment()`.

## Consequences

**Pros:**
- `EmailSender` no consumidor mantém o padrão fx (ADR-0002) e permite fake/no-op em testes e dev.
- Resend via HTTP pure-Go preserva a build sem CGO (ADR-0001); nenhuma toolchain C nova.
- Best-effort por padrão desacopla o fluxo chamador da disponibilidade do provider.
- Política de credencial espelha a de `JWT_SECRET`: produção segura por construção, dev sem atrito.

**Cons:**
- Dependência externa nova (Resend): indisponibilidade/limite de envio do provider afeta a entrega.
- Best-effort significa que uma falha de envio é silenciosa para o usuário (só observável no log/telemetria) — exige monitorar a taxa de falha de envio.
- Acopla o projeto à API de um provider específico; trocar de provider exige reimplementar o `EmailSender` (mitigado por a interface isolar o consumidor).

**Neutros:**
- O mecanismo concreto de chamada (cliente HTTP próprio × SDK do provider) e a sincronia do envio (síncrono × assíncrono) são detalhes de cada consumidor — ficam no tech-alignment da feature, não nesta ADR.
- Template/identidade visual do e-mail não é coberto aqui (decisão de produto por feature).

## Alternatives considered

- **Outro provider (SendGrid / Amazon SES / Mailgun)** — Motivo da rejeição: o Resend foi a escolha travada; sua API HTTP simples e pure-Go atende sem CGO e sem SDK pesado.
- **SMTP direto / servidor de e-mail próprio** — Motivo da rejeição: overhead operacional e de deliverability injustificado para um serviço único; reintroduziria infraestrutura que a arquitetura stateless evita.
- **Postura fail-closed** (propagar a falha de envio como erro fatal do fluxo chamador) — Motivo da rejeição: acopla o fluxo à disponibilidade do provider; no caso de recuperação ainda vazaria a existência do e-mail por timing/erro. Best-effort com log é o default.
- **Chamar o provider direto no service, sem interface** — Motivo da rejeição: quebra o padrão "interface no consumidor" (ADR-0002) e impede fake/no-op em teste e dev.

## Applied in

- Feature `esqueci-a-senha` (recuperação de senha): primeira consumidora — ver `docs/specs/features/esqueci-a-senha/v1/tech-alignment.md`.
