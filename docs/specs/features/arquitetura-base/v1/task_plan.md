# TASK PLAN – Plano de Execução das Tasks

## 1. Identificação
- **Feature/Projeto**: Arquitetura Base — API do Álbum da Copa do Mundo (backend Go/gRPC)
- **Responsável (Tech Lead)**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Status**: Concluído
- **TECH_SPEC**: `docs/specs/features/arquitetura-base/v1/tech_spec.md`
- **PRD**: `docs/prds/features/arquitetura-base/v1/prd.md`

---

## 2. Objetivo do Task Plan
Entregar a fundação de uma API gRPC em Go (go-standard-layout + uber-fx, SQLite via modernc pure-Go, sqlc, golang-migrate, protovalidate) com autenticação funcional (cadastro/login, bcrypt + JWT HS256) e o módulo de referência arquitetural NationalTeam (read-only). Ao final, o sistema sobe via fx, valida CA-01..CA-10 por testes (unit + integração + E2E) e compila cross-platform (`CGO_ENABLED=0`).

---

## 3. Macro-Fases (alto nível)
- **Fase 1 – Fundação técnica (tooling, contrato, infra base)**
  - Objetivo: tooling/build, contratos proto + stubs, config (fail-fast) e logger.
  - Tasks: T1, T2, T3, T4
- **Fase 2 – Persistência**
  - Objetivo: conexão SQLite (modernc + PRAGMAs), migrations, queries sqlc + código gerado, fx module e helper TestNewDB.
  - Tasks: T5
- **Fase 3 – Segurança & Domínio Auth**
  - Objetivo: TokenManager (JWT), AuthService (bcrypt/anti-timing), UserRepository, interceptors (JWT + protovalidate), AuthHandler + fx.Module.
  - Tasks: T6, T7, T8, T10, T11
- **Fase 4 – Módulo NationalTeam (referência)**
  - Objetivo: módulo read-only completo (repository/service/handler/module) como template arquitetural.
  - Tasks: T9
- **Fase 5 – Composição & E2E**
  - Objetivo: composition root (cmd/server), cadeia de interceptors, testes E2E (bufconn) e verificação de build cross-platform.
  - Tasks: T12

---

## 4. Lista de Tasks (visão macro)
| ID  | Nome da Task | Arquivo | Fase | Dependências | Pode Rodar em Paralelo? | Status |
| --- | ------------ | ------- | ---- | ------------ | ----------------------- | ------ |
| T1  | Tooling, dependências e build (buf, sqlc, Makefile) | [T1](tasks/T1.md) | 1 | — | Não | Concluído |
| T2  | Contratos proto + geração de stubs | [T2](tasks/T2.md) | 1 | T1 | Sim | Concluído |
| T3  | Config (viper) com fail-fast de JWT_SECRET | [T3](tasks/T3.md) | 1 | T1 | Sim | Concluído |
| T4  | Logger zap + interceptor de logging | [T4](tasks/T4.md) | 1 | T1 | Sim | Concluído |
| T5  | Persistência SQLite + migrations + sqlc + fx module | [T5](tasks/T5.md) | 2 | T1 | Não | Concluído |
| T6  | TokenManager JWT HS256 + Clock | [T6](tasks/T6.md) | 3 | T1 | Não | Concluído |
| T7  | AuthService (interfaces, bcrypt, anti-timing) | [T7](tasks/T7.md) | 3 | T6 | Sim | Concluído |
| T8  | UserRepository (sqlc) concreto | [T8](tasks/T8.md) | 3 | T5 | Sim | Concluído |
| T10 | Interceptors auth (JWT) + protovalidate | [T10](tasks/T10.md) | 3 | T2, T6 | Sim | Concluído |
| T11 | AuthHandler (Register/Login/GetMe) + fx.Module auth | [T11](tasks/T11.md) | 3 | T6, T7, T8, T10 | Não | Concluído |
| T9  | Módulo NationalTeam (repo+service+handler+module) | [T9](tasks/T9.md) | 4 | T2, T5 | Não | Concluído |
| T12 | Composition root + E2E + build cross-platform | [T12](tasks/T12.md) | 5 | T3, T4, T9, T10, T11 | Não | Concluído |

> **Frontmatter por task** — model / risk / gates:
> T1 sonnet/low/none · T2 sonnet/medium/[qa,tech_review] · T3 sonnet/medium/[qa,tech_review] · T4 sonnet/low/[qa] · T5 opus/medium/[qa,tech_review] · T6 opus/high/[qa,tech_review] · T7 opus/high/[qa,tech_review] · T8 sonnet/medium/[qa,tech_review] · T9 sonnet/low/[qa] · T10 opus/high/[qa,tech_review] · T11 opus/high/[qa,tech_review] · T12 sonnet/medium/[qa,tech_review]

---

## 5. Rastreabilidade: User Stories → Tasks

| User Story (PRD) | Definição Técnica (SPEC) | Tasks Relacionadas | Status |
| ---------------- | ------------------------ | ------------------ | ------ |
| US-01 (criar conta) | RPC Register + validação + hash + persistência (§4.1, §5.1, §7.2) | T2, T7, T8, T11, T12 | A Fazer |
| US-02 (login) | RPC Login + emissão JWT (§4.1, §5.1, §11.1) | T6, T7, T10, T11, T12 | A Fazer |
| US-03 (escolher seleção) | ListNationalTeams + FK selecao_id (§4.1, §6.1, §7.2) | T2, T5, T9, T12 | A Fazer |
| US-04 (senha segura) | bcrypt cost 12 + coluna senha_hash (§11.3, §7.2) | T5, T7, T8 | A Fazer |
| US-05 (fundação 3 SOs) | fx + modernc + build CGO_ENABLED=0 (§3, §7.1, §16.2) | T1, T3, T4, T5, T12 | A Fazer |
| US-06 (módulo referência) | Módulo internal/nationalteam como template (§3.2) | T9 | A Fazer |

> Todas as 6 User Stories cobertas. US-01/US-02 aparecem em 5 tasks por serem fluxos que atravessam todas as camadas da fundação (contrato→service→repo→handler→E2E) — esperado em feature fundacional; cada task entrega uma camada distinta e testável.

### Rastreabilidade complementar: CA → Tasks (origem dos CTs)
- CA-01: T7, T8, T11, T12 · CA-02: T7, T8 · CA-03: T10, T12 · CA-04: T7, T8, T9 · CA-05: T6, T7, T10 · CA-06: T7, T10, T11 · CA-07: T6, T10, T12 · CA-08: T7, T12 · CA-09: T3, T5, T12 · CA-10: T5, T9, T12

---

## 6. Dependências Gerais
- **T1 é gateway**: todas as demais dependem do `go.mod`/tooling.
- **Fase 1 paralela após T1**: T2, T3, T4 tocam paths disjuntos (`proto/`+`internal/pb/`, `internal/config/`, `internal/logger/`).
- **Fase 3 — ordenação interna**: T6 primeiro (gateway de Clock/TokenManager); depois T7 (dep T6), T8 (dep T5, independente de T6) e T10 (dep T2+T6) podem rodar em paralelo (sub-pacotes disjuntos sob `internal/auth/`); T11 fecha a fase (dep T6+T7+T8+T10).
- **T11 NÃO depende de T9**: a interface `NationalTeamRepository` é declarada no consumidor (AuthService, T7); o bind do concreto (T9) ocorre no composition root (T12).
- **T9 (Fase 4)** depende apenas de T2+T5 — poderia adiantar-se, mas foi isolado como módulo de referência.
- **T12 (Fase 5)** integra tudo: depende de T3, T4, T9, T10, T11.

---

## 7. Critérios de Conclusão da Feature
A feature será considerada concluída quando:
- [x] Todas as 12 tasks completas e testes verdes (unit + integração + E2E).
- [x] CA-01..CA-10 cobertos por ≥1 CT cada (35 CTs §19 + 17 CTs de fallback).
- [x] Servidor sobe via fx com fail-fast de config e cadeia de interceptors correta.
- [x] `make build-all` compila os 4 targets com `CGO_ENABLED=0` (CT-033).
- [x] Todas as User Stories cobertas (seção 5).
- [x] Conformidade com ADR-0001 (modernc/CGO off) e ADR-0002 (fx + go-standard-layout).

---

## 8. Riscos & Mitigações
- **FK não enforced sem `PRAGMA foreign_keys=ON`** → habilitar no `db.go` (T5); CT-019 (T8) valida com dados reais.
- **Reintrodução de CGO via driver `sqlite3`** → usar driver `sqlite` (modernc) em golang-migrate (T5); CT-033 (T12) prova `CGO_ENABLED=0`.
- **Determinismo de testes com `time.Now()`** → interface `Clock` injetável (T6) usada por T7/T10/T12.
- **Enumeração de e-mails por timing** → bcrypt dummy no Login (T7, CT-009).
- **`JWT_SECRET` fraco** → fail-fast no boot (T3, CT-F01..F03; abort no OnStart T12).
- **Erro de DI só em runtime (fx)** → CT-035 (smoke de wiring, T12).

---

## 9. Checklist Final
- [x] Task Plan completo
- [x] Tasks mapeadas (12 tasks em 5 fases)
- [x] Dependências validadas (DAG acíclico)
- [x] Rastreabilidade User Stories → Tasks preenchida (6/6)
- [x] Pronto para execução
