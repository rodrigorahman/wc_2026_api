# PRD -- Product Requirements Document (O QUE / POR QUÊ)

## 1. Metadados
- **Nome da Feature/Projeto**: Esqueci a senha — recuperação de acesso por e-mail
- **Responsável/Autor**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Versão**: v1
- **Status**: Draft
- **Relacionados**: `pre-refinement.md` (discovery desta feature), `arquitetura-base/v1` (domínio `auth` — Register/Login/GetMe), ADR-0003 (Sessão JWT 1h sem refresh), ADR-0006 (envio de e-mail transacional via Resend com interface no consumidor — capacidade de infra que esta feature consome), Glossário de Domínio (`User`, `Sessão`)

---

## 2. Contexto & Motivação
- **Qual problema ou dor existe hoje?** Um `User` que esquece a senha **não tem como recuperá-la**. A autenticação oferece apenas cadastro, login e consulta do próprio perfil. Sem recuperação, a conta fica inacessível para sempre.
- **Como funciona atualmente?** O usuário tenta logar, erra a senha repetidamente e não encontra saída — não há fluxo de recuperação. Acaba abandonando a conta (e potencialmente o app).
- **Por que isso precisa ser resolvido agora?** Recuperação de senha é capacidade básica de qualquer produto com login; sua ausência bloqueia usuários legítimos e gera perda de acesso ao progresso no Álbum da Copa 2026.
- **Quem sofre o impacto do problema?** O usuário final do Álbum da Copa 2026 que perde o acesso à própria conta.

---

## 3. Objetivo da Feature
- **O que se deseja alcançar?** Permitir que um `User` que esqueceu a senha recupere o acesso à conta sozinho, por e-mail, de forma segura.
- **Qual mudança de comportamento esta feature deve gerar?** A conta sai do estado "inacessível por esquecimento de senha" para "acesso restaurado", passando obrigatoriamente por uma **troca de senha** que invalida a credencial temporária recebida por e-mail.
- **Qual o resultado final esperado do ponto de vista do usuário?** O usuário solicita a recuperação, recebe por e-mail uma senha temporária, entra com ela e define uma nova senha definitiva — retomando o uso normal do app.

---

## 4. Escopo
### 4.1 O que está incluído (dentro do O QUE)
- [ ] Solicitação de recuperação informando apenas o e-mail da conta.
- [ ] Geração de uma **senha temporária pelo sistema** (aleatória e forte), enviada ao usuário por e-mail.
- [ ] Envio de **e-mail transacional em texto simples** contendo a senha temporária, sua validade e a instrução de troca.
- [ ] **A senha original permanece válida** durante a recuperação: a senha temporária é uma credencial **adicional**, não substitui a senha vigente. Só a troca definitiva substitui a senha.
- [ ] **Resposta sempre genérica** ao pedido de recuperação — não revela se o e-mail está ou não cadastrado.
- [ ] **Expiração da senha temporária em 15 minutos** a partir da emissão.
- [ ] **Troca obrigatória de senha** no login feito com a senha temporária.
- [ ] Sinalização, no login, de que há uma troca de senha pendente.
- [ ] Ação dedicada para o usuário trocar a senha (informando a senha atual e a nova).
- [ ] Invalidação da senha temporária assim que a nova senha é definida.
- [ ] **Envio de um e-mail de notificação** ao concluir a troca de senha, informando que a senha foi alterada e orientando a redefinição caso o usuário não reconheça a ação.

### 4.2 O que está explicitamente fora do escopo
- [ ] Recuperação por token/código de uso único ou OTP numérico — podado; optou-se por senha temporária.
- [ ] Login com escopo restrito (token que só autoriza a troca) — podado por over-engineering.
- [ ] **Limite de tentativas (rate limit) no pedido de recuperação** — adiado para v2; registrado como limitação conhecida.
- [ ] Falha de envio bloqueante (interromper o fluxo se o e-mail não sair) — podado; quebraria a postura anti-enumeração.
- [ ] Invalidação de Sessões (JWT) já emitidas após a troca de senha — fora do escopo; a Sessão é stateless e expira naturalmente em 1h (ADR-0003).
- [ ] Template/identidade visual do e-mail (logo, HTML) — adiado para v2.
- [ ] Troca obrigatória de senha para usuários do cadastro normal — não se aplica; o gatilho é apenas pós-recuperação.

---

## 5. Usuários & Personas
- **Quem é o usuário principal?** O usuário final do Álbum da Copa 2026, já cadastrado, que esqueceu a senha.
- **Qual é seu objetivo ao usar essa feature?** Reaver o acesso à própria conta sem depender de suporte.
- **Quais dores/dificuldades essa feature resolve pra ele?** Elimina o beco sem saída de "esqueci a senha e não tenho como entrar"; devolve o acesso ao progresso no app.

> **Persona secundária**: N/A — não há painel administrativo; o fluxo é inteiramente self-service.

### 5.1 Histórias de Usuário (User Stories)
- **US-01**: Como usuário que esqueceu a senha, quero solicitar a recuperação informando apenas meu e-mail para iniciar o processo de reaver o acesso.
- **US-02**: Como usuário, quero receber por e-mail uma senha temporária para conseguir entrar mesmo sem lembrar a senha original.
- **US-03**: Como usuário, quero ser informado, ao entrar com a senha temporária, de que preciso definir uma nova senha, para concluir a recuperação.
- **US-04**: Como usuário, quero definir uma nova senha definitiva, para retomar o uso normal da conta e invalidar a senha temporária.
- **US-05**: Como usuário, quero que o sistema não revele se meu e-mail está cadastrado ao pedir recuperação, para proteger minha privacidade.
- **US-06**: Como usuário, quero que a senha temporária deixe de funcionar após um curto período, para reduzir o risco de uso indevido caso o e-mail seja exposto.
- **US-07**: Como usuário que solicitou recuperação por engano (ou que teve a recuperação acionada por terceiro), quero que minha senha original continue funcionando, para não perder o acesso à conta sem ter trocado a senha de propósito.
- **US-08**: Como usuário, quero ser avisado por e-mail quando minha senha for alterada, para perceber e reagir caso a troca não tenha sido feita por mim.

---

## 6. Regras de Negócio (alto nível)

- **RN1** — A recuperação se dá exclusivamente por e-mail; o sistema gera uma senha temporária aleatória e forte. O usuário **não** informa nenhuma senha no momento do pedido.
- **RN2** — A senha temporária é válida por **15 minutos** a partir da emissão; após esse prazo, deixa de ser aceita.
- **RN3** — O pedido de recuperação responde **sempre de forma genérica**, independentemente de o e-mail existir ou não, para não revelar a existência de contas.
- **RN4** — Uma falha no envio do e-mail **não altera** a resposta ao usuário (comportamento best-effort); a falha é apenas registrada internamente.
- **RN5** — Ao entrar com a senha temporária, o usuário recebe a indicação de que há uma **troca de senha obrigatória pendente**.
- **RN6** — A troca obrigatória de senha é disparada **somente após uma recuperação**; o cadastro self-service permanece inalterado.
- **RN7** — A senha do usuário e a senha temporária **nunca** são registradas em log ou exibidas, exceto no corpo do e-mail destinado ao próprio usuário.
- **RN8** — Definir a nova senha **invalida** a senha temporária e remove a pendência de troca.
- **RN9 (limitação conhecida)** — A troca de senha **não invalida** Sessões (JWT) já emitidas; elas expiram naturalmente em até 1h (ADR-0003).
- **RN10** — A geração da senha temporária **não invalida a senha original**: durante a validade da temporária, ambas são aceitas no login. Logar com a original mantém o acesso normal (sem troca obrigatória); logar com a temporária dispara a troca obrigatória. Apenas a troca definitiva substitui a senha vigente.
- **RN11** — Ao concluir a troca de senha, o sistema envia (best-effort) um e-mail notificando a alteração e orientando o usuário a redefinir a senha caso não reconheça a ação. A falha no envio desse e-mail **não bloqueia** a troca (é apenas registrada internamente) e a senha/temporária **nunca** aparecem nesse e-mail.

---

## 7. Fluxo Comportamental (não técnico)

### 7.1 Fluxo Principal
1. O usuário, sem conseguir lembrar a senha, aciona "esqueci a senha" e informa o e-mail da conta.
2. O sistema responde com uma mensagem genérica ("se o e-mail estiver cadastrado, enviaremos as instruções"), sem confirmar a existência da conta.
3. Existindo a conta, o sistema gera uma senha temporária e a envia por e-mail (texto simples), com aviso de validade de 15 minutos e instrução de troca.
4. O usuário entra no app usando a senha temporária recebida.
5. O sistema autentica o usuário e sinaliza que há uma troca de senha obrigatória pendente.
6. O usuário define uma nova senha definitiva (informando a senha temporária atual e a nova).
7. O sistema confirma a troca, invalida a senha temporária, remove a pendência e **envia um e-mail notificando a alteração da senha**; o usuário segue usando o app normalmente.

### 7.2 Fluxos Alternativos
- **E-mail não cadastrado**: o sistema responde exatamente a mesma mensagem genérica e não envia e-mail algum — sem indicar que a conta não existe.
- **Falha no envio do e-mail**: o sistema mantém a resposta genérica ao usuário e registra a falha internamente; o usuário não percebe diferença. Vale tanto para o e-mail da senha temporária quanto para o de notificação de troca.
- **Usuário lembra a senha original**: mesmo após solicitar recuperação, o usuário pode entrar normalmente com a senha original; o acesso é concedido sem troca obrigatória e a senha temporária simplesmente expira sem uso.
- **Senha temporária expirada**: se o usuário tentar entrar com a senha temporária após 15 minutos, o acesso é negado; ele precisa solicitar uma nova recuperação.
- **Cadastro normal**: usuários que se cadastram escolhendo a própria senha não passam por troca obrigatória.

---

## 8. Critérios de Aceite (O QUE deve acontecer)
- [ ] **CA-01**: DADO um e-mail cadastrado QUANDO o usuário solicita recuperação ENTÃO o sistema gera uma senha temporária, envia o e-mail e responde com a mensagem genérica.
- [ ] **CA-02**: DADO um e-mail **não** cadastrado QUANDO o usuário solicita recuperação ENTÃO o sistema responde com a **mesma** mensagem genérica e não envia nenhum e-mail.
- [ ] **CA-03**: DADO que o envio do e-mail falha QUANDO o usuário solicita recuperação ENTÃO o sistema ainda responde com a mensagem genérica e registra a falha internamente.
- [ ] **CA-04**: DADO uma senha temporária válida (dentro de 15 min) QUANDO o usuário faz login com ela ENTÃO o acesso é concedido e o sistema sinaliza troca de senha obrigatória pendente.
- [ ] **CA-05**: DADO uma senha temporária emitida há mais de 15 minutos QUANDO o usuário tenta logar com ela ENTÃO o acesso é negado.
- [ ] **CA-06**: DADO um usuário com troca obrigatória pendente QUANDO ele informa a senha temporária atual e uma nova senha ENTÃO a nova senha é aceita e a pendência é removida.
- [ ] **CA-07**: DADO que o usuário concluiu a troca QUANDO ele tenta logar novamente com a senha temporária ENTÃO o acesso é negado (a temporária foi invalidada).
- [ ] **CA-08**: DADO um usuário que se cadastrou pelo fluxo normal QUANDO ele faz o primeiro login ENTÃO **nenhuma** troca obrigatória é exigida.
- [ ] **CA-09**: DADO qualquer etapa do fluxo QUANDO o sistema registra logs ENTÃO a senha e a senha temporária **nunca** aparecem nos registros.
- [ ] **CA-10**: DADO um usuário que solicitou recuperação mas lembra a senha original QUANDO ele faz login com a senha original (a temporária ainda válida ou já expirada) ENTÃO o acesso é concedido normalmente e **nenhuma** troca obrigatória é exigida.
- [ ] **CA-11**: DADO um usuário com troca pendente QUANDO ele conclui a troca informando a senha temporária e a nova senha ENTÃO o sistema envia um e-mail notificando a alteração da senha.
- [ ] **CA-12**: DADO que o envio do e-mail de notificação de troca falha QUANDO o usuário conclui a troca ENTÃO a troca é efetivada normalmente e a falha é apenas registrada internamente.

---

## 9. Restrições & Considerações
- **Dependência de serviço externo de envio de e-mail transacional** — capacidade nova no projeto; o COMO da integração e a política de configuração da credencial (ex.: obrigatória no boot × opcional em desenvolvimento) ficam para a etapa técnica. `[DELEGAR_TECH_SPEC]`
- **Janela de 15 minutos é apertada** para o ciclo "receber e-mail → logar → trocar"; atraso na entrega pode expirar a senha antes do uso. Decisão travada para a v1; ampliar (ex.: 30–60 min) se houver atrito medido.
- **Sem limite de tentativas (rate limit)** no pedido de recuperação na v1 — risco de flood de e-mails reconhecido e adiado para v2.
- **Senha temporária em texto trafega e persiste no inbox** do usuário — ônus assumido conscientemente, mitigado por expiração curta + troca obrigatória + proibição de logar a senha.
- **Coexistência de credenciais durante a recuperação** — manter a senha original válida ao lado da temporária (RN10) implica duas credenciais aceitas no login durante a janela de 15 min. Trade-off aceito: leve aumento de superfície em troca de eliminar o lockout/griefing por pedido acidental ou malicioso.
- **Dois e-mails transacionais** — a feature passa a enviar dois e-mails (senha temporária + notificação de troca, RN11), ambos best-effort e em texto simples, via a capacidade de e-mail transacional da ADR-0006. Template/identidade visual segue adiado para v2.
- **Sessões (JWT) são stateless** — a troca de senha não as invalida; expiram em 1h (ADR-0003).
- **Idioma**: mensagens ao usuário (e-mail e respostas) em pt-BR; identificadores de domínio seguem o glossário (`User`, `Sessão`).

---

## 10. Métricas de Sucesso
- **Taxa de conclusão do fluxo**: % de pedidos de recuperação que chegam até a troca de senha bem-sucedida.
- **Tempo médio entre emissão e troca**: indica se a janela de 15 min é confortável (sinal para revisão da janela).
- **Taxa de expiração antes do uso**: % de senhas temporárias que expiram sem login — alto valor sugere janela curta demais ou atraso de entrega.
- **Volume de pedidos de recuperação**: monitorar abuso/flood (insumo para priorizar o rate limit da v2).
- **Taxa de falha de envio de e-mail**: saúde da integração com o provedor externo.

---

## 11. Roadmap / Fases
- **Fase 1 (v1 — esta feature)**: fluxo completo de recuperação por senha temporária — pedido com resposta genérica, geração e envio por e-mail (texto simples), expiração de 15 min, login com sinalização de troca pendente, troca obrigatória e invalidação da temporária.
- **Fase 2**: limite de tentativas (rate limit) no pedido; possível ampliação da janela de expiração; template/identidade visual do e-mail.
- **Fase 3**: ampliação para outros e-mails transacionais (ex.: confirmação de cadastro, notificações), caso o produto demande.

---

## 12. Rastreabilidade de User Stories

| User Story | Descrição Resumida | Critério de Aceite Relacionado |
|------------|-------------------|-------------------------------|
| US-01 | Solicitar recuperação informando o e-mail | CA-01, CA-02 |
| US-02 | Receber senha temporária por e-mail | CA-01, CA-03 |
| US-03 | Ser avisado da troca obrigatória ao logar | CA-04 |
| US-04 | Definir nova senha e invalidar a temporária | CA-06, CA-07 |
| US-05 | Pedido não revela existência do e-mail | CA-02 |
| US-06 | Senha temporária expira em curto prazo | CA-05 |
| US-07 | Senha original continua válida durante a recuperação | CA-10 |
| US-08 | Aviso por e-mail quando a senha é alterada | CA-11, CA-12 |

> Regras transversais de segurança/escopo cobertas adicionalmente por: CA-08 (gatilho só pós-reset, ligado à RN6) e CA-09 (não logar senha, ligado à RN7).

---

## 13. Checklist Final
- [x] PRD descreve apenas O QUE / POR QUÊ
- [x] Escopo fechado
- [x] User Stories definidas e numeradas (US-XX)
- [x] Critérios de aceite claros
- [x] Tabela de rastreabilidade preenchida
- [x] Pronto para criar o TECH_SPEC (COMO)
