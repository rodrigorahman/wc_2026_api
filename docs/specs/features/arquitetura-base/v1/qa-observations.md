# QA / Tech Review — Observações (arquitetura-base/v1)

> Log de decisões de pipeline do run sdd-run-tasks.

[run] executor_discipline injetado (fonte: agent-spec-executor-discipline.md)
[run] executor resolvido: go-backend-implementer (origem: descoberta interativa)

## Fase 1 — lote paralelo
[Fase 1] lote paralelo: T2, T3, T4 (paths disjuntos confirmados) | base_sha=9401a80
[Fase 1] staged sequencial: T1 → T2 → T3 → T4
[T1] gates: none (config/build) — executado sem gates. Desvio de escopo aceito: executor criou tools.go + cmd/server/main.go (âncoras de build) fora da lista 5.1/5.2.
[T1] PENDÊNCIA cross-task: protovalidate migrou para module path `buf.build/go/protovalidate` (não github.com/bufbuild/protovalidate-go) — relevante para T10.

## Débito anotado (Tech Review approved_with_observations — low, não-bloqueante)
- [T2] low/project_pattern: buf.yaml usa categoria lint `DEFAULT` (deprecada); buf recomenda `STANDARD`. Cleanup futuro.
- [T3] low/technical_requirement: DB_PATH descrito como "obrigatório" na prosa mas não validado em Load() (critérios objetivos §4 não exigem fail-fast; só JWT_SECRET por §11.6). Esclarecer spec ou validar no composition root (T12).

## Vereditos Fase 1
- T1: executor OK (gates none)
- T2: QA APROVADO (9) → TR approved_with_observations (1 low) — CONCLUÍDA
- T3: QA APROVADO (9, 5/5 testes) → TR approved_with_observations (1 low) — CONCLUÍDA
- T4: QA APROVADO (9) [gates: qa] — CONCLUÍDA

## Fase 2 — T5
[T5] base_sha=9401a80 | executor opus | QA opus APROVADO(9), 5/5 testes verdes (ADR-0001 modernc/CGO off + ADR-0004 confirmados) → TR opus approved_with_observations(1 low) — CONCLUÍDA
[T5] DESVIO JUSTIFICADO: executor modificou sqlc.yaml (reference-only) para corrigir chaves de rename (usuario/seleco) e gerar structs User/NationalTeam. Aceito.
[T5] Débito anotado — low/error_handling: PRAGMA journal_mode=WAL aplicado via Exec descarta retorno; não confirma modo efetivo em runtime (só em teste). Cleanup futuro: usar QueryRow+Scan e validar 'wal'.

## Fase 3 — T6 (gateway)
[T6] base_sha=9401a80 | executor opus | QA opus APROVADO(10), 7/7 CTs verdes (alg-confusion none/RS256 com rigor) → TR opus approved (problems:[]) — CONCLUÍDA

## Fase 3 — lote paralelo T7, T8, T10
[Fase 3] lote paralelo: T7, T8, T10 (paths disjuntos: service/ repository/ interceptor/) | base_sha=9401a80
[Fase 3] staged sequencial: T7 → T8 → T10
[T7]  QA opus APROVADO_COM_OBSERVACOES(9, 9/9 testes) → TR opus approved_with_observations — CONCLUÍDA
[T8]  QA opus APROVADO(9, 4/4 testes) → TR opus approved_with_observations — CONCLUÍDA
[T10] QA opus APROVADO(9, 7/7 testes) → TR opus approved_with_observations — CONCLUÍDA

## Débito anotado Fase 3 (medium/low — não-bloqueante)
- [T7] medium/architecture: divergência de contrato service.UserRepository (CreateUser(ctx,User) error) vs concreto repository (CreateUser(ctx,User) (User,error)). Resolver com ADAPTER em T11.
- [T7] low/code_quality: campo clk injetado mas write-only (exigido pelo Aceite/CT-007).
- [T7] low/testability: fixture teamFound() em testes de Login (seleção irrelevante).
- [T8] medium/testability: asserts de constraint via require.Contains("UNIQUE"/"FOREIGN KEY") acoplam ao texto do driver modernc (frágil em upgrade). ADR-0001 fixa driver, não o texto.
- [T8] low/architecture: divergência de contrato (decisão de design documentada).
- [T10] low/technical_requirement: publicMethods sem validação de formato — risco fail-open se mal configurado em T12.

## ⚠️ CARRY-OVER OBRIGATÓRIO PARA T11/T12
- T11 (adapter fx bind): o adapter concreto→interface DEVE traduzir repository.ErrUserNotFound → service.ErrUserNotFound. Senão errors.Is do Register quebra e duplicidade de e-mail vira codes.Internal em vez de AlreadyExists.
- T11/T12 (interceptor JWT): popular publicMethods com as constantes geradas authv1.AuthService_*_FullMethodName (evita typo → fail-open). Métodos públicos: Register, Login, ListNationalTeams. Protegido: GetMe.

## Rule candidates (Fase 3)
[run] rule_candidates parciais: 2 sinais (qa-validator: repeated_fixture T7, T8)

## Fase 3 — T11 (fecha fase)
[T11] base_sha=9401a80 | executor opus | QA opus APROVADO(9, CT-023+companheiro) → TR opus approved_with_observations(2 low) — CONCLUÍDA
[T11] Carry-over T7/T8 RESOLVIDO: adapter userRepositoryAdapter traduz repository.ErrUserNotFound→service.ErrUserNotFound (protegido por teste de integração).
[T11] Débito: low/testability adapter duplicado (test vs module, defensável); low GetMe parcial.

## ⚠️ CARRY-OVER OBRIGATÓRIO PARA T12 (decomposição)
- GetMe hoje retorna SÓ user_id. Não existe GetUserByID em repository(T8)/service(T7). Para CT-029 (GetMe retorna dados do usuário), T12 DEVE: (a) adicionar query GetUserByID em internal/db/queries/usuarios.sql + regenerar sqlc; (b) método no UserRepository; (c) método de leitura no AuthService; (d) ampliar handler.GetMe; (e) teste. Isso toca arquivos de T5/T7/T8 — escopo aditivo legítimo no composition root/E2E.
- T12 wiring do interceptor JWT: publicMethods = [Register, Login, ListNationalTeams] via constantes authv1.*_FullMethodName; protegido = GetMe. Ordem cadeia: recovery → logging → protovalidate → auth.
- T12 deve bindar NationalTeamRepository concreto (T9) à interface do AuthService.

## Fase 4 — T9
[T9] base_sha=9401a80 | executor sonnet | gates:[qa]
[T9] tentativa 1: QA sonnet REJEITADO (1 alto AP-06, 1 médio, + bug dupla tradução em observação)
[T9] auto-escalado: executor sonnet→opus (rule: last_severity==high)
[T9] tentativa 2: correção opus (ALTO-001+MED-001 + BUG dupla tradução CONFIRMADO e corrigido: removida checagem morta do service, tradução única no adapter, novo module_test.go) → QA sonnet APROVADO(9) — CONCLUÍDA
[T9] memória lazy T9.md deletada (cleanup_on_approval)
[T9] Carry-over T12: nationalteam/service.ErrNationalTeamNotFound difere de auth/service.ErrNationalTeamNotFound — T12 precisa de adapter de bridge entre os sentinelas no bind cruzado.

## Fase 5 — T12 (integração final)
[T12] base_sha=9401a80 | executor opus | QA opus APROVADO(9, SUITE_COMPLETA, 11/11 CTs) → TR opus approved_with_observations(2 low) — CONCLUÍDA
[T12] Expansão aditiva AUTORIZADA: GetUserByID (usuarios.sql+sqlc+repo+service+handler) p/ GetMe completo (CT-029); pacote internal/server (cadeia compartilhada prod/E2E); correção de gap pré-existente de T11 (bind TokenManager concreto→interface, quebrava CT-035).
[T12] Validação geral: go build ./... OK | go test ./... todos verdes (incl test/e2e) | make build-all 4 targets CGO_ENABLED=0 OK (CT-033).
[T12] Débito: low/best_practices (bridge NationalTeam retorna name não usado); low/code_quality (FixedClock duplicado em pacotes de teste).

## ⚠️ SECURITY FOLLOW-UP (não-bloqueante, fora do escopo desta feature)
- Vulnerabilidades de dependência GO-2026-* em google.golang.org/grpc e golang.org/x/net (versões fixadas em T1). Recomendado bump de versão em task/feature futura (go get -u + revalidação). NÃO regressão de nenhuma task.

## Fechamento do run
[run] 12/12 tasks concluídas. 1 task entrou em loop de correção (T9, 2 tentativas, aprovada). 0 tasks bloqueadas.
[run] rule_candidates: 7 sinais persistidos em rule-candidates.md (qa=7: repeated_fixture x4, repeated_assertion_shape x2... + 1). source=qa-validator. staff=0.
