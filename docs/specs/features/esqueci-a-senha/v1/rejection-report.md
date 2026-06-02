# Relatório de Rejeições — esqueci-a-senha/v1

> Gerado pelo orquestrador `agent-spec-sdd-run-tasks` ao fim do run (2026-06-02).
> Objetivo: consolidar os motivos de rejeição/correção nos gates para avaliar correções no **fluxo** (skills, agents, rules, task generator). Não é avaliação do código final — esse passou (make test + make build-all verdes, 10/10 tasks).

## 1. Visão geral

| Métrica | Valor |
|---|---|
| Tasks | 10 (todas concluídas, 0 bloqueadas) |
| Ciclos de gate executados | QA: 13 · Tech Review: 8 |
| Tasks que precisaram de retry | 3 de 10 (T1, T3, T8) |
| Retries totais | 3 (1 por task; nenhuma chegou a 2/3) |
| Reprovações bloqueantes | 3 (T1 QA, T3 Tech Review, T8 QA) |
| `approved_with_observations` (débito não-bloqueante) | T4, T5, T6, T3(final via partial→corrigido) |
| Reorder de plano necessário | 1 (T3 movido para depois de T5) |

Taxa de acerto na 1ª passada: **7/10**. As 3 reprovações têm causas-raiz distintas e **recorrentes o suficiente para virar regra** — detalhadas abaixo.

---

## 2. Reprovações bloqueantes (detalhe)

### R1 — T1 (QA, attempt 1): regressão cross-package de migração
- **Severidade**: crítico · **Categoria**: `tests` (regressão)
- **O quê**: a migração `000009` entrou no topo da pilha. Três testes pré-existentes de *down-migration* em **outros pacotes** (`match`, `nationalteam`) usam `migrator.Steps(-N)` com `N` relativo à profundidade anterior. O executor corrigiu só o teste análogo de `auth` (000004), não detectou os outros 3.
- **Por que passou batido**: o executor rodou testes escopados ao **próprio pacote** (`auth/repository`) — verde — e não a suíte completa. A quebra só aparece em `go test ./...`.
- **Causa-raiz**: acoplamento implícito de testes de migração à profundidade da pilha + executor testando escopo estreito.

### R2 — T3 (Tech Review, attempt 1): asserção tautológica + test-only em produção
- **Severidade**: 1 high (P1) + 3 medium · **Categoria**: `tests` / `code_quality` / `adr_compliance`
- **O quê (P1, bloqueante)**: `assert.True(t, strings.Contains(err, "[ERROR]") || sendErr != nil)` — a 2ª cláusula é sempre verdadeira (o `require.Error` anterior já garante), tornando a asserção infalível (não pega regressão).
- **Divergência entre gates**: o **QA aprovou** essa mesma asserção classificando-a como **medium** (MED-002); o **Tech Review** a classificou como **high**. Rubrica de severidade desalinhada para o mesmo achado.
- **P2 (medium)**: `newResendSenderWithClient` (test-only) declarada em arquivo de **produção** `resend.go` → compila no binário (Iron Law #6). *(mesmo padrão de R3 — ver cluster B)*.
- **P3 (medium)**: nomes de função de teste em **pt-BR** — Tech Review apontou violação de ADR-0005; mas **T1/T4/T5 usaram pt-BR e passaram nos dois gates**. Doutrina inconsistente. (Decisão do orquestrador: não renomear, por consistência; registrado como rule-candidate `convention_drift`.)

### R3 — T8 (QA, attempt 1): setter de auth exportado só para teste
- **Severidade**: 1 high + `security_flag` · **Categoria**: `code_quality` + segurança
- **O quê**: para injetar o `sub` no contexto nos testes do handler, o executor exportou `ContextWithSubject` em `interceptor/auth_jwt.go` (**produção, arquivo auth-sensível**). Implicação: qualquer pacote poderia forjar um contexto autenticado, contornando o JWT. Também é desvio de escopo (T8 declarou só `handler/`).
- **Causa-raiz**: o símbolo `subKey` é unexported; o executor precisava de um seam de teste cross-package e **alcançou a produção** em vez de usar o caminho real do interceptor. Corrigido reescrevendo o teste via interceptor real + TokenManager fake.

---

## 3. Clusters de causa-raiz (o que merece correção no fluxo)

### Cluster A — Migração no topo da pilha quebra down-tests relativos *(R1)*
- **Recorrência**: 1 ocorrência, mas alto impacto (suíte inteira quebra; passa despercebido em teste escopado).
- **Correção de fluxo proposta**:
  1. **Rule de persistência/migração**: padronizar down-tests para **alvo absoluto** (migrar até versão N explícita) em vez de `Steps(-N)` relativo ao topo — elimina o acoplamento na origem.
  2. **Task generator**: para tasks `db_migrations`, injetar no §5.2/Notas um aviso explícito *"adicionar migração no topo da pilha exige auditar todos os `Steps(-N)` de down-tests do repositório"* e listar os arquivos candidatos.
  3. **Executor discipline**: tasks de migração devem rodar a **suíte completa** (não só o pacote), pois o blast radius é cross-package por natureza.

### Cluster B — Código test-only vazando em produção *(R2-P2 e R3)* — **RECORRENTE (2×)**
- **Recorrência**: 2 ocorrências no mesmo run (T3 `newResendSenderWithClient`; T8 `ContextWithSubject`). Mesma origem: **executor precisa de um seam de teste e cria/expõe símbolo em arquivo de produção** em vez de usar `export_test.go` ou o caminho real.
- **Severidade variável**: T8 foi grave (símbolo de auth forjável); T3 foi cosmético. O risco escala com a sensibilidade do arquivo.
- **Correção de fluxo proposta (prioritária)**:
  1. **Reforçar EXECUTOR_DISCIPLINE (Iron Law de testes)**: adicionar regra explícita — *"Precisa de um seam de teste? Use `export_test.go` (mesmo pacote) ou o caminho real do boundary. NUNCA exporte/adicione um símbolo em arquivo de produção só para teste — especialmente em arquivos de auth/security/crypto. Se o símbolo necessário é unexported e o teste é cross-package, monte o cenário pelo caminho real."*
  2. **go-backend-implementer**: a seção "Diretrizes para escrita de testes" já tem Iron Law #6 implícito; tornar explícito o padrão `export_test.go` vs export em produção.
  3. **QA Camada 5 / Tech Review**: ambos já pegam isso, mas com severidades divergentes (T3 medium no QA vs T8 high) — ver Cluster D.

### Cluster C — Mudança de assinatura força tocar arquivos fora do escopo declarado *(T5, T8; também T7 adapter)*
- **Recorrência**: 3 deviações de escopo, todas mecanicamente forçadas: T5 (construtor `NewAuthService` +emailSender/+logger → `module.go` + `auth_handler_test.go`); T8 (interceptor); T7 (adapter no test file). Foram julgadas justificadas pelos gates, mas geram ruído de "escopo declarado vs real".
- **Causa-raiz**: a decomposição em tasks não antecipa o **blast radius de build** de mudanças de assinatura (composition root + callers + testes).
- **Correção de fluxo proposta**:
  1. **Task generator**: quando uma task altera a assinatura de um construtor/função pública, **incluir em §5.2 os callers mecanicamente afetados** (composition root, testes de integração) — ou uma nota autorizando explicitamente tocá-los.
  2. **EXECUTOR_DISCIPLINE (Iron Law #3)**: adicionar nota — *"mudança de parâmetro obrigatório em construtor implica tocar seus callers para o build compilar; isso é 'limpar a própria bagunça', não expansão de escopo — faça cirúrgico e registre em Pendências."* (Reduz falsos-positivos de `scope_deviation`.)

### Cluster D — Divergência de severidade/doutrina entre QA e Tech Review *(R2)*
- **Recorrência**: 2 sinais. (a) Severidade: mesma asserção tautológica = medium no QA, high no Tech Review. (b) Doutrina: nomes de teste pt-BR aceitos em T1/T4/T5, marcados como violação ADR-0005 em T3.
- **Impacto**: inconsistência de gate gera retrabalho (a asserção tautológica teria entrado como débito se só o QA opinasse; o Tech Review bloqueou) e decisões ad-hoc do orquestrador.
- **Correção de fluxo proposta**:
  1. **Alinhar rubrica de severidade** entre `agent-spec-qa-validator` (Camada 5) e `agent-spec-staff-architecture-review` para smells de teste compartilhados (asserção tautológica/oca deveria ter severidade única — sugiro **high/blocking**, pois mascara regressão).
  2. **Resolver ADR-0005 quanto a nomes de função de teste**: decidir explicitamente se identificadores de teste seguem inglês ou se pt-BR é aceito (o codebase de fato usa pt-BR). Já há rule-candidate `convention_drift` registrado para a curadoria.

### Cluster E — Ordenação de interface cross-task no task plan *(reorder T3→após T5)*
- **Recorrência**: 1, resolvido antes de executar (o orquestrador detectou e o usuário decidiu reordenar).
- **Causa-raiz**: o task plan colocou em T3 (Fase 1) uma compile-time assertion (`var _ service.EmailSender = ...`) referenciando uma interface **criada em T5 (Fase 2)**. T3 não compilaria/passaria QA antes de T5.
- **Correção de fluxo proposta**:
  1. **Task plan generator / challenge-spec**: validar **direção de dependência de símbolos** — uma task não pode referenciar (nem em assertion de teste) um símbolo cujo nascimento está numa task posterior. Marcar a dependência explicitamente ou mover a assertion para a task que cria o símbolo.

---

## 4. Priorização sugerida das correções de fluxo

| Prioridade | Correção | Cluster | Artefato a mudar |
|---|---|---|---|
| **Alta** | Regra explícita "seam de teste via export_test.go / caminho real, nunca export em produção (esp. auth)" | B | `executor-discipline.md` + `go-backend-implementer.md` |
| **Alta** | Alinhar severidade de asserção tautológica/oca entre QA e Tech Review (→ blocking) | D | `agent-spec-qa-validator` + `agent-spec-staff-architecture-review` |
| **Média** | Task generator lista callers afetados por mudança de assinatura + nota no executor discipline | C | task generator + `executor-discipline.md` |
| **Média** | Validar direção de dependência de símbolos no task plan | E | task plan generator / `agent-spec-challenge-spec` |
| **Média** | Migração: down-tests com alvo absoluto + suíte completa em tasks db_migrations | A | rule de persistência + task generator + executor discipline |
| **Baixa** | Decidir doutrina ADR-0005 p/ nomes de função de teste (pt-BR vs inglês) | D | ADR-0005 / `language-naming.md` |

---

## 5. O que funcionou bem (manter)

- **Débito-controlado**: medium/low não bloquearam (T4/T5/T6/T7 com observações low avançaram), evitando retrabalho desnecessário; só critical/high dispararam loop. As 3 reprovações foram todas legítimas (high+).
- **CT mutation-killing**: CT-W3 (T9) e CT-032 (T10) foram verificados como mutation-killing pelo QA (removendo a propagação do adapter, falham) — pegaram exatamente o bug mais crítico da feature (propagação temp no adapter).
- **Isolamento de diff por índice staged**: para tasks sequenciais que compartilham `auth_service.go` (T5→T6→T7), usar o índice staged como baseline (`git diff` unstaged) isolou cada delta sem commitar.
- **Auto-escalonamento**: T3 e T8 (sonnet) escalaram para opus na correção por `last_severity=high`, ambos resolvidos em 1 retry.
