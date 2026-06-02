# TASK PLAN – Plano de Execução das Tasks

## 1. Identificação
- **Feature/Projeto**: Esqueci a senha — recuperação de acesso por e-mail
- **Responsável (Tech Lead)**: Rodrigo Rahman
- **Data**: 2026-05-31
- **Status**: Concluído
- **TECH_SPEC**: `docs/specs/features/esqueci-a-senha/v1/tech_spec.md`
- **PRD**: `docs/specs/features/esqueci-a-senha/v1/prd.md`

---

## 2. Objetivo do Task Plan
Implementar a recuperação de acesso por **senha temporária** estendendo o domínio `auth` (gRPC): pedido público com resposta genérica e envio de e-mail best-effort, login com segundo branch de comparação (temp) + sinal `password_change_required`, troca obrigatória validada contra a temporária, e a capacidade de e-mail transacional (ADR-0006) como infra reusável. Estado de recuperação vive em 2 colunas nullable de `users` (migração 000009). Ao final: 2 RPCs novos + 1 campo add-only, build pure-Go (CGO off) verde e os 38 CTs do tech_spec cobertos.

---

## 3. Macro-Fases (alto nível)
- **Fase 1 – Fundação (paralela)**
  - Objetivo: base independente de persistência, contrato e capacidades — paths disjuntos, executável em paralelo.
  - Tasks: T1, T2, T3, T4
- **Fase 2 – Domínio / AuthService (sequencial)**
  - Objetivo: lógica de recuperação no `AuthService`. Sequencial — todas tocam `auth_service.go`.
  - Tasks: T5, T6, T7
- **Fase 3 – Apresentação + Wiring (paralela)**
  - Objetivo: expor os RPCs (handler) e ligar a capacidade de e-mail + propagar os campos temp no adapter + public method.
  - Tasks: T8, T9
- **Fase 4 – E2E (validação ponta-a-ponta)**
  - Objetivo: cobrir o fluxo completo pela cadeia real de interceptors.
  - Tasks: T10

---

## 4. Lista de Tasks (visão macro)
| ID  | Nome da Task | Arquivo | Fase | Dependências | Pode Rodar em Paralelo? | Status |
| --- | ------------ | ------- | ---- | ------------ | ----------------------- | ------ |
| T1  | Persistência — migração 000009, queries sqlc e repository | [T1](tasks/T1.md) | 1 | — | Sim | Concluído |
| T2  | Contrato proto — RPCs + password_change_required | [T2](tasks/T2.md) | 1 | — | Sim | Concluído |
| T3  | Capacidade de e-mail — ResendSender/NoopSender | [T3](tasks/T3.md) | 1 | — | Sim (reordenado: após T5) | Concluído |
| T4  | Config — secrets do Resend (fail-fast condicional) | [T4](tasks/T4.md) | 1 | — | Sim | Concluído |
| T5  | AuthService — base + RequestPasswordRecovery/processRecovery | [T5](tasks/T5.md) | 2 | T1 | Não | Concluído |
| T6  | AuthService.Login — 2º branch da temp + password_change_required | [T6](tasks/T6.md) | 2 | T5 | Não | Concluído |
| T7  | AuthService.ChangePassword — troca validada + notificação | [T7](tasks/T7.md) | 2 | T5 | Não | Concluído |
| T8  | AuthHandler — mappers dos 2 RPCs + campo no Login | [T8](tasks/T8.md) | 3 | T2, T5, T6, T7 | Sim | Concluído |
| T9  | Wiring — bind EmailSender, propagação temp no adapter, public method | [T9](tasks/T9.md) | 3 | T1, T2, T3, T4, T5 | Sim | Concluído |
| T10 | E2E — extensão do bufconn e fluxos ponta-a-ponta | [T10](tasks/T10.md) | 4 | T8, T9 | Não | Concluído |

> **Lote paralelo Fase 1**: T1, T2, T3, T4 — paths disjuntos confirmados (db+repo / proto+gen / infra/email / config). 4 tasks = `MAX_PARALLEL`.
> **Fase 2 sequencial**: T5 → T6 → T7 (todas tocam `service/auth_service.go` e seu `_test.go`).
> **Lote paralelo Fase 3**: T8 (handler/) ∥ T9 (module.go + server.go) — paths disjuntos.

---

## 5. Rastreabilidade: User Stories → Tasks

| User Story (PRD) | Definição Técnica (SPEC) | Tasks Relacionadas | Status |
| ---------------- | ------------------------ | ------------------ | ------ |
| US-01 | `RequestPasswordRecovery` público + resposta genérica | T1, T2, T5, T8, T9, T10 | Concluído |
| US-02 | `processRecovery` gera temp e envia via `EmailSender` | T3, T4, T5, T9 | Concluído |
| US-03 | `password_change_required=true` no `LoginResponse` | T2, T6, T8, T10 | Concluído |
| US-04 | `ChangePassword` → `UpdatePassword` (zera temp) | T2, T7, T8, T10 | Concluído |
| US-05 | Resposta genérica + dispatch assíncrono (anti-enumeração) | T5, T8, T10 | Concluído |
| US-06 | Expiração 15 min via `clock.Now().Before(expires_at)` | T1, T6 | Concluído |
| US-07 | 2º branch (temp) no Login; original sempre válida | T6, T9, T10 | Concluído |
| US-08 | E-mail de notificação best-effort em `ChangePassword` | T3, T7 | Concluído |

> Todas as 8 User Stories cobertas. Regras transversais: CA-08 (gatilho só pós-recovery) em T6/T1; CA-09 (não logar senha) em T3/T5/T7.

---

## 6. Dependências Gerais
- **Contrato de assinatura cross-task**: `EmailSender.Send(ctx, to, subject, body string) error` é fixada em T5 (interface) e espelhada por T3 (concretos) e T9 (bind). `SetTempPassword`/`UpdatePassword` fixadas em T1 (concreto), T5 (interface) e T9 (adapter).
- **Código gerado**: T1 exige `make sqlc`; T2 exige `make proto`. O gerado é commitado.
- **Pré-requisito do E2E determinístico (T10)**: seam de `EmailSender`/`genTempPassword` no `TestNewBufconnServer` — coordenar com T5/T9; fallback aceito (chamar `processRecovery` direto) com a limitação de sincronização registrada.
- **Build**: `resend-go` (T3) deve manter `make build-all` verde (pure-Go, ADR-0001).

---

## 7. Critérios de Conclusão da Feature
A feature será considerada concluída quando:
- [x] Todas as 10 tasks estiverem completas e gates aprovados.
- [x] Os 38 CTs do tech_spec §19 (+ CTs de infra T3/T4) validados (`make test` verde).
- [x] `make build-all` (4 targets, CGO off) verde.
- [x] Critérios técnicos do SPEC atendidos (anti-enumeração, expiração 15 min, troca obrigatória, best-effort).
- [x] Nenhum comportamento divergente do PRD (CA-01..CA-12).
- [x] Todas as User Stories cobertas (tabela seção 5).

---

## 8. Riscos & Mitigações
- **Propagação dos campos temp no adapter (T9)** → bug silencioso que zera a temp no Login. Mitigação: CT-W3/CT-W4 focados + CT-023/CT-032 ponta-a-ponta.
- **Mapeamento NULL das colunas temp (T1)** → panic no scan. Mitigação: `sql.NullString`/`sql.NullTime` + CT-022/CT-038 (regression guard).
- **Determinismo do E2E (T10)** → goroutine de background não sincronizada. Mitigação: seam de `genTempPassword`/`EmailSender` no bufconn ou `processRecovery` direto; limitação registrada.
- **`resend-go` não pure-Go** → quebra build sem CGO. Mitigação: verificar no T3; `make build-all` no gate.
- **Fronteira de expiração `<` vs `<=` (T6/T7)** → reuso de temp expirada. Mitigação: CT-009 (fronteira exata) + CT-013.

---

## 9. Checklist Final
- [x] Task Plan completo
- [x] Tasks mapeadas (10 tasks, 4 fases)
- [x] Dependências validadas
- [x] Rastreabilidade User Stories → Tasks preenchida (8/8)
- [x] Pronto para execução paralela (lotes Fase 1 e Fase 3)
