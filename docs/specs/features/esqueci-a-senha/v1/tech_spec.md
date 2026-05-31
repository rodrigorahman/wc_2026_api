# TECH_SPEC -- Especificação Técnica (Backend)

## 1. Identificação
- **Feature/Projeto**: Esqueci a senha — recuperação de acesso por e-mail
- **Variante**: backend
- **Stack**: Go (uber-fx + modernc.org/sqlite pure-Go + sqlc + gRPC)
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-31
- **Status**: Draft
- **PRD Relacionado**: `docs/specs/features/esqueci-a-senha/v1/prd.md` · Tech Alignment: `tech-alignment.md`

---

## 2. Resumo Técnico da Solução

Estende o domínio `auth` (gRPC) com recuperação de acesso por **senha temporária**, sem tabela dedicada: a `AuthService` ganha `RequestPasswordRecovery` (público) e `ChangePassword` (autenticado), e o `Login` ganha um segundo branch de comparação e o campo `password_change_required` na resposta. O estado de recuperação vive em duas colunas nullable de `users` (`temp_password_hash`, `temp_password_expires_at`). O envio de e-mail usa a capacidade `EmailSender` (ADR-0006, SDK `resend-go` pure-Go) em **dispatch assíncrono best-effort** — a resposta ao pedido é imediata e idêntica para qualquer e-mail (anti-enumeração por resposta e por timing). A senha temporária coexiste com a senha original (D4); apenas `ChangePassword` substitui a senha vigente, validando estritamente a temporária e invalidando-a ao concluir, com e-mail de notificação best-effort (D5).

---

## 3. Arquitetura da Solução

### 3.1 Visão Geral

Mantém a arquitetura em camadas por domínio (`handler → service → repository`) e o wiring por `fx.Module` (ADR-0002). Nenhum componente estrutural novo no grafo além da capacidade de e-mail:

```
gRPC client
   │  (cadeia fixa: recovery → logging → protovalidate → auth JWT)
   ▼
AuthHandler (mapper puro)
   │  RequestPasswordRecovery / Login / ChangePassword
   ▼
AuthService (regra de negócio + status.Error)
   ├── UserRepository      (SQLite/sqlc)  — SetTempPassword, UpdatePassword, GetUserBy*
   ├── TokenManager        (JWT HS256)
   ├── clock.Clock         (expiração determinística)
   ├── compareHash         (bcrypt — boundary testável)
   ├── genTempPassword     (crypto/rand — boundary testável)
   └── EmailSender ────────► ResendSender (resend-go, pure-Go)  [prod]
                             NoopSender   (loga metadados)       [dev]
            ▲ dispatch assíncrono (goroutine, context.WithoutCancel) best-effort
```

### 3.2 Componentes / Módulos

| Componente | Responsabilidade | Camada |
|------------|------------------|--------|
| `AuthHandler` (modificado) | Mapear os RPCs `RequestPasswordRecovery`, `ChangePassword` e `Login` (campo novo) proto↔domínio; propagar erro verbatim | handler |
| `AuthService` (modificado) | Lógica de recuperação: `RequestPasswordRecovery`, `processRecovery` (síncrono, testável), `ChangePassword`, `Login` (2º branch) | service |
| `service.EmailSender` (novo, interface) | Contrato de envio de e-mail declarado no consumidor | service |
| `UserRepository` (modificado) | `SetTempPassword`, `UpdatePassword`; `GetUserBy*` passam a ler colunas temp | repository |
| `service.User` + `repository.User` (modificados) | Ganham `TempPasswordHash string` e `TempPasswordExpiresAt time.Time` (zero-value quando NULL) | service / repository |
| `userRepositoryAdapter` (modificado, em `module.go`) | Propagar os 2 campos temp na cópia `repository.User`→`service.User` nos métodos `GetUserByEmail`/`GetUserByID` — sem isso o 2º branch do Login nunca enxerga a temporária | wiring (adapter) |
| `ResendSender` (novo) | Implementação concreta de `EmailSender` via SDK `resend-go` | infra/email |
| `NoopSender` (novo) | `EmailSender` no-op para dev — loga `to`/`subject`, nunca o body | infra/email |
| Migração `000009` (novo) | Adiciona `temp_password_hash`, `temp_password_expires_at` a `users` | infra/db |

### 3.3 Camadas e Fronteiras

Layered por domínio (ADR-0002). Interface no consumidor: `EmailSender` é declarada em `auth/service`; o concreto (`internal/infra/email`) não conhece o consumidor; o bind concreto→interface acontece no `auth/module.go`. Sentinelas traduzidas em um único ponto (adapter do módulo). A capacidade de e-mail é infra reusável (ADR-0006) — vive em `internal/infra/email`, fora do domínio.

---

## 4. Contratos de API

### 4.1 Endpoints Expostos

| Ação | Método (RPC) | Service | Payload | Resposta | Status Codes | Auth |
|------|--------------|---------|---------|----------|--------------|------|
| Solicitar recuperação | `RequestPasswordRecovery` | `wc2026.auth.v1.AuthService` | `email` | `message` (genérica) | `OK`, `InvalidArgument` (email malformado) | **Público** |
| Login | `Login` (modificado) | `AuthService` | `email`, `password` | `access_token`, `expires_at`, **`password_change_required`** | `OK`, `Unauthenticated`, `InvalidArgument` | Público |
| Trocar senha | `ChangePassword` | `AuthService` | `temp_password`, `new_password` | `{}` | `OK`, `InvalidArgument`, `Unauthenticated` | **Autenticado** (JWT) |

> Nenhum endpoint usa atualização parcial (`PUT`/`PATCH`) — N/A para a observação anti-`required` da §4.1.1.

### 4.2 Schemas / DTOs

| Schema | Origem | Campos principais | Versão |
|--------|--------|-------------------|--------|
| `RequestPasswordRecoveryRequest` | proto (novo) | `string email` (`buf.validate` email) | v1 |
| `RequestPasswordRecoveryResponse` | proto (novo) | `string message` | v1 |
| `ChangePasswordRequest` | proto (novo) | `string temp_password` (min_len 1), `string new_password` (min_len 8) | v1 |
| `ChangePasswordResponse` | proto (novo) | (vazio) | v1 |
| `LoginResponse` | proto (modificado) | `access_token`, `expires_at`, **`bool password_change_required`** (campo 3, add-only) | v1 |

### 4.3 Eventos Publicados / Consumidos

N/A — não há mensageria. O envio de e-mail é uma chamada HTTP best-effort, não um evento.

---

## 5. Fluxos de Negócio

### 5.1 Fluxo Principal

**Pedido de recuperação (`RequestPasswordRecovery`)** — handler mapeia → `svc.RequestPasswordRecovery(ctx, email)` dispara goroutine de background (`context.WithoutCancel(ctx)`) e retorna **imediatamente** a mensagem genérica. Em background (`processRecovery`): `GetUserByEmail` → se existe, gera senha temporária (`genTempPassword`/`crypto/rand`), bcrypt cost 12, `SetTempPassword(id, hash, clock.Now()+15min)`, e `EmailSender.Send` (texto simples pt-BR com a senha, validade e instrução de troca). Falha de envio → log sem a senha.

**Login (`Login`)** — `GetUserByEmail`; compara `password_hash`. Casou → sucesso, `password_change_required=false`. Não casou e há temp ativa e não expirada (`clock.Now().Before(expires_at)`) → compara `temp_password_hash`; casou → sucesso, `password_change_required=true`. Caso contrário → `Unauthenticated` genérico.

**Troca (`ChangePassword`)** — lê `sub` do contexto → `GetUserByID(sub)` → valida `temp_password` estritamente contra `temp_password_hash` (e somente se não expirada) → `UpdatePassword(id, bcrypt(new_password))` (que zera as colunas temp) → dispara e-mail de notificação (assíncrono best-effort).

### 5.2 Fluxos Alternativos

- **E-mail não cadastrado**: mesma resposta genérica; `processRecovery` não persiste nem envia nada.
- **Falha de envio (best-effort)**: resposta/troca inalteradas; falha apenas logada (RN4/RN11).
- **Login com a senha original durante a recuperação**: acesso normal, `password_change_required=false` (RN10/CA-10).
- **Senha temporária expirada**: login negado (CA-05) e `ChangePassword` recusa (CA-06/CT-013).
- **Cadastro normal**: usuário sem temp ativa → `password_change_required=false` (CA-08).

### 5.3 Mapeamento de User Stories

| User Story (PRD) | Fluxo / Endpoint | Componentes Envolvidos |
|------------------|------------------|------------------------|
| US-01 | `RequestPasswordRecovery` | Handler, Service.RequestPasswordRecovery |
| US-02 | `processRecovery` → `EmailSender.Send` | Service, EmailSender/ResendSender |
| US-03 | `Login` → `password_change_required=true` | Service.Login, Handler |
| US-04 | `ChangePassword` → `UpdatePassword` | Service.ChangePassword, UserRepository |
| US-05 | Resposta genérica + dispatch assíncrono | Service.RequestPasswordRecovery |
| US-06 | Expiração 15 min via `clock` | Service.Login/ChangePassword, clock.Clock |
| US-07 | 2º branch de comparação no Login | Service.Login |
| US-08 | E-mail de notificação de troca | Service.ChangePassword, EmailSender |

---

## 6. Regras de Processamento

### 6.1 Validações de Input

| Regra | Onde Aplica | Comportamento em Falha |
|-------|-------------|------------------------|
| `email` formato válido | proto (`buf.validate.field.string.email`) — interceptor protovalidate | `InvalidArgument` antes do handler |
| `new_password` min_len 8 | proto `ChangePasswordRequest` | `InvalidArgument` |
| `temp_password` min_len 1 | proto `ChangePasswordRequest` | `InvalidArgument` |
| `sub` presente no contexto | `ChangePassword` handler (igual GetMe) | `Unauthenticated` |

### 6.2 Transformações de Dados

- Senha temporária plana → bcrypt hash (cost 12) antes de persistir; a plana só viaja no corpo do e-mail.
- Colunas nullable `temp_password_hash`/`temp_password_expires_at` (sqlc `sql.NullString`/`sql.NullTime`) ↔ domínio: `string` vazio / `time.Time` zero quando NULL. Há **dois** pontos de mapeamento na cadeia: (1) `toUser` (sqlc row → `repository.User`) e (2) `userRepositoryAdapter` (`repository.User` → `service.User`, em `module.go`). Ambos devem propagar os campos temp; omitir (2) faz o Login receber temp sempre zerada.

### 6.3 Regras de Domínio

| Regra | Descrição | Erro de Domínio Associado |
|-------|-----------|---------------------------|
| RN1/RN3 | Pedido gera temp e responde sempre genérico | — (sempre `OK`) |
| RN2/RN5 | Temp válida por 15 min; expiração via `clock` | `Unauthenticated` (login) / `InvalidArgument` (troca) |
| RN4/RN11 | Envio best-effort; falha não altera fluxo | — (só log) |
| RN6 | Troca obrigatória só pós-recuperação | — |
| RN7 | Senha/temp nunca logadas (exceto e-mail ao usuário) | — |
| RN8 | Definir nova senha invalida a temp e remove pendência | — |
| RN10 | Senha original coexiste com a temp; só troca substitui | — |

---

## 7. Persistência de Dados

### 7.1 Banco de Dados Principal

SQLite via `modernc.org/sqlite` (pure-Go, sem CGO — ADR-0001). `SetMaxOpenConns(1)`, PRAGMAs `foreign_keys=ON`/`journal_mode=WAL`/`busy_timeout=5000`.

### 7.2 Tabelas / Coleções

| Nome | Colunas / Campos | Tipos | Constraints | Índices |
|------|------------------|-------|-------------|---------|
| `users` (modificada) | `temp_password_hash` | `TEXT` | NULL (default NULL) | — |
| `users` (modificada) | `temp_password_expires_at` | `TIMESTAMP` | NULL (default NULL) | — |

> Demais colunas inalteradas (`id`, `full_name`, `email UNIQUE`, `password_hash`, `created_at`). Nenhuma tabela nova (restrição do tech-alignment §5).

### 7.3 Migrações

| Versão | Arquivo | Operação |
|--------|---------|----------|
| 000009 | `000009_add_temp_password_to_users.up.sql` | `ALTER TABLE users ADD COLUMN temp_password_hash TEXT; ADD COLUMN temp_password_expires_at TIMESTAMP;` |
| 000009 | `000009_add_temp_password_to_users.down.sql` | `ALTER TABLE users DROP COLUMN temp_password_expires_at; DROP COLUMN temp_password_hash;` |

### 7.4 Estratégia de Transação e Consistência

`SetTempPassword` e `UpdatePassword` são `UPDATE` single-row por `id` — atômicos por natureza, sem transação multi-statement. `UpdatePassword` zera as duas colunas temp no mesmo `UPDATE` (invalidação atômica da temporária).

### 7.5 Política de Retenção / Archival

Sem TTL automático: a temp expirada permanece nas colunas (inerte — rejeitada por comparação de `clock`) até a próxima recuperação sobrescrevê-la ou a troca zerá-la. Não há job de limpeza na v1 (YAGNI).

---

## 8. Integração com APIs Externas

| Serviço Externo | Tipo | Auth | Timeouts | Retry |
|-----------------|------|------|----------|-------|
| Resend (e-mail transacional) | SDK `resend-go` (HTTP, pure-Go) | `RESEND_API_KEY` | timeout no contexto de background do envio | **Sem retry** (best-effort; reentrega adiada p/ fase 2/3) |

Cliente: SDK oficial `resend-go` (D2.b). Isolado atrás de `EmailSender`. Sem circuit breaker/fallback (best-effort, ADR-0006): falha → log. Em dev sem `RESEND_API_KEY` → `NoopSender`.

---

## 9. Sincronização de Dados

### 9.1 Eventos / Filas

N/A — não há fila. O dispatch assíncrono é uma goroutine fire-and-forget simples (D3.b), **não** um worker/queue (seria over-engineering p/ v1).

### 9.2 Idempotência

Não aplicável ao envio (best-effort, sem reentrega). Um novo pedido de recuperação sobrescreve a temp anterior (`SetTempPassword` por `id`).

### 9.3 Outbox / Saga

N/A.

---

## 10. Gerenciamento de Erros

### 10.1 Mapeamento Erro de Negócio → gRPC Status

| Erro | Código | Mensagem | Camada de Origem |
|------|--------|----------|------------------|
| E-mail/senha inválidos (login) | `Unauthenticated` | "e-mail ou senha inválidos" (genérica, idêntica) | service |
| Temp inválida/expirada/ausente (troca) | `InvalidArgument` | "senha temporária inválida ou expirada" | service |
| `sub` ausente (troca) | `Unauthenticated` | "não autenticado" | handler |
| Falha de repositório | `Internal` | "erro interno" | service |
| Pedido de recuperação | sempre `OK` | mensagem genérica | service/handler |

### 10.2 Resiliência

Envio de e-mail best-effort (sem retry/circuit breaker). A goroutine de background usa contexto desacoplado (`context.WithoutCancel`) para não ser cancelada quando o RPC retorna; o `recovery` interceptor protege contra panic.

### 10.3 Estratégia de Logging de Erros

Falha de envio logada em `warn`/`error` via zap com `to`/`subject` — **nunca** a senha, o `password_hash` ou o body do e-mail de recuperação (RN7/CA-09). `NoopSender` loga apenas `to`/`subject`.

---

## 11. Segurança

### 11.1 Autenticação

`ChangePassword` é protegido pelo interceptor JWT (lê `sub` do contexto, igual `GetMe`); `RequestPasswordRecovery` e `Login` são públicos (lista de public methods via constantes `_FullMethodName` geradas). JWT HS256 com `WithValidMethods(["HS256"])` (ADR-0003).

### 11.2 Autorização

`ChangePassword` age sobre o próprio usuário (`sub` do token, nunca de input) — evita IDOR. Sem RBAC (não há papéis).

### 11.3 Criptografia

bcrypt cost 12 para `password_hash` e `temp_password_hash`. Senha temporária gerada por fonte criptográfica (`crypto/rand`), nunca previsível. Senha plana e temp nunca persistidas/retornadas/logadas.

### 11.4 Sanitização e Validação

Queries sqlc 100% parametrizadas (sem concatenação). Validação de input declarativa (protovalidate). E-mail validado no proto.

### 11.5 Rate Limiting / Anti-abuse

**Ausente na v1** — limitação conhecida e aceita (PRD §4.2/§9), adiada para v2. Mitigação parcial: dispatch assíncrono desacopla o custo do envio da resposta. **Anti-enumeração** é a defesa central: resposta e timing idênticos (existência do e-mail não vaza).

### 11.6 Secrets Management

`RESEND_API_KEY` (e `RESEND_FROM_EMAIL`) via env, carregados em `config.Load` com **fail-fast condicional**: ausência em produção (`!IsDevelopment`) aborta o boot (espelha `JWT_SECRET`); em dev, ausência → `NoopSender`. Nunca logados.

---

## 12. Performance

### 12.1 Metas

- Latência p95/p99 do `RequestPasswordRecovery`: **independente do Resend** (resposta imediata, envio em background).
- `Login`/`ChangePassword`: dominadas pelo custo bcrypt (cost 12), como os fluxos auth existentes.

### 12.2 Estratégias

Sem cache novo. O custo bcrypt é intencional (segurança). O dispatch assíncrono evita acoplar a latência da resposta ao provedor.

### 12.3 Limites Conhecidos

`SetMaxOpenConns(1)` (single-writer SQLite) — `SetTempPassword`/`UpdatePassword` são writes rápidos single-row. Sem rate limit, há risco de flood de e-mails (aceito; v2).

---

## 13. Logs e Observabilidade

### 13.1 Logs Estruturados

| Evento | Nível | Campos Chave | Sensibilidade |
|--------|-------|--------------|---------------|
| RPC handled (todos) | info/warn/error | `rpc`, `code`, `duration_ms` | sem PII sensível (interceptor de logging existente) |
| Falha de envio de e-mail | warn/error | `to`, `subject`, `err` | **nunca** senha/temp/body |
| NoopSender (dev) | info | `to`, `subject` | **nunca** body |

### 13.2 Métricas

Sem stack de métricas no projeto (zap apenas). Recomenda-se observar a **taxa de falha de envio** via logs (PRD §10) — sem instrumentação formal na v1.

### 13.3 Tracing

N/A — sem tracing distribuído no projeto.

### 13.4 Alertas

N/A na v1 (sem stack de alerta). Taxa de falha de envio é insumo manual via logs.

---

## 14. Feature Flags

### 14.1 Solução

N/A — projeto não usa feature flags. O comportamento dev/prod do `EmailSender` é decidido por `APP_ENV` + presença de `RESEND_API_KEY` (config), não por flag.

### 14.2 Flags Envolvidas

N/A.

---

## 15. Versionamento de API

### 15.1 Estratégia

Versionamento por path do pacote proto (`wc2026.auth.v1`). Todas as mudanças são **add-only** (2 RPCs novos + 1 campo novo no `LoginResponse`), compatíveis com clientes existentes.

### 15.2 Compatibilidade

Sem breaking change: `password_change_required` (campo 3) é adicionado ao `LoginResponse` sem realocar tags; clientes antigos ignoram o campo novo.

### 15.3 Schemas / Contratos

Proto regenerado via `make proto` (buf) → `gen/wc2026/auth/v1/**` (código gerado, não editar). sqlc regenerado via `make sqlc`.

---

## 16. Deploy e Infraestrutura

### 16.1 Pipeline

Inalterado. `make test` + `make build-all` (cross-compile 4 targets, `CGO_ENABLED=0`) devem permanecer verdes após adicionar `resend-go`.

### 16.2 Empacotamento

Binário único Go, pure-Go (sem toolchain C). `resend-go` é pure-Go — preserva a build sem CGO (ADR-0001).

### 16.3 Infraestrutura como Código

N/A — não há IaC no repositório.

### 16.4 Estratégia de Rollout

N/A (serviço único). A migração 000009 é aditiva (colunas nullable) — segura para rollout sem downtime de schema.

### 16.5 Escalabilidade

Inalterada — serviço stateless (exceto SQLite single-writer).

### 16.6 Rollback

Migração reversível via `000009_*.down.sql`. Rollback do binário não exige rollback de schema (colunas nullable são inertes para a versão anterior).

---

## 17. Mapeamento de User Stories para Definições Técnicas

| User Story (PRD) | Definição Técnica | Componentes Envolvidos |
|------------------|-------------------|------------------------|
| US-01 | `RequestPasswordRecovery` (RPC público) + resposta genérica | Handler, Service |
| US-02 | `processRecovery` gera temp e envia via `EmailSender` | Service, ResendSender |
| US-03 | `password_change_required=true` no `LoginResponse` | Service.Login, Handler |
| US-04 | `ChangePassword` → `UpdatePassword` (zera temp) | Service, UserRepository |
| US-05 | Resposta genérica + dispatch assíncrono (anti-enumeração) | Service.RequestPasswordRecovery |
| US-06 | Expiração 15 min checada com `clock.Now().Before(expires_at)` | Service, clock.Clock |
| US-07 | 2º branch de comparação (temp) no Login; original sempre válida | Service.Login |
| US-08 | E-mail de notificação best-effort em `ChangePassword` | Service, EmailSender |

---

## 18. Dependências Externas

| Tipo | Nome | Versão | Motivo |
|------|------|--------|--------|
| SDK e-mail | `github.com/resend/resend-go` | latest pure-Go | Cliente oficial Resend (D2.b); encapsula contrato HTTP; pure-Go (sem CGO) |
| Crypto (já presente) | `golang.org/x/crypto/bcrypt` | v0.45.0 | Hash da senha temporária (cost 12) |
| Stdlib | `crypto/rand` | — | Geração da senha temporária |
| DI (já presente) | `go.uber.org/fx` | v1.23.0 | Bind `EmailSender` no módulo auth |

---

## 19. Estratégia de Testes

> **Resumo**: 38 casos de teste | Unitários: 19 | Integração: 12 | E2E: 7
> **Padrão**: `go test` + testify/require; mocks de interfaces declaradas no consumidor; SQLite real efêmero (`testutil.TestNewDB`); servidor completo via `testutil.TestNewBufconnServer` (bufconn, cadeia de interceptors real); `clock.Clock` fixo/avançável (sem `time.Sleep`); `errors.Is` em vez de substring; sem mock-driven confidence (captura de argumento + `bcrypt.CompareHashAndPassword`).
> **Conformidade `agent-spec-testing-best-practices`**: `mock_budget_observado=true`; 7 gates aplicados (invariant_first, owning_layer, real_execution, failure_means_fix_production, no_snapshot_without_contract, no_self_set_mock_assertion, negative_companion); 20 CTs com `real_execution_boundary != none`.

### 19.1 Testes Unitários

#### Service: AuthService (`auth_service_test.go`)

| CT | Cenário | CA | Input | Resultado Esperado | Mock/Setup |
|----|---------|----|-------|--------------------|------------|
| CT-001 | processRecovery com e-mail cadastrado: persiste temp_password_hash bcrypt-válido e expires_at = now+15min | CA-01 | email='recovery@example.com' | SetTempPassword chamado 1 vez com hash bcrypt válido da senha temporária conhecida e expires_at = 2026-06-01T12:15:00Z; Send chamado 1 vez | userRepoMock.GetUserByEmail retorna User existente; genTempPassword injetado retorna senha conhecida 'TempPass-KNOWN-1!' |
| CT-002 | processRecovery com e-mail não cadastrado: não chama SetTempPassword nem Send | CA-02 | email='ghost@example.com' | Nenhuma persistência. Nenhum e-mail. Método retorna sem erro (CA-02: silêncio total). | userRepoMock.GetUserByEmail retorna ErrUserNotFound; SetTempPassword registra se foi chamado (bool flag) |
| CT-003 | processRecovery com falha no Send: método retorna sem erro e SetTempPassword já foi chamado | CA-03 | email='exists@example.com'; emailSender falha | processRecovery retorna nil; SetTempPassword foi chamado; falha de Send apenas logada internamente. | userRepoMock.GetUserByEmail retorna User existente; emailSenderMock.Send retorna errors.New('resend timeout') |
| CT-006 | Login com senha original (e-mail com temp válida): acesso concedido, password_change_required=false | CA-10 | password='senha-original' | Login bem-sucedido; PasswordChangeRequired=false. | User tem password_hash=bcrypt('senha-original'), temp_password_hash=bcrypt('TempXxx!'), temp_password_expires_at=clock.Now()+5min; clock fixo, compareFunc real bcrypt, bcryptCost=MinCost |
| CT-007 | Login com senha temporária válida (< 15 min): acesso concedido, password_change_required=true | CA-04 | password='TempXxx!' | Login bem-sucedido; PasswordChangeRequired=true. | User tem password_hash=bcrypt('senha-original'), temp_password_hash=bcrypt('TempXxx!'), temp_password_expires_at=clock.Now()+10min; clock fixo |
| CT-008 | Login com senha temporária expirada: acesso negado (Unauthenticated genérico) | CA-05 | password='TempXxx!' com temp expirada | codes.Unauthenticated; mensagem genérica; nenhum token emitido. | User tem temp_password_hash=bcrypt('TempXxx!'), temp_password_expires_at=clock.Now()-time.Second (já expirada); clock fixo |
| CT-009 | Login — fronteira exata de expiração (expires_at == clock.Now()): acesso negado | CA-05 | senha temporária com expires_at = clock.Now() | Acesso negado: o sistema usa `clock.Now().Before(expires_at)` ou equivalente que exclui o instante exato. | clock fixo em T0; temp_password_expires_at = T0 (exatamente igual ao now) |
| CT-010 | Login com senha original quando temp está expirada: acesso concedido, PasswordChangeRequired=false | CA-10 | password='senha-original' com temp expirada | Acesso normal sem troca forçada. | temp_password_expires_at expirada; password_hash bcrypt da senha original presente |
| CT-011 | ChangePassword com temp válida: nova senha persistida via bcrypt, temp invalidada (NULL) | CA-06 | temp_password='TempXxx!', new_password='NovaSenha@2026!' | UpdatePassword chamado com hash bcrypt válido da nova senha; campos temp zerados. | User autenticado (sub extraído do contexto via mock de extração de sub); GetUserByID retorna User com temp_password_hash=bcrypt('TempXxx!'), temp_password_expires_at=clock.Now()+10min |
| CT-012 | ChangePassword com temp_password incorreta: InvalidArgument, UpdatePassword não chamado | CA-06 | temp_password='SenhaErrada!', new_password='NovaSenha@2026!' | codes.InvalidArgument; senha não alterada. | User com temp ativa; UpdatePassword mock registra se foi chamado |
| CT-013 | ChangePassword com temp expirada: InvalidArgument mesmo com senha temporária correta | CA-06 | temp_password='TempXxx!' (hash correto mas expirada) | InvalidArgument 'senha temporária inválida ou expirada'. | User com temp_password_expires_at=clock.Now()-time.Minute (expirada) |
| CT-014 | ChangePassword sem temp ativa (campos NULL): InvalidArgument | CA-06 | temp_password='qualquer', new_password='NovaSenha@2026!' | Falha com mensagem 'senha temporária inválida ou expirada'. | GetUserByID retorna User com TempPasswordHash='' e TempPasswordExpiresAt=zero time |
| CT-015 | ChangePassword dispara e-mail de notificação best-effort após troca bem-sucedida | CA-11 | troca bem-sucedida seguida de notificação | e-mail de notificação enviado; body não contém senha; troca efetivada. | User com temp ativa e válida; emailSenderMock captura argumentos de Send e retorna nil |
| CT-016 | ChangePassword com falha no e-mail de notificação: troca efetivada normalmente | CA-12 | troca bem-sucedida + Send falha | ChangePassword retorna nil; UpdatePassword executado; CA-12 satisfeito. | emailSenderMock.Send retorna errors.New('smtp timeout'); UpdatePassword mock registra chamada e retorna nil |
| CT-017 | Login sem recovery (usuário de cadastro normal): PasswordChangeRequired=false | CA-08 | password='senha-normal' (cadastro sem recovery) | Acesso normal; PasswordChangeRequired=false. | User com TempPasswordHash='' e TempPasswordExpiresAt=zero time; password_hash=bcrypt('senha-normal') |
| CT-036 | Segurança — senha temporária não aparece no body do e-mail de notificação de troca | CA-09, CA-11 | temp_password='TempXxx!', new_password='NovaSenha@2026!' | Nenhuma senha no corpo ou assunto do e-mail. | emailSenderMock captura to, subject, body de Send; User com temp ativa |
| CT-037 | Segurança — senha temporária gerada por genTempPassword usa crypto/rand (não math/rand) |  | chamada dupla ao gerador padrão | Duas chamadas produzem strings distintas de comprimento adequado. | AuthService construído sem substituição de genTempPassword (produção) |

#### Apresentação: AuthHandler (`auth_handler_test.go`)

| CT | Cenário | CA | Input | Resultado Esperado | Mock/Setup |
|----|---------|----|-------|--------------------|------------|
| CT-004 | RequestPasswordRecovery RPC responde mensagem genérica imediatamente (e-mail existe) | CA-01 | RequestPasswordRecoveryRequest{email: 'user@example.com'} | Resposta genérica pt-BR sem erro gRPC. | authServiceMock.RequestPasswordRecovery configurado para retornar (msg genérica, nil); handler construído sobre o mock |
| CT-005 | RequestPasswordRecovery RPC responde mesma mensagem genérica para e-mail não cadastrado | CA-02 | RequestPasswordRecoveryRequest{email: 'nobody@example.com'} | Mesma string de mensagem de CT-004. Status OK. Anti-enumeração por resposta confirmada. | authServiceMock.RequestPasswordRecovery retorna a mensagem genérica (mesma função independente do e-mail) |

### 19.2 Testes de Integração

- **Setup**: SQLite real efêmero via `internal/testutil.TestNewDB` (migrations + `foreign_keys=ON`); `clock.Clock` injetável avançável; `EmailSender` fake captura `Send`; senha temporária via `genTempPassword` injetado.

| CT | Cenário | CA | Fluxo | Resultado Esperado |
|----|---------|----|-------|--------------------|
| CT-018 | SetTempPassword: persiste temp_password_hash e expires_at com FK e NOT NULL observáveis no banco | CA-01 | Criar usuário via repo.CreateUser; Chamar repo.SetTempPassword(ctx, userID, hashedTemp, expiresAt); Chamar repo.GetUserByID(ctx, userID) e verificar campos lidos do banco | Campos temp persistidos corretamente; round-trip de leitura confirma o que foi gravado. |
| CT-019 | SetTempPassword em userID inexistente: retorna erro (zero linhas afetadas) |  | Chamar repo.SetTempPassword(ctx, 'id-invalido', hash, expiresAt); Verificar err != nil ou errors.Is(err, repository.ErrUserNotFound) | Erro retornado; nenhuma linha gravada. |
| CT-020 | UpdatePassword: nova senha gravada, campos temp zerados, GetUserByID confirma round-trip | CA-06, CA-07 | Criar usuário e definir temp via SetTempPassword; Chamar repo.UpdatePassword(ctx, userID, newHash); Chamar repo.GetUserByID(ctx, userID) | Estado limpo confirmado no banco; temp zerada. |
| CT-021 | GetUserByEmail e GetUserByID retornam TempPasswordHash e TempPasswordExpiresAt corretamente | CA-01 | Criar usuário; chamar SetTempPassword com tempHash e expiresAt conhecidos; Chamar repo.GetUserByEmail(ctx, email) e repo.GetUserByID(ctx, id); Comparar TempPasswordHash e TempPasswordExpiresAt de ambas as leituras | Ambas as queries retornam TempPasswordHash e TempPasswordExpiresAt corretos. |
| CT-022 | GetUserByEmail para usuário sem recovery: campos temp retornados como zero values | CA-08 | Criar usuário padrão; Chamar repo.GetUserByEmail(ctx, email); Verificar err == nil | Campos NULL mapeados para zero values Go sem erro. |
| CT-023 | Integração Login com temp válida via stack real: PasswordChangeRequired=true em LoginResponse | CA-04 | Construir handler+service+repository com SQLite efêmero e clock fixo; Chamar h.Login(ctx, req) com password='TempXxx!'; Verificar resp.PasswordChangeRequired == true | PasswordChangeRequired=true; acesso concedido; campo novo em LoginResponse. |
| CT-024 | Integração Login com temp expirada via stack real: Unauthenticated | CA-05 | Configurar stack com FixedClock em T0; SetTempPassword com expiresAt = T0 + 15min; Avançar clock para T0 + 16min | Acesso negado; clock avançável sem time.Sleep. |
| CT-025 | Integração ChangePassword: nova senha aceita no Login subsequente; temp rejeitada | CA-06, CA-07 | SetTempPassword no usuário; Chamar svc.ChangePassword(ctx, userID, 'TempXxx!', 'NovaSenha@2026!'); Chamar h.Login(ctx, {email, 'NovaSenha@2026!'}): verificar err==nil, PasswordChangeRequired==false | CA-06 + CA-07 verificados via stack real. |
| CT-026 | Integração processRecovery via stack real: temp persistida e verificável com bcrypt | CA-01 | Construir AuthService com repo real + emailSender mock + genTempPassword fixo; Chamar svc.ProcessRecovery(ctx, email); Chamar repo.GetUserByEmail(ctx, email) | Hash persistido no banco válido via bcrypt. Expiração correta. E-mail disparado. |
| CT-027 | Integração processRecovery com e-mail inexistente: banco inalterado | CA-02 | Chamar svc.ProcessRecovery(ctx, 'ghost@example.com'); Verificar sendCalled == false; Confirmar que o banco continua vazio (nenhum usuário com temp_password_hash) | Nenhuma mutação no banco; nenhum e-mail. CA-02 no nível de integração. |
| CT-035 | Migração 000009: tabela users tem colunas temp_password_hash e temp_password_expires_at nullable |  | Criar usuário via repo.CreateUser; Verificar err == nil; Chamar repo.GetUserByID: TempPasswordHash=='' e TempPasswordExpiresAt.IsZero() | Schema compatível com dados existentes; migração não quebra inserções pré-existentes. |
| CT-038 | UserRepository.withNationalTeams não quebra com colunas temp NULL (regression guard) |  | Inserir usuário diretamente via SQL sem temp fields; Chamar repo.GetUserByID(ctx, userID); Verificar err == nil | Scan de NULL não causa panic nem erro. Campos zero-valued. |

### 19.3 Testes End-to-End

- **Framework**: HTTP black-box gRPC via `internal/testutil.TestNewBufconnServer` (cadeia de interceptors real, em `test/e2e/`).

#### CT-028 — E2E RequestPasswordRecovery: resposta genérica independente de existência do e-mail
- **CA**: CA-01, CA-02
- **Pré-condições**: testutil.TestNewBufconnServer; Usuário cadastrado via client.Register para o primeiro sub-caso; Nenhum usuário para o segundo sub-caso
- **Passos**:
  1. Registrar usuário no servidor E2E
  2. Chamar client.RequestPasswordRecovery(ctx, {email do usuário}) → capturar msg
  3. Chamar client.RequestPasswordRecovery(ctx, {email desconhecido}) → capturar msg
  4. Comparar as duas mensagens: devem ser idênticas
  5. Verificar status.Code em ambos: OK
- **Validações**: Mensagens idênticas em ambos os casos. Anti-enumeração por resposta confirmada ponta-a-ponta.
- **Notas**: Teste table-driven com 2 sub-casos em 1 teste. Cobre anti-enumeração na cadeia completa incluindo interceptors.

#### CT-029 — E2E RequestPasswordRecovery sem autenticação: deve ser RPC público (sem token)
- **CA**: 
- **Pré-condições**: testutil.TestNewBufconnServer sem clock especial
- **Passos**:
  1. Chamar client.RequestPasswordRecovery(ctx_sem_token, req)
  2. Verificar que status.Code != codes.Unauthenticated
  3. Verificar que a resposta é OK (mensagem genérica retornada)
- **Validações**: RPC público: interceptor JWT não bloqueia. Constante FullMethodName configurada como pública.
- **Notas**: Verifica configuração correta de public methods no servidor. Typo em FullMethodName literaliza um bug de fail-open.

#### CT-030 — E2E ChangePassword sem token: Unauthenticated pelo interceptor JWT
- **CA**: 
- **Pré-condições**: testutil.TestNewBufconnServer
- **Passos**:
  1. Chamar client.ChangePassword(ctx_sem_token, req)
  2. Verificar codes.Unauthenticated
- **Validações**: Interceptor JWT bloqueia antes do handler.
- **Notas**: Simetria com TestE2E_GetMe_NoToken_Unauthenticated já existente.

#### CT-031 — E2E ChangePassword com token válido mas temp_password errada: InvalidArgument
- **CA**: CA-06
- **Pré-condições**: Usuário registrado e com temp ativa via RequestPasswordRecovery E2E ou via setup direto; Login com temp para obter token; Clock fixo para não expirar
- **Passos**:
  1. Register → SetTempPassword (via infra direta no teste ou via processRecovery exposto)
  2. Login com temp para obter token
  3. Chamar client.ChangePassword(authCtx, {temp_password='Errada!', new_password='Nova@2026!'})
  4. Verificar codes.InvalidArgument
- **Validações**: Handler recebido, service rejeita com InvalidArgument.
- **Notas**: Companion de CT-030. Demonstra que o handler é alcançado com token válido.

#### CT-032 — E2E fluxo completo recovery: RequestPasswordRecovery → Login(temp) → ChangePassword → Login(nova) → Login(temp) negado
- **CA**: CA-04, CA-06, CA-07
- **Pré-condições**: testutil.TestNewBufconnServer com clock fixo; NoopEmailSender (dev mode) configurado para não enviar de verdade
- **Passos**:
  1. Register usuário
  2. RequestPasswordRecovery(email) → aguardar processamento (goroutine pode precisar de sincronização via WaitGroup exposto ou via sleep determinístico com clock)
  3. Consultar banco para obter temp_password_hash e usar genTempPassword fixo injetado no servidor de teste
  4. Login(temp) → verificar PasswordChangeRequired==true
  5. ChangePassword(temp, nova) → verificar OK
  6. Login(nova) → verificar OK, PasswordChangeRequired==false
  7. Login(temp antiga) → verificar codes.Unauthenticated
- **Validações**: Todos os CAs 04, 06, 07 verificados ponta-a-ponta.
- **Notas**: Teste E2E de maior valor da feature. genTempPassword injetável no TestNewBufconnServer é condição para testabilidade determinística (sem precisar sniffar o banco). O WaitGroup ou mecanismo de sincronização da goroutine de background deve ser exposto para o servidor de teste (ou o teste chama processRecovery diretamente via setup). Registrar necessidade de sincronização da goroutine em cenarios_nao_cobertos se não resolvido.

#### CT-033 — E2E Login com senha original durante recovery ativa: PasswordChangeRequired=false ponta-a-ponta
- **CA**: CA-10
- **Pré-condições**: Usuário com senha original e temp ativa válida; clock fixo
- **Passos**:
  1. Setup: Register + SetTempPassword via infra do servidor de teste
  2. Login com senha original
  3. Verificar resp.PasswordChangeRequired == false
  4. Verificar resp.AccessToken não vazio
- **Validações**: Campo novo PasswordChangeRequired=false presente no proto response.
- **Notas**: Verifica que o novo campo boolean no proto está corretamente mapeado no handler.

#### CT-034 — E2E Login com temp válida: PasswordChangeRequired=true no proto response
- **CA**: CA-04
- **Pré-condições**: Usuário com temp ativa não expirada; clock fixo
- **Passos**:
  1. Login com temp
  2. Verificar resp.PasswordChangeRequired == true
- **Validações**: PasswordChangeRequired=true mapeado corretamente no proto.
- **Notas**: Companion de CT-033. Juntos cobrem CA-04 e CA-10 ponta-a-ponta.

### 19.4 Cenários de Erro

| CT | Cenário | CA | Trigger | Comportamento Esperado | Log/Observabilidade |
|----|---------|----|---------|------------------------|----------------------|
| CT-003 | processRecovery com falha no Send: método retorna sem erro e SetTempPassword já foi chamado | CA-03 | email='exists@example.com'; emailSender falha | processRecovery retorna nil; SetTempPassword foi chamado; falha de Send apenas logada internamente. | Cobre RN4. A falha de Send não desfaz hash persistido. |
| CT-008 | Login com senha temporária expirada: acesso negado (Unauthenticated genérico) | CA-05 | password='TempXxx!' com temp expirada | codes.Unauthenticated; mensagem genérica; nenhum token emitido. | Fronteira exata: expires_at = now-1s. Complementar: testar expires_at = now (exato) em CT-009. |
| CT-009 | Login — fronteira exata de expiração (expires_at == clock.Now()): acesso negado | CA-05 | senha temporária com expires_at = clock.Now() | Acesso negado: o sistema usa `clock.Now().Before(expires_at)` ou equivalente que exclui o instante exato. | Captura bug clássico de boundary: `<=` vs `<`. Se falhar, o SUT usa operador errado de comparação. |
| CT-013 | ChangePassword com temp expirada: InvalidArgument mesmo com senha temporária correta | CA-06 | temp_password='TempXxx!' (hash correto mas expirada) | InvalidArgument 'senha temporária inválida ou expirada'. | Boundary crítico: impede reuso de temp expirada para troca. |
| CT-014 | ChangePassword sem temp ativa (campos NULL): InvalidArgument | CA-06 | temp_password='qualquer', new_password='NovaSenha@2026!' | Falha com mensagem 'senha temporária inválida ou expirada'. | Usuário de cadastro normal nunca deve ter acesso ao ChangePassword sem recovery prévia. |
| CT-016 | ChangePassword com falha no e-mail de notificação: troca efetivada normalmente | CA-12 | troca bem-sucedida + Send falha | ChangePassword retorna nil; UpdatePassword executado; CA-12 satisfeito. | Companion de CT-015. Cobre D5 best-effort. |
| CT-022 | GetUserByEmail para usuário sem recovery: campos temp retornados como zero values | CA-08 | email de usuário sem recovery | Campos NULL mapeados para zero values Go sem erro. | Crítico: sqlc precisa usar sql.NullString/sql.NullTime para os campos nullable. Falha aqui revela bug de mapeamento. |
| CT-024 | Integração Login com temp expirada via stack real: Unauthenticated | CA-05 | password='TempXxx!' com clock após expiração | Acesso negado; clock avançável sem time.Sleep. | Demonstra uso correto de clock injetado para expiração determinística sem sleep. |
| CT-028 | E2E RequestPasswordRecovery: resposta genérica independente de existência do e-mail | CA-01, CA-02 | Parametrizado: {email existente} e {email não existente} | Mensagens idênticas em ambos os casos. Anti-enumeração por resposta confirmada ponta-a-ponta. | Teste table-driven com 2 sub-casos em 1 teste. Cobre anti-enumeração na cadeia completa incluindo interceptors. |
| CT-029 | E2E RequestPasswordRecovery sem autenticação: deve ser RPC público (sem token) |  | RequestPasswordRecoveryRequest sem metadata de autorização | RPC público: interceptor JWT não bloqueia. Constante FullMethodName configurada como pública. | Verifica configuração correta de public methods no servidor. Typo em FullMethodName literaliza um bug de fail-open. |
| CT-030 | E2E ChangePassword sem token: Unauthenticated pelo interceptor JWT |  | ChangePasswordRequest sem Authorization header | Interceptor JWT bloqueia antes do handler. | Simetria com TestE2E_GetMe_NoToken_Unauthenticated já existente. |
| CT-036 | Segurança — senha temporária não aparece no body do e-mail de notificação de troca | CA-09, CA-11 | temp_password='TempXxx!', new_password='NovaSenha@2026!' | Nenhuma senha no corpo ou assunto do e-mail. | Gate de segurança direto. Assertion de ausência de string específica — não snapshot (Gate 5 cumprido). |
| CT-037 | Segurança — senha temporária gerada por genTempPassword usa crypto/rand (não math/rand) |  | chamada dupla ao gerador padrão | Duas chamadas produzem strings distintas de comprimento adequado. | Não é possível verificar crypto/rand diretamente sem inspecionar código; a verificação de unicidade e comprimento é o proxy observável. O revisor de código deve confirmar a importação. |
| CT-038 | UserRepository.withNationalTeams não quebra com colunas temp NULL (regression guard) |  | usuário sem campos temp no banco | Scan de NULL não causa panic nem erro. Campos zero-valued. | Detecta bug clássico de scan de sql.NullString sem tratamento. Se falhar, o mapeamento toUser precisa usar sql.NullString. |

### Rastreabilidade: Critérios de Aceite → Testes

| CA (PRD) | Unitários | Integração | E2E |
|----------|-----------|------------|-----|
| CA-01 | CT-001, CT-004 | CT-018, CT-021, CT-026 | CT-028 |
| CA-02 | CT-002, CT-005 | CT-027 | CT-028 |
| CA-03 | CT-003 | — | — |
| CA-04 | CT-007 | CT-023 | CT-032, CT-034 |
| CA-05 | CT-008, CT-009 | CT-024 | — |
| CA-06 | CT-011, CT-012, CT-013, CT-014 | CT-020, CT-025 | CT-031, CT-032 |
| CA-07 | — | CT-020, CT-025 | CT-032 |
| CA-08 | CT-017 | CT-022 | — |
| CA-09 | CT-036 | — | — |
| CA-10 | CT-006, CT-010 | — | CT-033 |
| CA-11 | CT-015, CT-036 | — | — |
| CA-12 | CT-016 | — | — |

---

> **Cenários não cobertos** (decisões conscientes): rate limiting (fora de escopo v1, PRD §4.2); invalidação de JWT após troca (RN9/ADR-0003 — stateless, expira em 1h); template HTML do e-mail (v2); verificação do conteúdo exato do body do e-mail (detalhe de implementação — apenas ausência de senha é asserida, CT-036); load/throughput (fase de load testing futura); sincronização determinística da goroutine no E2E (CT-032 exige estender `TestNewBufconnServer` para aceitar `EmailSender`/`genTempPassword` de teste).
>
> **Recomendações de implementação** (do QA generator): expor `processRecovery` e o `genTempPassword` default via `export_test.go`; estender `TestNewBufconnServer` para injetar `EmailSender`/`genTempPassword` de teste (pré-requisito do E2E determinístico); adicionar `SetTempPassword`/`UpdatePassword` ao mock de `UserRepository` e ao `userRepositoryAdapter`; declarar `emailSenderMock` no pacote de teste do service; usar `FixedClock.Advance` (não `time.Sleep`) nas expirações; `down.sql` reverte as colunas; teste de fumaça garantindo que o `NoopSender` não loga o body.

---

## 20. Riscos Técnicos

| Risco | Probabilidade | Impacto | Mitigação |
|-------|---------------|---------|-----------|
| Goroutine de background usar contexto cancelado do RPC | Média | Envio nunca ocorre | `context.WithoutCancel(ctx)` + cobertura de `processRecovery` síncrono |
| Timing leak no 2º branch do Login (temp = 2 bcrypts vs 1) | Baixa | Revela "há recuperação ativa" (não a existência genérica do e-mail) | Aceito: o caminho de e-mail inexistente já faz 1 compare contra `dummyHash`, equalizando a existência; o 2º compare só ocorre após um pedido prévio de recuperação. **Equalização total (sempre 2 compares) avaliada e rejeitada no challenge (2026-05-31): dobraria o custo bcrypt cost 12 de todo login para fechar um canal que exige e-mail já conhecido + senha errada e vaza apenas "recuperação ativa".** Documentado. |
| Janela de 15 min apertada (entrega lenta do e-mail) | Média | Temp expira antes do uso | Decisão travada v1 (PRD §9); métrica de expiração-antes-do-uso sinaliza revisão |
| Mapeamento NULL incorreto das colunas temp (panic no scan) | Baixa | Quebra `GetUserBy*` | sqlc `sql.NullString`/`sql.NullTime` + CT-022/CT-038 (regression guard) |
| `resend-go` introduzir dependência não-pure-Go | Baixa | Quebra build sem CGO | Verificado pure-Go (D2); `make build-all` no gate |
| Flood de e-mails (sem rate limit) | Média | Custo/abuso | Limitação conhecida v1; volume monitorado p/ priorizar v2 |

---

## 21. Observações Técnicas

### Decisões de challenge (2026-05-31)
- **`password_change_required` é derivado, não persistido**: não há coluna de "flag de troca". O sinal é computado no Login (true só quando o login casa pelo branch da temporária). Reconcilia a menção a "flag de troca" no `tech-alignment.md` §5 — o estado de recuperação vive em apenas 2 colunas (`temp_password_hash`, `temp_password_expires_at`); a pendência é implícita (existência de temp ativa). Decisão KISS confirmada no challenge.
- **"Troca obrigatória" = sinalização, não enforcement**: o backend retorna `password_change_required=true` e não bloqueia outras operações até a troca. O enforcement é responsabilidade do cliente. Coerente com RN9 (JWT stateless, sem gate) e com a poda de "token de escopo restrito" no PRD §4.2. Mitigação: temp expira em 15 min; token em 1h. Confirmado no challenge.

### Decisões de design (FASE 3)
- **Decomposição**: estender o `AuthService` (não criar service dedicado) — Login já muda (D4) e compartilha `UserRepository`/`compareHash`/`clock`/`dummyHash`; KISS. `AuthService` ganha 1 dependência nova (`EmailSender`).
- **Contrato**: 2 RPCs novos no `AuthService` proto + campo `bool password_change_required` no `LoginResponse` (add-only).
- **Validação de `ChangePassword`**: estritamente contra a temporária (decisão do dono do produto) — só funciona pós-recuperação; não aceita a senha vigente.
- **Geração da senha temporária**: `genTempPassword` injetado (default `crypto/rand`) como boundary testável, espelhando `compareHash`/`clock`.

### ADRs Aplicáveis nesta Feature
- **ADR-0001 (pure-Go, sem CGO)** — APLICÁVEL (§8, §16, §18): `resend-go` é pure-Go; `make build-all` permanece verde.
- **ADR-0002 (uber-fx + interface no consumidor)** — APLICÁVEL (§3): `EmailSender` no consumidor (`auth/service`); bind no `auth/module.go`.
- **ADR-0003 (JWT HS256 + bcrypt + TTL 1h)** — APLICÁVEL (§11): `ChangePassword` protegido por JWT; bcrypt cost 12 na temp; RN9 (troca não invalida tokens já emitidos).
- **ADR-0004 (idioma pt-BR + rename)** — N/A (superseded pela ADR-0005).
- **ADR-0005 (schema em inglês, sem bridge)** — APLICÁVEL (§7): colunas `temp_password_hash`/`temp_password_expires_at` em inglês; sqlc gera direto.
- **ADR-0006 (e-mail transacional Resend, interface no consumidor)** — APLICÁVEL (§8, §11.6): capacidade central da feature; postura best-effort; política de credencial fail-fast prod / no-op dev.

### Detecção de candidatos a ADR (FASE 4.5)
Nenhum candidato novo a ADR (5/5). As decisões D0–D5 são **feature-scoped** (falham C1 — transversal), conforme já consolidado no `tech-alignment.md` §4: a única decisão transversal (envio de e-mail) já é a ADR-0006. Senha temporária, coexistência de credenciais e notificação de troca são específicas do fluxo de recuperação do `auth`.

### Glossário
Termos canônicos globais respeitados (`User`, `Sessão`). Termos da feature **canonizados no challenge (2026-05-31)** em `/docs/specs/features/esqueci-a-senha/domain-glossary.md` (nível feature): **senha temporária** (credencial adicional, de tempo limitado, que coexiste com a senha vigente; _evitar_ token/OTP/código de reset), **troca obrigatória** (sinalização pós-login-com-temp) e **recuperação de acesso** (fluxo completo).

---

## 22. Arquivos Envolvidos e Ações

### 22.0 Visão em Árvore

```
api/proto/wc2026/auth/v1/
└── auth.proto                                          [M]
internal/domain/auth/
├── service/
│   ├── auth_service.go                                 [M]
│   ├── auth_service_test.go                            [M]
│   └── export_test.go                                  [M]
├── handler/
│   ├── auth_handler.go                                 [M]
│   └── auth_handler_test.go                            [M]
├── repository/
│   ├── user_repository.go                              [M]
│   └── user_repository_test.go                         [M]
└── module.go                                           [M]
internal/infra/email/                                   [N]
├── resend.go                                           [N]
├── noop.go                                             [N]
└── email_test.go                                       [N]
internal/infra/config/
├── config.go                                           [M]
└── config_test.go                                      [M]
internal/infra/db/
├── migrations/
│   ├── 000009_add_temp_password_to_users.up.sql        [N]
│   └── 000009_add_temp_password_to_users.down.sql      [N]
└── queries/users.sql                                   [M]
internal/server/server.go                              [M]
internal/testutil/bufconn.go                           [M]
test/e2e/auth_e2e_test.go                              [M]
gen/wc2026/auth/v1/**                                   [N] (regenerado — make proto)
internal/infra/db/sqlc/**                               [M] (regenerado — make sqlc)
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

### 22.1 Arquivos a Criar

| Arquivo | Descrição | Camada |
|---------|-----------|--------|
| `internal/infra/email/resend.go` | `ResendSender` (SDK `resend-go`) implementando `EmailSender` | infra |
| `internal/infra/email/noop.go` | `NoopSender` para dev (loga `to`/`subject`) | infra |
| `internal/infra/email/email_test.go` | Testes do NoopSender (não loga body) e do ResendSender | infra (teste) |
| `internal/infra/db/migrations/000009_add_temp_password_to_users.up.sql` | Adiciona colunas temp | db |
| `internal/infra/db/migrations/000009_add_temp_password_to_users.down.sql` | Reverte colunas temp | db |

### 22.2 Arquivos a Modificar

| Arquivo | Modificação | Motivo |
|---------|-------------|--------|
| `api/proto/wc2026/auth/v1/auth.proto` | +`RequestPasswordRecovery`, +`ChangePassword`, +campo `password_change_required` | Contrato dos novos RPCs |
| `internal/domain/auth/service/auth_service.go` | +`RequestPasswordRecovery`/`processRecovery`/`ChangePassword`; Login 2º branch; +interface `EmailSender`; +`genTempPassword`; **+campos `TempPasswordHash`/`TempPasswordExpiresAt` no struct `service.User`** | Lógica de recuperação |
| `internal/domain/auth/handler/auth_handler.go` | +mappers dos 2 RPCs; campo novo no LoginResponse | Apresentação |
| `internal/domain/auth/repository/user_repository.go` | +`SetTempPassword`/`UpdatePassword`; `toUser`/`GetUserBy*` leem temp | Persistência |
| `internal/domain/auth/module.go` | +bind `EmailSender` (Resend/Noop por config); **+propagar `TempPasswordHash`/`TempPasswordExpiresAt` no `userRepositoryAdapter` (GetUserByEmail/GetUserByID)** | Wiring + adapter |
| `internal/infra/config/config.go` | +`ResendAPIKey`/`ResendFromEmail`; fail-fast condicional | Secrets |
| `internal/infra/db/queries/users.sql` | +`SetTempPassword`/`UpdatePassword`; `GetUserBy*` selecionam temp | Queries sqlc |
| `internal/server/server.go` | `RequestPasswordRecovery` na lista de public methods | Cadeia gRPC |
| `internal/testutil/bufconn.go` | Injetar `EmailSender`/`genTempPassword` de teste | E2E determinístico |
| `*_test.go` (service/handler/repository/config) | Casos de teste da seção 19 | Cobertura |
| `test/e2e/auth_e2e_test.go` | Fluxos E2E de recuperação | Cobertura E2E |

### 22.3 Arquivos de Referência (somente leitura)

| Arquivo | Motivo da Consulta |
|---------|--------------------|
| `internal/domain/auth/token/token_manager.go` | Padrão de injeção de `clock`/boundary |
| `internal/domain/auth/interceptor/auth_jwt.go` | `SubjectFromContext` p/ `ChangePassword` |
| `internal/infra/logger/interceptor.go` | Padrão de log sem campos sensíveis |
| `internal/infra/clock/clock.go` | Interface `Clock` |
| `internal/testutil/db.go` | `TestNewDB` p/ integração |

---

## 23. Checklist Final

- [x] Variante registrada (backend) na seção 1
- [x] Stack identificada
- [x] TECH_SPEC cobre todo o PRD (todas as US-XX mapeadas em 17)
- [x] Resumo técnico claro e objetivo (seção 2)
- [x] Arquitetura definida com componentes e camadas (seção 3)
- [x] Contratos de API definidos com payloads, status codes e schemas (seção 4)
- [x] Fluxos de negócio descritos (seção 5)
- [x] Regras de processamento e validações (seção 6)
- [x] Persistência: tabelas, índices, migrações, transação (seção 7)
- [x] Integrações externas mapeadas (seção 8)
- [x] Sincronização: eventos, idempotência (seção 9 — N/A justificado)
- [x] Gerenciamento de erros e resiliência (seção 10)
- [x] Segurança: auth, autorização, criptografia, sanitização (seção 11)
- [x] Performance: metas, estratégias, limites (seção 12)
- [x] Logs, métricas, tracing e alertas (seção 13)
- [x] Feature flags listadas (seção 14 — N/A justificado)
- [x] Versionamento de API definido (seção 15)
- [x] Deploy e infraestrutura (seção 16)
- [x] Dependências externas listadas (seção 18)
- [x] Estratégia de testes via `agent-spec-qa-test-generator` integrada (seção 19, com rastreabilidade CA→CT)
- [x] Riscos técnicos identificados (seção 20)
- [x] Observações técnicas registradas (seção 21)
- [x] Arquivos envolvidos listados — criar, modificar, referência (seção 22)
- [x] Pronto para geração das TASKS
