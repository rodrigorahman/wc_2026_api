# Pré-Refinamento — Definição Inicial da Ideia

> Este documento é um artefato **intermediário**, anterior ao PRD / INTENT / TaskCard.
> Serve para reduzir ambiguidade cedo e preparar terreno para a próxima etapa do workflow.
>
> **Legenda:**
> - Linhas sem marcação = **FATO** (afirmado pelo usuário).
> - `[HIPÓTESE]` = inferência da skill que precisa ser validada.
> - `[DÚVIDA]` = ponto em aberto, detalhado na seção 13.

---

## 1. Metadados

- **Nome da Ideia / Feature**: Arquitetura Base — API do Álbum da Copa do Mundo
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-27
- **Versão**: v1
- **Status**: Refinado
- **Relacionados**: `docs/specs/arquitetura_base/prompt.md` (fonte da ideia)
- **Fonte da ideia**: `docs/specs/arquitetura_base/prompt.md`

---

## 2. Ideia Resumida (uma frase)

Construir a arquitetura base (fundação técnica + autenticação + seleção favorita) de uma API gRPC em Go para o produto de álbum de figurinhas da Copa do Mundo, sobre a qual os módulos de figurinhas e trocas serão construídos depois.

---

## 3. Problema

- **Qual é a dor real hoje?** Não existe base de código sobre a qual construir o produto — o repositório tem apenas `go.mod`. Sem uma fundação padronizada (layout, DI, config, logging, migrations, contrato gRPC, autenticação), cada módulo futuro reinventaria estrutura e divergiria.
- **Como o problema aparece no dia a dia?** [HIPÓTESE] Sem arquitetura base definida, o time perderia tempo decidindo padrões ad-hoc a cada módulo (figurinhas, trocas), gerando retrabalho e inconsistência.
- **Quem sente o impacto?** O time de desenvolvimento que vai construir os módulos seguintes sobre essa base.
- **Por que resolver agora?** É o ponto de partida do projeto — nenhum módulo de domínio pode começar antes da fundação e do mecanismo de autenticação existirem.

---

## 4. Objetivo Principal

- **Qual é o resultado esperado ao final?** Um projeto Go compilável e executável nos três sistemas operacionais (Windows, Linux, macOS), com servidor gRPC no ar, módulo de autenticação funcional (cadastro + login emitindo JWT validado pelo middleware) e o módulo de Seleção Favorita disponível como referência.
- **Qual mudança de comportamento/estado deve acontecer?** Passar de repositório vazio para uma fundação reutilizável que define os padrões (layout, DI, contrato gRPC, persistência, migrations) que todos os módulos futuros seguirão.

---

## 5. Público / Usuário Envolvido

- **Persona primária**: Time de desenvolvimento backend que construirá os módulos seguintes (figurinhas, trocas) sobre essa base.
- **Persona secundária**: [HIPÓTESE] Usuário final colecionador, que será o consumidor do fluxo de autenticação entregue aqui (cadastro/login) — embora o restante do produto que ele usa ainda não exista nesta versão.
- **Contexto de uso**: Binário compilado executado como serviço backend nos três SOs; frontend consome via gRPC.

---

## 6. Contexto

- **O que está acontecendo ao redor dessa ideia?** Início do projeto "Álbum da Copa do Mundo" — produto onde o usuário cadastra figurinhas da coleção e organiza as repetidas para trocas. Esta entrega é a fundação que precede esses módulos.
- **Existe solução parcial hoje?** Não. Repositório greenfield (apenas `go.mod`, `go 1.26.1`).
- **Há dependências conhecidas?** A definição do contrato gRPC (`.proto`) e do schema de banco impactarão diretamente o frontend e os módulos futuros — são decisões estruturantes (candidatas a ADR).

---

## 7. Escopo Inicial (rascunho)

Itens que **parecem** fazer parte da primeira versão (confirmados pelo usuário, salvo onde marcado):

- [ ] Estrutura de pastas no padrão go-standard-layout
- [ ] Injeção de dependências com uber-fx
- [ ] Configuração de ambiente com viper
- [ ] Logging estruturado com zap
- [ ] Servidor gRPC (protoc-gen-go + protoc-gen-go-grpc)
- [ ] Persistência em SQLite com queries via sqlc
- [ ] Migrations com golang-migrate; IDs em UUID v4
- [ ] Módulo de Usuário: cadastro e login (campos Nome Completo, E-mail, Senha com hash, Seleção favorita)
- [ ] Autenticação via golang-jwt + middleware go-grpc-middleware/auth
- [ ] Validação de dados com buf.validate (`protovalidate` — validação declarada via opções no `.proto`)
- [ ] Módulo de Seleção Favorita (campos ID, Nome) populado por seed via migration
- [ ] Testes com testify cobrindo o fluxo de autenticação
- [ ] Build cross-platform (Windows, Linux, macOS)

> Esta lista NÃO é definitiva — é um ponto de partida para o PRD/Tech Spec.

---

## 8. Fora do Escopo (rascunho)

Itens que **explicitamente** NÃO fazem parte dessa primeira versão:

- [ ] Módulo de Figurinhas (cadastro da coleção do usuário) — adiado
- [ ] Organização de figurinhas repetidas e sistema de trocas entre usuários — adiado
- [ ] CRUD administrativo de seleções (a tabela vem por seed fixo nesta versão)
- [ ] Pipeline de CI e meta formal de cobertura de testes (sucesso aqui = server sobe + auth funciona com testes do fluxo)
- [ ] Recuperação de senha / verificação de e-mail / refresh token (`[HIPÓTESE]` fora do escopo — ver Dúvida #3)

---

## 9. Restrições

- **Prazo / janela de tempo**: N/A — a ser definido (ver Dúvidas em Aberto #5).
- **Integrações obrigatórias**: Comunicação frontend ↔ backend exclusivamente via gRPC.
- **Regras de negócio / compliance / privacidade**: Senha deve ser armazenada com hash (nunca em texto plano). [HIPÓTESE] E-mail é identificador único do usuário.
- **Decisões já tomadas** (fora de negociação): Toda a stack está fixada pelo prompt — gRPC, SQLite, buf.validate (protovalidate), golang-jwt, go-grpc-middleware/auth, viper, zap, testify, sqlc, golang-migrate, protoc-gen-go(-grpc), go-standard-layout, UUID v4, uber-fx, build cross-platform. **Driver de conexão SQLite: `modernc.org/sqlite`** (pure-Go, sem CGO).

---

## 10. Aproveitamento de Capacidades Existentes

> Projeto greenfield — não há capacidades internas pré-existentes a reutilizar. Toda a "stack a reutilizar" é, na prática, a stack obrigatória declarada no prompt, que deve ser a base canônica para os módulos futuros.

- **Persistência**: SQLite via driver `modernc.org/sqlite` (pure-Go) + sqlc (queries) + golang-migrate (migrations). IDs UUID v4.
- **Autenticação / autorização**: golang-jwt (emissão/validação de token) + go-grpc-middleware/auth (interceptor de autenticação gRPC).
- **Mensageria / filas / sincronismo**: N/A — não previsto nesta versão.
- **Armazenamento de arquivos / objetos**: N/A — não previsto nesta versão.
- **Observabilidade / logging**: zap (logging estruturado).
- **Padrão arquitetural**: go-standard-layout + injeção de dependências com uber-fx.
- **Outros módulos internos reutilizáveis**: Configuração via viper; validação via buf.validate (protovalidate, declarada no `.proto`); contrato gRPC via protoc-gen-go/protoc-gen-go-grpc; testes com testify.

---

## 11. Premissas

- [HIPÓTESE] O frontend é um cliente gRPC separado, fora do escopo deste backend.
- [HIPÓTESE] E-mail é o identificador único de login do usuário (unicidade na tabela).
- "buf.validate" refere-se ao **protovalidate** (`github.com/bufbuild/protovalidate-go`) — pacote de validação que lê regras declaradas via opções `buf.validate.*` no `.proto`. Confirmado pelo usuário (Dúvida #1 resolvida).
- [HIPÓTESE] O cadastro do usuário exige uma `seleção favorita` já existente (FK para a tabela de seleções populada por seed).
- [HIPÓTESE] SQLite roda como arquivo local embarcado no binário (sem servidor de banco separado), coerente com a portabilidade cross-platform.

---

## 11.1 Inventário de Impacto (grep de consumidores)

**N/A — sem centralização de símbolo legado.** Embora a ideia introduza injeção de dependências (uber-fx), trata-se de projeto greenfield: não existem consumidores legados de construtores (`NewDB`, `NewAWS`, etc.) a refatorar. A DI nasce já como padrão. O grep de consumidores se aplicará apenas em features futuras que centralizem símbolos já em uso.

---

## 12. Riscos e Pontos de Atenção

- **Risco técnico ou operacional**: Mitigado — o driver SQLite será `modernc.org/sqlite` (pure-Go, sem CGO), o que elimina a principal dificuldade de build cross-platform (Windows/Linux/macOS). Decisão tomada (ver seção 9).
- **Risco de produto / aceitação**: Baixo nesta fase (entrega de fundação). O valor para o usuário final só aparece quando figurinhas/trocas existirem.
- **Risco de privacidade / segurança / compliance**: Manuseio de senha (hash + custo de algoritmo), segredo de assinatura do JWT (gestão via viper/env), expiração de token. Área crítica de autenticação/crypto.
- **Risco de escopo** (pode explodir facilmente?): Médio — a tentação de já incluir figurinhas/trocas ("já que estou na base") deve ser contida; escopo foi explicitamente limitado a fundação + auth + seleção.

---

## 13. Dúvidas em Aberto

1. ~~[DÚVIDA] Qual é o pacote exato referido por "uf.validate"?~~ **RESOLVIDA**: era erro de transcrição — o pacote é **buf.validate / protovalidate** (`github.com/bufbuild/protovalidate-go`), com regras declaradas via opções `buf.validate.*` no `.proto`.
2. [DÚVIDA] O JWT terá expiração e algum mecanismo de refresh, ou é token de longa duração sem refresh nesta versão?
3. [DÚVIDA] Recuperação de senha / verificação de e-mail entram em alguma versão futura próxima ou estão totalmente fora do roadmap inicial?
4. ~~[DÚVIDA] O driver SQLite será CGO ou pure-Go?~~ **RESOLVIDA**: `modernc.org/sqlite` (pure-Go, sem CGO). Decisão registrada na seção 9; ainda recomendável virar ADR para documentar o "porquê".
5. [DÚVIDA] Há prazo/janela de entrega para esta arquitetura base?
6. [DÚVIDA] O servidor gRPC precisa expor reflection / health-check / gRPC-Gateway (REST) nesta versão, ou apenas gRPC puro?

---

## 14. Separação: Fato × Hipótese × Dúvida (visão consolidada)

| Categoria | Item | Observação |
|-----------|------|------------|
| FATO      | Stack completa fixada (gRPC, SQLite, sqlc, golang-migrate, golang-jwt, go-grpc-middleware/auth, viper, zap, testify, uber-fx, go-standard-layout, UUID v4, build cross-platform) | Decisões fora de negociação |
| FATO      | Escopo v1 = fundação + auth (senha com hash) + seleção favorita (seed) | Confirmado pelo usuário |
| FATO      | Sucesso = server sobe nos 3 SOs + cadastro/login emitindo JWT validado, com testes | Confirmado pelo usuário |
| HIPÓTESE  | E-mail é identificador único de login | Precisa validar com: usuário |
| FATO      | Pacote de validação = **buf.validate / protovalidate** (`github.com/bufbuild/protovalidate-go`) | Resolvido pelo usuário (Dúvida #1) |
| HIPÓTESE  | SQLite como arquivo embarcado, sem servidor | Precisa validar com: decisão de driver (Dúvida #4) |
| FATO      | Driver SQLite = `modernc.org/sqlite` (pure-Go) | Resolvido pelo usuário |
| DÚVIDA    | Expiração/refresh de JWT | Bloqueia: design do módulo de auth |

---

## 14b. Brainstorm de Produto

> N/A aprofundado — a entrega é uma **fundação técnica com escopo deliberadamente fechado** pelo usuário (fundação + auth + seleção). O espaço de divergência de produto (figurinhas, trocas, social) pertence às features seguintes, não a esta base. Registramos abaixo apenas os ângulos que afetam decisões da própria fundação, para não sub-especificar a base.

### 14b.3 Features adjacentes de baixo custo (potencializadores da base)

- [ ] Health-check e reflection no gRPC server — barato, facilita testes e ferramentas (grpcurl). Ver Dúvida #6.
- [ ] Interceptor de logging (zap) + recovery já no boot — reusado por todos os módulos futuros.

### 14b.4 Riscos de produto não pensados

- **Risco 1**: Modelar a tabela de usuário sem prever os relacionamentos das figurinhas pode forçar migration disruptiva depois → mitigação: manter schema de usuário coeso e isolado, sem antecipar figurinhas, mas documentando a intenção.

### 14b.6 Síntese do brainstorm

- **Absorvido no escopo inicial (seção 7)**: nada novo além do confirmado nas perguntas.
- **Descartado com justificativa**: figurinhas e trocas (pertencem a features futuras, não à base).
- **Adiado para v2/v3**: health-check/reflection gRPC e gRPC-Gateway (decidir via Dúvida #6); recuperação de senha/refresh token.

---

## 15. Recomendação de Framework (Strategy Selector)

### 15.1 Sinais Observados

| Sinal | Valor detectado | Confirmação |
|---|---|---|
| S1 — # User Stories implícitas | 1-3 (cadastrar conta, fazer login, escolher seleção favorita) | inferido |
| S2 — Stakeholders | dev+1 (time de dev + usuário final no fluxo de auth) | inferido |
| S3 — Novidade técnica | greenfield | confirmado |
| S4 — Artefatos tocados (est.) | 10+ (layout completo, proto, migrations, providers fx, handlers, repos, testes) | inferido |
| S5 — Tempo estimado | 1-5 dias | inferido |
| S6 — Decisões arquiteturais novas | sim (layout, DI, contrato gRPC, driver SQLite, estratégia de migrations/JWT — todas candidatas a ADR) | inferido |
| S7 — Onboarding necessário | sim (outros devs construirão os módulos seguindo esta base) | inferido |
| S8 — Risco de regressão | médio (toca auth/crypto e fundação compartilhada por todos os módulos) | inferido |
| S9 — CRUD-pattern-repeat | não (não há pattern pré-existente; esta entrega é quem cria o pattern) | inferido |

### 15.2 Framework Recomendado

**Escolhido**: `SDD`

**Justificativa**: Três sinais disparam SDD de forma independente — **S3=greenfield** (projeto nasce aqui), **S6=sim** (a entrega define decisões arquiteturais estruturantes: layout, DI com uber-fx, contrato gRPC, driver SQLite, estratégia de migrations e de JWT) e **S7=sim** (é a base que todos os módulos futuros vão seguir, exigindo documentação formal). A formalização PRD + Tech Spec + ADRs é justamente o que evita que cada módulo subsequente reinvente padrões.

### 15.3 Alternativas Consideradas

**Por que não miniSpec** (alternativa mais próxima): miniSpec atende incrementos sobre base existente com `S6=não`. Aqui `S6=sim` e o projeto é greenfield — miniSpec não tem espaço para registrar as ADRs estruturantes (driver SQLite, contrato gRPC, padrão de DI) que precisam ser canônicas para o time. Economizaria tokens, mas deixaria dívida arquitetural na fundação, exatamente onde ela é mais cara de corrigir depois.

**Por que não TaskCard** (alternativa mais distante): TaskCard é para tarefa atômica de 1 dev, `S6=não`, `<1 dia`, 1-3 artefatos. Esta entrega atravessa 10+ artefatos, múltiplas camadas e decisões arquiteturais — colapsá-la em um TaskCard perderia rastreabilidade US→task e não comportaria as decisões de fundação.

### 15.4 Próximo Passo

Como `S6=sim` (múltiplas decisões arquiteturais estruturantes e transversais), registre primeiro as decisões evergreen como ADRs e depois gere o PRD que as referencia:

```bash
# 1. Registrar as decisões arquiteturais estruturantes (uma ADR por decisão relevante)
/adr-create "uso do driver modernc.org/sqlite (pure-Go) para portabilidade cross-platform"
/adr-create "padrao de injecao de dependencias com uber-fx e go-standard-layout"

# 2. Gerar o PRD da arquitetura base
/sdd-generate-prd "arquitetura base da API gRPC do album da Copa: fundacao + autenticacao por senha + selecao favorita"
```

### 15.5 Quando Reconsiderar a Recomendação

- **Manter/reforçar SDD** se durante a execução: o número de decisões arquiteturais crescer (ex.: necessidade de gRPC-Gateway, observabilidade distribuída), ou se surgirem novas personas (admin de seleções).
- **Downgrade para miniSpec** se: o usuário decidir cortar autenticação real (só esqueleto compilável) e adiar todas as decisões arquiteturais — aí a entrega vira um incremento de estrutura sem decisões a formalizar.

---

## 16. Checklist Final

- [x] Ideia resumida em uma frase clara
- [x] Problema descrito com dor concreta
- [x] Público identificado
- [x] Escopo inicial e fora de escopo delimitados
- [x] Toda inferência marcada `[HIPÓTESE]`
- [x] Seção de "Aproveitamento de Capacidades Existentes" preenchida
- [x] Nenhuma proposta de tecnologia nova sem justificativa registrada
- [x] Dúvidas em aberto listadas como perguntas objetivas
- [x] Brainstorm (seção 14b) tratado (escopo fechado — justificado)
- [x] Síntese 14b.6 registra absorvido/descartado/adiado
- [x] Tabela 15.1 de sinais preenchida (S1-S9)
- [x] Framework recomendado justificado com 2+ sinais decisivos (15.2)
- [x] Alternativas consideradas preenchidas (15.3)
- [x] Comando exato escrito (15.4)
- [x] Gatilhos de reclassificação listados (15.5)
- [x] Pronto para alimentar PRD / Tech Spec
