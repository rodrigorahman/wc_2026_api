# Pré-Refinamento — Brainstorm de Produto

> Artefato **intermediário** (anterior ao PRD / INTENT / TaskCard), produto de um brainstorm em **Tree of Thought**: divergir os rumos possíveis, podar com o usuário e convergir.
>
> **Legenda:**
> - Linhas sem marcação = **FATO** (afirmado pelo usuário).
> - `[HIPÓTESE]` = inferência da skill que precisa ser validada.
> - `[DÚVIDA]` = ponto em aberto, detalhado na seção 13.
> - `[fora do escopo do projeto]` = rumo que extrapola o que este projeto se propõe a ser.

---

## 1. Metadados

- **Nome da Ideia / Feature**: Esqueci a senha (recuperação de senha por e-mail)
- **Fonte da ideia**: texto livre (descrição do usuário no comando)
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Versão**: v1
- **Status**: Refinado
- **Relacionados**: `arquitetura-base/v1` (domínio `auth` — Register/Login/GetMe), ADR-0003 (JWT HS256 + bcrypt + TTL 1h sem refresh)

---

## 2. Ideia Resumida (uma frase)

Permitir que um usuário que esqueceu a senha solicite uma **senha temporária enviada por e-mail (via Resend)** e seja **forçado a trocá-la no primeiro login** com essa senha.

---

## 3. Esqueleto do Tema (Fase 1 — ramos da árvore)

| # | Ramo | Status (Fase 1) |
|---|------|-----------------|
| A | Mecanismo de recuperação (senha temporária × token × OTP) | explorar |
| B | Integração com Resend (envio de e-mail e falha) | explorar |
| C | Troca obrigatória de senha (flag, gatilho, RPC) | explorar |
| D | Segurança e anti-abuso (expiração, rate limit, anti-enumeração) | explorar |
| E | Escopo do "primeiro login" (só pós-reset × todo novo usuário) | explorar |

---

## 4. Árvore de Rumos (Fase 2 — Tree of Thought)

### Ramo A — Mecanismo de recuperação

**Direções candidatas:**

- **A1 — Senha temporária por e-mail**: o sistema gera uma senha aleatória, grava o hash bcrypt em `password_hash` e envia a senha em texto no e-mail; o usuário loga com ela e é forçado a trocar.
  - _Exemplo:_ `RequestPasswordReset(email)` gera `Xk7$9pQz`, salva o hash, envia o e-mail; usuário faz Login normal com a temporária.
  - _Viabilidade:_ reusa bcrypt e Login já existentes; **sem tabela nova** (precisa apenas de campos para expiração e flag de troca). Ônus de segurança: a senha em texto permanece no inbox e é credencial válida até a troca/expiração.
- **A2 — Token/código de reset de uso único**: gera um token opaco, grava hash + expiração numa tabela `password_resets` e envia o código; o usuário chama `ResetPassword(token, newPassword)` definindo a senha definitiva.
  - _Exemplo:_ e-mail "seu código: `821455`, válido por 15 min" → app chama `ResetPassword`.
  - _Viabilidade:_ padrão mais seguro (não deixa senha válida no inbox); requer 1 tabela + 1 RPC novos. **Elimina o Ramo C** porque o usuário já define a senha no reset.
- **A3 — OTP numérico de 6 dígitos**: variante de A2 com código numérico curto (UX mobile). Mesmo backend, mesma segurança e custo.
  - _Exemplo:_ e-mail "código `482190`" digitado no app.
  - _Viabilidade:_ igual ao A2.

**Direção escolhida**: **A1** — decisão do usuário; aceita conscientemente o ônus da senha em texto, mitigado pela troca obrigatória (Ramo C) e expiração curta (Ramo D).
**Podadas / adiadas**: A2 (podado — mais seguro, porém o usuário optou pela abordagem proposta), A3 (podado — variante de A2).

### Ramo B — Integração com Resend

**Direções candidatas:**

- **B1 — Interface `EmailSender` no consumidor + implementação Resend (HTTP REST)**: segue o padrão "interface no consumidor" do projeto; Resend é HTTP puro (sem CGO). Fake/no-op sender em testes, como já se faz com `clock` e `compareFunc` no `AuthService`.
  - _Exemplo:_ o `AuthService` (ou um service de recuperação) depende de `EmailSender.Send(to, subject, body)`; fx faz o bind concreto→interface no módulo.
  - _Viabilidade:_ encaixa direto na arquitetura fx; API key nova via config (viper/env). **Capacidade nova no projeto** (hoje não há envio de e-mail nem dependência Resend).
- **B2 — Comportamento em falha de envio**: _fail-closed_ (retorna erro se o e-mail não sair) × _best-effort_ (loga a falha e responde sempre genérico).
  - _Exemplo:_ Resend retorna 500 → o RPC ainda responde "se o e-mail existir, enviamos as instruções".
  - _Viabilidade:_ a postura anti-enumeração já existente no Login empurra para resposta sempre genérica.

**Direção escolhida**: **B1** (interface `EmailSender` + impl Resend + fake em testes) e **B2 = resposta sempre genérica** com a falha de envio apenas logada.
**Podadas / adiadas**: B2 fail-closed (podado — vazaria existência do e-mail e quebraria a postura anti-enumeração).

### Ramo C — Troca obrigatória de senha

**Direções candidatas:**

- **C1 — Flag `must_change_password` em `users`**: Login emite o token normalmente mas devolve o flag; o cliente redireciona para um `ChangePassword(current, new)`.
  - _Exemplo:_ Login responde `{ access_token, must_change_password: true }`; o app abre a tela de troca.
  - _Viabilidade:_ requer 1 coluna nova + 1 RPC `ChangePassword`. Simples; o controle de fluxo fica no cliente.
- **C2 — Login restrito**: enquanto o flag está ativo, o Login emite um token que só autoriza `ChangePassword`. Mais rígido.
  - _Exemplo:_ token com escopo limitado; demais RPCs retornam `PermissionDenied` até a troca.
  - _Viabilidade:_ exige escopo/claim no token e checagem no interceptor — mais código e mais superfície.

**Direção escolhida**: **C1** — mais simples e suficiente para o objetivo; o front controla o fluxo de troca.
**Podadas / adiadas**: C2 (podado — over-engineering para o escopo; token com escopo restrito não se justifica agora).

### Ramo D — Segurança e anti-abuso

**Direções candidatas:**

- **D1 — Expiração da senha temporária**: a temporária deixa de funcionar após uma janela.
  - _Exemplo:_ campo de expiração consultado no Login; após o prazo, a temporária é rejeitada.
  - _Viabilidade:_ requer registrar quando a temporária foi emitida; checagem no Login.
- **D2 — Rate limit no `RequestPasswordReset`**: limita pedidos por e-mail/IP para evitar flood de e-mails.
  - _Exemplo:_ no máximo N pedidos por janela de tempo.
  - _Viabilidade:_ **não há mecanismo de rate limit pronto** no projeto hoje; exigiria estado server-side novo.
- **D3 — Anti-enumeração**: `RequestPasswordReset` responde sempre genérico (alinhado a B2).
  - _Viabilidade:_ reusa a postura já adotada no Login.

**Direção escolhida**: **D1 = expiração de 15 minutos** + **D3 = resposta sempre genérica**.
**Podadas / adiadas**: D2 rate limit (**adiado para v2** — sem mecanismo pronto; registrado como limitação conhecida).

### Ramo E — Escopo do "primeiro login"

**Direções candidatas:**

- **E1 — Só após reset**: a troca obrigatória dispara apenas quando o usuário logou com a senha temporária recebida por e-mail.
  - _Exemplo:_ cadastro self-service continua escolhendo a própria senha, sem troca forçada.
  - _Viabilidade:_ coerente com o fluxo atual de `Register`.
- **E2 — Todo novo usuário**: qualquer cadastro força troca no primeiro login.
  - _Viabilidade:_ incomum no fluxo self-service atual (a pessoa acabou de escolher a senha no Register).

**Direção escolhida**: **E1 — só após reset.** O cadastro normal não muda.
**Podadas / adiadas**: E2 (podado — não faz sentido no cadastro self-service atual).

---

## 5. Problema

- **Qual é a dor real hoje?** O usuário que esquece a senha **não tem como recuperá-la** — `auth` só oferece Register/Login/GetMe. Sem recuperação, a conta fica inacessível para sempre.
- **Como o problema aparece no dia a dia?** Usuário tenta logar, erra a senha repetidamente e não encontra saída; abandona a conta (e potencialmente o app).
- **Quem sente o impacto?** O usuário final do Álbum da Copa (quem perde acesso à conta e ao progresso).
- **Por que resolver agora?** Recuperação de senha é capacidade básica de qualquer produto com login; sua ausência bloqueia usuários legítimos.

---

## 6. Objetivo Principal

- **Qual é o resultado esperado ao final?** Um usuário que esqueceu a senha consegue, sozinho e por e-mail, **recuperar o acesso** à conta de forma segura.
- **Qual mudança de comportamento/estado deve acontecer?** A conta sai do estado "inacessível por esquecimento de senha" para "acesso restaurado", passando por uma **troca obrigatória** que invalida a senha temporária.

---

## 7. Público / Usuário Envolvido

- **Persona primária**: usuário final do Álbum da Copa 2026 (já cadastrado) que esqueceu a senha.
- **Persona secundária**: nenhuma (não há painel administrativo; o fluxo é self-service).
- **Contexto de uso**: app cliente consumindo a API gRPC; usuário aciona "esqueci a senha", recebe o e-mail (Resend) e retoma o acesso pelo mesmo app.

---

## 8. Escopo Inicial (resultado da convergência)

Direções escolhidas na árvore de rumos que entram na primeira versão:

- [ ] **Recuperação por senha temporária** (A1): RPC para solicitar reset, geração de senha aleatória, hash bcrypt persistido.
- [ ] **Envio de e-mail via Resend** (B1): interface `EmailSender` no consumidor + implementação Resend (HTTP), fake em testes, API key via config.
- [ ] **Resposta sempre genérica** no pedido de reset (B2/D3): não revela se o e-mail existe; falha de envio apenas logada.
- [ ] **Troca obrigatória de senha** (C1): flag `must_change_password` em `users`, Login sinaliza o flag, RPC `ChangePassword(current, new)`.
- [ ] **Expiração de 15 minutos** (D1): a senha temporária é rejeitada após a janela.
- [ ] **Gatilho de troca só pós-reset** (E1): cadastro self-service permanece inalterado.

> Ponto de partida para o PRD/INTENT/TaskCard — não é definitivo.

---

## 9. Fora do Escopo (podado / adiado)

- **Token/código de reset de uso único (A2) e OTP (A3)** — _podados_: mais seguros, mas o usuário optou pela senha temporária por e-mail.
- **Login restrito por escopo de token (C2)** — _podado_: over-engineering; o flag + RPC dedicado bastam.
- **Rate limit no pedido de reset (D2)** — _adiado para v2_: não há mecanismo pronto no projeto; registrado como limitação conhecida.
- **Fail-closed no envio de e-mail** — _podado_: quebraria a postura anti-enumeração.
- **Invalidação de tokens JWT já emitidos após a troca** — _fora do escopo_: a arquitetura é stateless sem refresh (ADR-0003); tokens existentes expiram naturalmente em 1h.

---

## 10. Ancoramento no Projeto (guarda de escopo)

- **O que o projeto É** (CLAUDE.md / README): API gRPC em Go (uber-fx + SQLite pure-Go + sqlc), auth JWT/bcrypt, compilável sem CGO. Domínio: Álbum da Copa 2026 (auth + seleções nacionais).
- **PRDs / specs existentes consultados** (`/docs/specs/**/*.md`):
  - `arquitetura-base/v1` — **adjacente/base**: define o domínio `auth` (Register/Login/GetMe), o `UserRepository` e a defesa anti-enumeração por timing no Login. É a fundação que esta feature estende.
  - `usuario-multiplas-selecoes/v1`, `national-team-flag-url/v1`, `proximos-jogos-selecoes/v1` — **nada a ver** (domínio de seleções/jogos).
  - **Nenhuma spec existente cobre recuperação de senha** — sem duplicação.
- **Capacidades reutilizáveis** (apenas para viabilidade):
  - **Persistência**: tabela `users` (`id, full_name, email UNIQUE, password_hash, created_at`); migrations embarcadas via golang-migrate; queries via sqlc. Precisará de coluna(s) nova(s) (flag de troca + controle de expiração) — `make sqlc` após alterar schema.
  - **Autenticação / autorização**: `AuthService` (bcrypt cost 12, `compareFunc` e `clock` injetados como boundaries testáveis), `TokenManager` (JWT HS256), interceptor JWT. Reaproveitáveis para o fluxo de troca.
  - **Outros módulos internos**: `internal/infra/config` (viper + fail-fast) para a API key do Resend; `internal/infra/clock` para expiração determinística; `internal/infra/logger` (zap) para logar falha de envio. **Não há** módulo de e-mail hoje — capacidade nova.
- **Conflitos / sobreposições detectados**: nenhum. A feature **estende** o domínio `auth` sem colidir com specs existentes.

---

## 11. Premissas e Decisões já tomadas

**Premissas** — suposições assumidas para que a ideia faça sentido:

- [HIPÓTESE] Existe um app cliente capaz de chamar os novos RPCs (solicitar reset e trocar senha) — o handoff-frontend de features anteriores sugere um frontend em construção.
- [HIPÓTESE] A senha temporária é gerada pelo backend (aleatória, forte) — o trecho "uma nova senha cadastrada" foi interpretado como senha gerada pelo sistema, não informada pelo usuário no pedido.
- [HIPÓTESE] O `EmailSender` será um módulo novo em `internal/infra` (ex.: `internal/infra/email`), seguindo o padrão de infra cross-cutting do projeto.

**Decisões já tomadas (fora de negociação)** — restrições travadas pelo usuário:

- Usar **Resend** (https://resend.com) como serviço de envio de e-mail.
- Recuperação via **senha temporária enviada por e-mail** (não token/OTP).
- **Troca obrigatória de senha no primeiro login após o reset** (cadastro normal não muda).
- **Resposta sempre genérica** no pedido de reset (anti-enumeração).
- **Expiração da senha temporária: 15 minutos.**
- **Rate limit fica para v2.**

---

## 12. Riscos e Pontos de Atenção

- **Risco de produto / aceitação**: janela de **15 min** é apertada para o ciclo "receber e-mail → logar com a temporária → trocar senha"; atraso na entrega do e-mail pode expirar antes do uso → **mitigação**: revisitar a janela (ex.: 30–60 min) se houver atrito; medir tempo real de entrega do Resend.
- **Risco de escopo** (pode explodir?): integração de e-mail tende a puxar templates, múltiplos idiomas, reenvio, etc. → **mitigação**: manter v1 a um único e-mail transacional simples; rate limit e refinos ficam para v2.
- **Risco técnico ou operacional**: dependência externa nova (Resend) — indisponibilidade/limite de envio do provedor → **mitigação**: best-effort com falha logada; resposta genérica não acopla a UX à disponibilidade do provedor.
- **Risco de privacidade / segurança / compliance**: senha em texto trafega e **persiste no inbox** do usuário (ônus do A1) → **mitigação**: expiração curta (15 min) + troca obrigatória + nunca logar a senha/temporária (regra de auth já vigente). **Limitação conhecida**: a troca não invalida JWTs já emitidos (stateless, ADR-0003) — expiram em 1h.

---

## 13. Dúvidas em Aberto

1. [DÚVIDA] A janela de **15 min** é confortável para o tempo real de entrega do Resend? (medir; possivelmente ampliar.)
2. [DÚVIDA] A API key do Resend deve seguir o padrão **fail-fast no boot** (como `JWT_SECRET`) ou ser opcional (degradando o envio em dev)?
3. [DÚVIDA] O e-mail terá **template/identidade visual** ou basta texto simples em v1?
4. [DÚVIDA] Confirmar a premissa: a senha temporária é **gerada pelo sistema** (não informada pelo usuário no pedido)?

---

## 14. Síntese do Brainstorm

- **Absorvido no escopo inicial (seção 8)**: senha temporária por e-mail (A1), `EmailSender` + Resend com fake em testes (B1), resposta sempre genérica (B2/D3), flag `must_change_password` + `ChangePassword` (C1), expiração 15 min (D1), troca só pós-reset (E1).
- **Descartado com justificativa**: token/OTP (A2/A3 — usuário optou por senha temporária); Login restrito por escopo (C2 — over-engineering); fail-closed no envio (vaza enumeração).
- **Adiado para v2/v3**: rate limit no pedido de reset (D2); possível ampliação da janela de expiração; template/identidade visual do e-mail.
- **Provocações que mudaram o rumo**: a comparação A1×A2 evidenciou que o token de reset (A2) seria *mais simples no produto* (colapsaria o Ramo C); o usuário manteve A1 conscientemente, então a troca obrigatória + expiração curta entraram como mitigadores explícitos.

---

## 15. Recomendação de Framework

### 15.1 Complexidade Observada

| Dimensão | Valor detectado | Confirmação |
|---|---|---|
| Amplitude — # rumos/US que sobreviveram | 2-3 | confirmado |
| Personas | dev+1 (usuário final) | inferido |
| Novidade | incremento ao `auth` **+ integração externa nova (Resend)** | confirmado |
| Decisão arquitetural transversal nova? | **sim** — envio de e-mail transacional (provider externo + postura de falha) e postura de segurança "senha temporária em texto" são decisões evergreen | inferido |

### 15.2 Framework Recomendado

**Escolhido**: `SDD`

**Justificativa** (2-3 frases citando 2 dimensões decisivas): há uma **decisão arquitetural transversal nova** — introdução de envio de e-mail transacional via provider externo (Resend) com interface `EmailSender`, postura de falha e localização da API key — que o projeto ainda não possui e que merece registro evergreen (à semelhança do ADR-0003 para auth). Soma-se uma **decisão de segurança não-trivial** (recuperação por senha temporária em texto, com expiração + troca obrigatória como mitigadores), que toca caminhos críticos (`auth`, `crypto`, `db_migrations`). Pela regra de desempate, qualquer sinal arquitetural novo faz o SDD vencer.

### 15.3 Alternativas Consideradas

**Por que NÃO miniSpec** (vizinho mais próximo): o miniSpec pressupõe incremento **sem decisão arquitetural transversal nova**. Aqui há justamente isso — uma capacidade de infra inédita (e-mail/Resend) e uma postura de segurança que vira ADR. O miniSpec não comporta o registro formal dessas decisões evergreen nem o PRD que alinha o trade-off de segurança com o produto.

**Por que NÃO TaskCard** (vizinho mais distante): sub-dimensionado. O escopo atravessa migration de schema, novo módulo de infra (e-mail), 2+ RPCs novos (solicitar reset, trocar senha), mudança no Login e área crítica de segurança — longe de um ajuste pontual de 1 arquivo.

### 15.4 Próximo Passo

```bash
# Há decisão arquitetural transversal nova — registre a ADR ANTES do PRD:
/agent-spec-adr-create "recuperacao de senha via senha temporaria por e-mail (Resend)"

# Em seguida, gere o PRD do SDD a partir deste pré-refinamento:
/agent-spec-sdd-generate-prd "sistema de recuperacao de senha (esqueci a senha) por e-mail via Resend"
```

### 15.5 Quando Reconsiderar a Recomendação

- **Upgrade** (mantém SDD / amplia): se surgir um painel administrativo com persona nova, ou se a integração de e-mail crescer para múltiplos fluxos transacionais (confirmação de cadastro, notificações).
- **Downgrade** para miniSpec: se a decisão de e-mail/Resend for registrada/resolvida em ADR à parte **e** o restante se reduzir a um incremento mecânico no `auth` sem novas decisões — aí o PRD completo perde valor.

---

## 16. Checklist Final

- [x] Ideia resumida em uma frase clara
- [x] **Esqueleto (seção 3)** com 3-5 ramos, validado com o usuário na Fase 1
- [x] **Árvore de rumos (seção 4)**: cada ramo com direções candidatas + exemplo concreto + viabilidade + direção escolhida/podada
- [x] Rumos fora do escopo do projeto marcados como `[fora do escopo do projeto]` (nenhum aplicável)
- [x] Problema, público, escopo inicial e fora de escopo delimitados
- [x] **Ancoramento (seção 10)** preenchido com PRDs/capacidades concretos
- [x] Toda inferência marcada `[HIPÓTESE]`; dúvidas listadas como perguntas objetivas
- [x] **Síntese (seção 14)** registra absorvido / descartado / adiado
- [x] **Complexidade (15.1)** preenchida
- [x] **Framework recomendado (15.2)** justificado com 2 dimensões decisivas
- [x] **Alternativas (15.3)** explicam por que NÃO o vizinho mais próximo
- [x] **Comando exato (15.4)** escrito
- [x] **Gatilhos (15.5)** de reclassificação listados
- [x] Pronto para alimentar PRD / INTENT / TaskCard
