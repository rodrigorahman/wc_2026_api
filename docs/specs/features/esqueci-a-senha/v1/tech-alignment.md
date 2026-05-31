# Tech Alignment — Esqueci a senha (recuperação de acesso por e-mail)

## 1. Metadados

- **Feature**: `esqueci-a-senha`
- **Versão**: `v1`
- **Framework**: SDD
- **Variante**: `backend`
- **Documento de definição**: `docs/specs/features/esqueci-a-senha/v1/prd.md`
- **Discovery lido**: `pre-refinement.md` (Tree of Thought completo, seção 11 "Decisões já tomadas")
- **ADRs consultadas**: ADR-0006 (envio de e-mail transacional via Resend com interface no consumidor — capacidade de infra reusada por esta feature), ADR-0003 (JWT HS256 + bcrypt + TTL 1h sem refresh), ADR-0002 (DI por uber-fx, interface no consumidor), ADR-0001 (driver pure-Go, sem CGO), ADR-0005 (idioma)
- **Data**: 2026-05-30
- **Status**: Decidido

---

## 2. Contexto técnico

O domínio `auth` (`Register`/`Login`/`GetMe`) ganha um fluxo de recuperação de credencial sobre gRPC, sem página web hospedada. Uma decisão desta feature é **transversal** e foi canonizada à parte: o **envio de e-mail transacional via Resend com interface `EmailSender` no consumidor** (ADR-0006) — capacidade de infra que outras features reusarão. Tudo o mais aqui é **feature-scoped**: o *design da recuperação* (recuperação por senha temporária aleatória em vez de token/OTP, coexistência de credenciais, expiração de 15 min, troca obrigatória, notificação de troca) é decisão específica desta feature e vive **neste documento**, consumindo a capacidade de e-mail da ADR-0006.

A invariante de segurança que governa todo o fluxo é a **postura anti-enumeração** — o sistema não pode revelar, por resposta nem por timing, se um e-mail está cadastrado. O Login já materializa essa postura com mensagem genérica idêntica e equalização de tempo via comparação bcrypt contra um hash dummy. Qualquer novo ponto de entrada (o pedido de recuperação) herda essa invariante e **não pode introduzir um canal lateral** que a viole.

A escolha de **recuperação por senha temporária** (sobre token/código de uso único ou OTP) foi travada na seção 11 do pré-refinamento — decisão feature-scoped registrada em D0 abaixo. Sobre ela, este documento detalha o *como* (D1–D5), incluindo duas decisões do dono do produto: o **modelo de coexistência de credenciais** (D4) — a senha temporária não sobrescreve a senha vigente — e a **notificação de troca por e-mail** (D5).

---

## 3. Soluções técnicas decididas

### D0 — Mecanismo de recuperação: senha temporária (decisão direta, feature-scoped)

**Decisão direta (travada no pré-refinamento, seção 11).** A recuperação se dá por **senha temporária** gerada pelo backend e enviada por e-mail — não por token/código de uso único nem OTP numérico. É decisão **feature-scoped** (só o fluxo de recuperação do `auth` a usa; não é transversal), por isso vive aqui e não em ADR.

Alternativas avaliadas e rejeitadas (registro do pré-refinamento):
- **Token/código de reset de uso único** — o e-mail levaria um código e o usuário definiria a senha definitiva direto, sem senha válida no inbox (mais seguro). _Rejeição:_ o dono do produto optou pela senha temporária; o token exigiria tabela + RPC dedicados.
- **OTP numérico de 6 dígitos** — variante do token com código curto (UX mobile). _Rejeição:_ mesma do token.

**Trade-off aceito (conscientemente)**: a senha temporária em texto trafega e **persiste no inbox** do usuário, sendo credencial válida até a troca/expiração — superfície de exposição inerente, mitigada por expiração curta (15 min, D3/janela), troca obrigatória e coexistência (D4). As decisões D1–D5 detalham o *como* desta escolha.

### D1 — Política de configuração da credencial Resend

> **Promovida à ADR-0006.** Esta decisão nasceu aqui (era o `[DELEGAR_TECH_SPEC]` do PRD §9) mas, por ser **transversal** à capacidade de e-mail, foi canonizada na ADR-0006 como política da capacidade. Mantida abaixo como registro da proposta e da sua aplicação nesta feature; a fonte de verdade é a ADR.

**Por que decidir**: o Resend é a primeira integração externa com credencial do projeto. O boot precisa de uma política clara: a ausência da chave aborta o servidor (como `JWT_SECRET`) ou degrada graciosamente? Decisão técnica de ops/boot, não de produto. *(corresponde ao `[DELEGAR_TECH_SPEC]` do PRD §9 e à DÚVIDA #2 do pré-refinamento)*

**Solução recomendada: D1.b — fail-fast em produção + sender no-op em desenvolvimento.** Em produção a credencial é obrigatória e sua ausência aborta o boot, espelhando a política `JWT_SECRET` já vigente (`config.Load` retorna `error`, o composition root aborta antes de servir). Quando `APP_ENV=development`, a ausência da chave faz o módulo fazer bind de um `EmailSender` no-op que **loga** o e-mail em vez de enviá-lo — reusa o `IsDevelopment()` que já existe e a mesma interface que já será fakeada em testes. Não loga a senha temporária (RN7).

Caminhos viáveis avaliados:
- **D1.a — sempre obrigatória (fail-fast incondicional)**: a chave é exigida em qualquer ambiente.
  - _Exemplo:_ subir o servidor localmente sem `RESEND_API_KEY` aborta, igual a rodar sem `JWT_SECRET`.
  - _Prós:_ uma só regra, zero ramificação por ambiente; impossível subir "meio configurado".
  - _Contras:_ todo desenvolvedor local precisa de uma conta/chave Resend só para tocar em qualquer parte da API; atrito alto para uma capacidade que o dev raramente exercita.
  - _Viabilidade:_ reusa o padrão fail-fast de `config.go` exatamente; nenhuma novidade.
- **D1.b — fail-fast em prod + no-op em dev** (recomendada): degradação só no ambiente de desenvolvimento.
  - _Exemplo:_ produção sem a chave aborta; `APP_ENV=development` sem a chave loga `"[email no-op] to=... subject=..."` e segue.
  - _Prós:_ produção continua segura por construção; dev não exige conta Resend; o no-op é o mesmo formato de fake já planejado para testes.
  - _Contras:_ uma ramificação por ambiente no wiring; risco teórico de o no-op mascarar um problema de integração até produção — mitigado porque o caminho de envio real é coberto por teste.
  - _Viabilidade:_ reusa `Config.IsDevelopment()` (já existe) + interface `EmailSender` (já no escopo); requer `RESEND_API_KEY` novo na `Config`.

**Trade-off aceito**: aceita-se uma ramificação por ambiente no bind do `EmailSender` em troca de manter produção fail-fast **e** remover o atrito de obrigar credencial Resend no dia a dia do desenvolvedor.

### D2 — Mecanismo de integração com o Resend

**Por que decidir**: o Resend expõe uma API HTTP. Há duas formas de consumi-la atrás da interface `EmailSender`, com diferença de superfície de dependência — decisão arquitetural de reuso, que a própria ADR-0006 delega ao consumidor.

**Decisão (definida pelo dono da técnica): D2.b — SDK oficial `resend-go`.** A implementação concreta de `EmailSender` delega a chamada ao client do SDK. A dependência nova foi avaliada como **justificada**: encapsula o contrato da API (marshaling, endpoints, mapeamento de erro), é mantida pelo fornecedor e é **pure-Go** — não introduz toolchain C, preservando a build sem CGO (ADR-0001). O acoplamento ao Resend já existe por decisão de projeto (ADR-0006); o SDK apenas o materializa de forma suportada.

Caminhos viáveis avaliados:
- **D2.b — SDK oficial `resend-go`** (escolhida): usa a biblioteca do fornecedor.
  - _Exemplo:_ `resendSender.Send(to, subject, body)` delega ao client do SDK; falha de envio é mapeada para erro logado (best-effort, ADR-0006).
  - _Prós:_ menos boilerplate; contrato da API mantido pelo fornecedor; pure-Go (compatível com sem-CGO); caminho suportado para evoluções futuras (fase 3 — múltiplos e-mails transacionais).
  - _Contras:_ uma dependência nova (e sua árvore transitiva); acopla o ritmo de atualização ao fornecedor.
  - _Viabilidade:_ pure-Go, não conflita com ADR-0001; isolada atrás de `EmailSender`, então um eventual fake/no-op em testes e dev (D1) não toca o SDK.
- **D2.a — cliente fino sobre `net/http` (stdlib)**: implementação concreta usa só a stdlib.
  - _Exemplo:_ a implementação monta o POST, injeta o header de autorização e mapeia falha de transporte/status.
  - _Prós:_ nenhuma dependência transitiva nova; controle total sobre timeout e mapeamento de erro.
  - _Contras:_ boilerplate (marshaling, timeout, mapeamento de status) a manter e testar à mão; reimplementa o que o SDK já entrega.
  - _Viabilidade:_ reusa stdlib + `logger` (zap); rejeitada por reinventar o cliente que o fornecedor mantém.

**Trade-off aceito**: aceita-se adicionar a dependência `resend-go` (e sua árvore transitiva) em troca de eliminar o boilerplate de um cliente HTTP feito à mão e usar o caminho suportado pelo fornecedor — coerente com a expansão prevista para múltiplos e-mails transacionais (roadmap fase 3). A interface `EmailSender` (ADR-0006) mantém o SDK isolado do domínio.

### D3 — Envio do e-mail: síncrono × assíncrono no pedido de recuperação

**Por que decidir**: este é o ponto de maior peso de segurança e o que ADR/PRD **não** trataram. A invariante anti-enumeração (RN3) e a postura best-effort (RN4) impõem que a resposta ao pedido seja idêntica e indistinguível — inclusive em **tempo** — exista ou não a conta. Um envio síncrono dentro do RPC viola isso: o caminho do e-mail existente chama o Resend (lento, sujeito à latência do provedor), enquanto o inexistente retorna imediatamente — um canal lateral de timing que permite enumerar contas, exatamente o que o Login já se esforça para fechar.

**Solução recomendada: D3.b — dispatch assíncrono (fire-and-forget).** O RPC valida o pedido, responde a mensagem genérica **imediatamente** em ambos os caminhos (conta existente ou não) e dispara o envio do e-mail em background; a falha de envio é logada ali, sem afetar a resposta já entregue. Resolve as três forças de uma vez: RN3 (a resposta não depende do tempo de chamada ao Resend), RN4 (best-effort — falha só logada) e o risco operacional mapeado no pré-refinamento §12 (UX desacoplada da disponibilidade do provedor).

Caminhos viáveis avaliados:
- **D3.a — envio síncrono dentro do RPC**: o RPC só responde após o Resend retornar.
  - _Exemplo:_ pedido para e-mail cadastrado bloqueia ~Xms aguardando o Resend; pedido para e-mail inexistente responde na hora.
  - _Prós:_ sem concorrência; o e-mail terminou (ou falhou) antes de responder.
  - _Contras:_ **vaza enumeração por timing** (caminho existente é mais lento); acopla a latência da resposta à do Resend. Para ficar seguro exigiria equalização de tempo no caminho inexistente (simular o custo do envio) — mais complexo do que o assíncrono.
  - _Viabilidade:_ simples de escrever, mas reintroduz o canal lateral que `auth-security` (equalização de timing) trabalha para eliminar — conflita com a invariante do projeto.
- **D3.b — dispatch assíncrono** (recomendada): resposta imediata e idêntica; envio em background.
  - _Exemplo:_ ambos os caminhos respondem "se o e-mail estiver cadastrado, enviaremos as instruções" sem aguardar o Resend; o envio ocorre depois e loga falha se houver.
  - _Prós:_ fecha o timing por construção; best-effort natural; resiliente à indisponibilidade do Resend.
  - _Contras:_ sem confirmação de envio na resposta (que é justamente a postura best-effort desejada); exige cuidado de não vazar contexto cancelado/segredo no goroutine de background.
  - _Viabilidade:_ reusa `logger` para a falha; nenhuma infra de fila nova — fire-and-forget simples, não um worker/queue (isso seria over-engineering para v1).

**Trade-off aceito**: abre-se mão da confirmação de envio na resposta do RPC — coerente com o best-effort exigido (RN4) — em troca de eliminar o canal lateral de timing e desacoplar a UX da disponibilidade do Resend. **Não** se introduz fila/worker: o dispatch é um disparo em background simples; reavaliar só se a fase 2/3 exigir reentrega.

### D4 — Coexistência da senha original com a temporária (a temporária não sobrescreve a senha vigente)

**Decisão direta (definida pelo dono do produto).** A senha original do usuário **permanece válida** até que ele conclua a troca definindo a nova senha. A senha temporária é uma credencial **adicional**, de uso paralelo e tempo limitado (15 min) — não uma substituição da senha vigente.

Consequências da decisão:
- **Login** passa a aceitar a senha original (acesso normal, **sem** troca forçada) **ou** a senha temporária dentro da janela (acesso **+ sinalização de troca obrigatória**). Quem casa com nenhuma das duas recebe o erro genérico de sempre.
- Apenas a temporária expira (15 min); a senha original **nunca** expira por conta da recuperação.
- `ChangePassword` passa a ser o **único** ponto que substitui a senha vigente: define a nova senha definitiva, invalida a temporária e remove a pendência.
- Um pedido de recuperação **acidental ou malicioso não derruba** o acesso do usuário legítimo — quem conhece a senha original continua entrando, e a temporária vai apenas para o e-mail do dono da conta (que o atacante não controla).

**Por que importa (e não é mero detalhe)**: define o *modelo de credencial* da recuperação — "senha vigente + credencial temporária coexistentes durante a janela" em vez de uma sobrescrita. Muda a lógica do Login (dois caminhos de comparação) e o significado da expiração. É decisão **feature-scoped** (só o fluxo de recuperação a exige) — por isso registrada aqui, não em ADR.

**Substitui a direção A1 do pré-refinamento**: a A1 propunha a temporária **sobrescrevendo** a senha (hash em `password_hash`). Esta decisão a substitui — a temporária vive separada da senha vigente. (A versão original da ADR-0006, antes da refocagem, chegou a descrever a sobrescrita; isso não é mais matéria de ADR, que agora trata só do e-mail transacional.)

**Trade-off aceito**: durante a janela de 15 min coexistem duas credenciais válidas (original + temporária) — leve aumento de superfície — em troca de eliminar o lockout/griefing por pedido acidental ou malicioso e preservar o acesso do usuário legítimo. O Login ganha um segundo branch de comparação (testar a temporária quando a original não casa), exigindo atenção à **equalização de timing** já praticada (ver §6).

### D5 — Notificação de senha alterada por e-mail

**Decisão direta (definida pelo dono do produto).** Ao concluir a troca de senha (`ChangePassword`), o sistema envia ao usuário um e-mail informando que a senha **foi alterada** e orientando-o a redefini-la **caso não reconheça** a alteração (sinal de comprometimento da conta).

Forma técnica (reuso, sem novidade arquitetural):
- Reusa a capacidade `EmailSender` (ADR-0006) e a postura de **dispatch assíncrono best-effort** (D3): a troca de senha **não falha** se o e-mail não sair; a falha é apenas logada.
- É o **segundo** e-mail transacional da feature (o primeiro é a senha temporária). Texto simples em pt-BR; **nunca** inclui senha (RN7) — só o aviso e a orientação.
- Aqui **não** há postura anti-enumeração a proteger: o usuário já está autenticado e trocando a própria senha, então o gatilho é direto (sem equalização de timing).

**Dimensão de produto/escopo (refletida no PRD)**: esta notificação é um **segundo** e-mail transacional (o primeiro é a senha temporária) — portanto uma **adição de escopo**. Já refletida no PRD: item de escopo (§4.1), RN11, CA-11/CA-12 e US-08. Tecnicamente já está coberta por D2/D3.

---

## 4. Fronteira com ADR

**A única decisão transversal desta feature é o envio de e-mail transacional** — canonizada na **ADR-0006** (*"Envio de e-mail transacional via provider externo (Resend) com interface no consumidor"*), que registra a interface `EmailSender`, o provider Resend (HTTP pure-Go), a postura best-effort padrão e a política de credencial (fail-fast em prod / no-op em dev, promovida de D1). Toda feature futura que enviar e-mail herda essa ADR.

**As demais decisões são feature-scoped e ficam neste documento** — falham no critério C1 (transversal) por serem específicas do fluxo de recuperação do `auth`:
- **D0** — recuperação por senha temporária (sobre token/OTP).
- **D2** — mecanismo concreto de chamada ao Resend (decidido: SDK `resend-go`), que a própria ADR-0006 delega ao consumidor.
- **D3** — envio assíncrono + equalização de timing no pedido de recuperação.
- **D4** — coexistência de credenciais (a temporária não sobrescreve a senha vigente).
- **D5** — e-mail de notificação de troca de senha.

**Reflexos no PRD (executados):** D4 (RN10, CA-10, US-07, escopo, fluxo alternativo) e D5 (RN11, CA-11/CA-12, US-08, escopo) refletidos; §9 reflete o trade-off da coexistência e os dois e-mails.

**Nenhuma nova ADR a criar.** D4/D5 são aplicações da capacidade da ADR-0006, não decisões transversais novas.

---

## 5. Restrições e invariantes técnicas

Qualquer implementação desta feature **deve** respeitar:

- **Anti-enumeração (resposta e timing)** — o pedido de recuperação responde sempre a mesma mensagem genérica, e o tempo de resposta não pode depender da existência da conta (base da decisão D3). Espelha a equalização de timing já feita no Login.
- **Coexistência de credenciais durante a recuperação (D4)** — a senha temporária **não** sobrescreve a senha vigente; ambas são válidas durante a janela de 15 min. Só a temporária expira; `ChangePassword` é o único ponto que substitui a senha vigente.
- **Sem tabela dedicada de resets** — o estado de recuperação vive em colunas de `users` (credencial temporária + expiração + flag de troca); não se cria tabela nova.
- **Troca de senha notificada por e-mail (D5)** — `ChangePassword` dispara um e-mail best-effort de "senha alterada"; segundo e-mail transacional da feature (refletido no PRD: RN11, CA-11/CA-12, US-08).
- **Senha temporária via fonte criptográfica** — geração aleatória e forte usando fonte segura (`crypto/rand`); nunca um gerador previsível.
- **Senha e senha temporária nunca logadas** (RN7 / `auth-security`) — exceto no corpo do e-mail destinado ao próprio usuário. O no-op sender de dev (D1) loga metadados, nunca o segredo.
- **bcrypt cost 12** com `compareFunc`/`clock` injetados como boundaries testáveis — reuso do padrão do `AuthService`.
- **Expiração determinística via `clock.Clock` injetado** — nunca `time.Now()` direto; a janela de 15 min é checada com o clock injetado.
- **Sentinelas de erro com `errors.Is`**, service retorna `status.Error(codes.X)`, handler é mapper puro — convenções de `grpc-layers`.
- **Persistência sqlc parametrizada**, driver `modernc.org/sqlite`, build `CGO_ENABLED=0` (ADR-0001) — vale também para o SDK `resend-go` (D2), que é pure-Go e não introduz toolchain C; `make build-all` deve permanecer verde após a adição da dependência.
- **Idioma** — identificadores de schema/Go/proto em inglês (ADR-0005); mensagens ao usuário e corpo do e-mail em pt-BR.
- **JWT stateless sem refresh (ADR-0003)** — a troca de senha **não** invalida tokens já emitidos; expiram em até 1h (limitação conhecida RN9).
- **Sem rate limit na v1** — ausência de limite no pedido de recuperação é limitação conhecida e aceita, adiada para v2.

---

## 6. Pontos em aberto

**Decisões técnicas a critério do arquiteto do TECH_SPEC:**

- **Decomposição do módulo** — estender o `AuthService` existente com os novos métodos × extrair um service dedicado de recuperação no mesmo domínio `auth`. Decisão de organização de pacotes/responsabilidade, deixada deliberadamente para o TECH_SPEC (limite inferior desta skill). Sinal a considerar: o `AuthService` ganharia uma dependência de `EmailSender` que `Register`/`Login`/`GetMe` não usam.
- **Equalização de timing no Login e no pedido de recuperação** — com a coexistência de credenciais (D4), o Login passa a ter um segundo branch de comparação (testar a temporária quando a original não casa); avaliar a equalização de tempo entre os caminhos (original casa / temporária casa / nenhuma casa) à luz da técnica de hash dummy já existente. No pedido de recuperação, mesmo com dispatch assíncrono (D3.b), avaliar se o lookup síncrono do e-mail introduz diferença mensurável entre conta existente e inexistente. Detalhe de implementação a confirmar no TECH_SPEC.

**Reconciliações upstream (executadas em 2026-05-31):**

- **ADR-0006** — **refocada** na decisão transversal real (envio de e-mail transacional via Resend com interface no consumidor) e renomeada; o design de recuperação por senha temporária (D0, D4, D5) saiu da ADR e passou a viver aqui como feature-scoped. INDEX regenerado.
- **PRD** — atualizado para D4 (RN10, CA-10, US-07, item de escopo, fluxo alternativo) e D5 (RN11, CA-11/CA-12, US-08, item de escopo); §9 reflete o trade-off da coexistência e os dois e-mails.

**Dependências de produto não resolvidas:** nenhuma. O PRD fechou janela (15 min), formato dos e-mails (texto simples, sem template) e gatilho da troca (só pós-reset).
