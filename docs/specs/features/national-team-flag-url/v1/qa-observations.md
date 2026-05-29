# QA / Tech Review Observations — national-team-flag-url/v1

## TC-001 — Adicionar bandeira_url ao domínio NationalTeam

- [run] executor resolvido: `go-backend-implementer` (origem: descoberta interativa)
- [run] executor_discipline injetado (fonte: agent-spec-executor-discipline.md)
- [run] resolução: executor=opus (declarado no frontmatter) | gates=[qa, tech_review] (declarado) | qa=opus, tech=opus (rule: db_migrations + api_contracts critical_path, risk=high)
- [run] base_sha=9401a80622e94451690d12b884175d6b0970aa9c
- [exec] executor concluiu: 2 criados, 11 modificados (4 gerados), 14/14 CTs; make test/build-all verdes (relato)
- [scope] executor tocou arquivo FORA da lista 8.3: `internal/auth/repository/user_repository_test.go` (Steps(-1)→Steps(-2)); justificativa: migration 000005 tornou stale o teste de down da 000004 (Iron Rule 3). A ser avaliado pelos gates.

### TC-001 — Gate 1 (QA)
- veredito: APROVADO_COM_OBSERVACOES (nota 9); 0 críticos, 0 altos, 2 médios, 1 baixo. tocou_area_critica=true; security_flags=[].
- MED-001 (tests): `user_repository_test.go` usa assert de erro por substring (`require.Contains "UNIQUE"/"FOREIGN KEY"`) vs `errors.Is` exigido por `.claude/rules/testing.md`. Débito (teste de domínio adjacente pré-existente).
- MED-002 (code_quality/scope): `user_repository_test.go` fora da §8.3 — ruling definitivo no Tech Review.
- BAIXO-001 (code_quality): magic strings de IDs/URLs de seed replicados entre testes.
- rule_candidates (qa): repeated_assertion_shape (asserts FlagURL), repeated_fixture (seed Brasil ID cross-package).

### TC-001 — Nota de infraestrutura git (workflow)
- base_sha 9401a80 ("feat: init project") contém SÓ os 112 arquivos `.claude/`; todo `internal/`, `proto/`, `sqlc.yaml` está UNTRACKED. `git diff base_sha -- path` retorna arquivos integrais, não diff cirúrgico. Diff-isolation degradada (não quebrada). Recomendação ao usuário: criar baseline commit do código-fonte para que próximas TaskCards tenham diff cirúrgico real.

### TC-001 — Gate 2 (Tech Review)
- status: approved; problems: []. RULING scope_deviation (user_repository_test.go Steps(-1)→Steps(-2)): JUSTIFICADO (Iron Rule 3), não over-reach.
- ADR-0001/0002/0004 conformes; MED-001 do QA confirmado como pré-existente fora de escopo (registrado em observacoes, não bloqueante).
- 16 task_paths staged (git add) após aprovação de ambos os gates. Sem memória lazy (sem rejeição) — nada a limpar.

### TC-001 — encerramento
- [run] rule_candidates: 2 sinais persistidos em rule-candidates.md (qa=2, staff=0, orquestrador=0)
- Status final: CONCLUÍDO (QA APROVADO_COM_OBSERVACOES + Tech Review approved). Tentativas: 1 (sem loop de correção).

### TC-001 — refactor pós-pipeline (decisão do usuário, fora dos gates)
- Decisão do usuário: eliminar a ponte de idioma do sqlc (`rename` cresceria demais). Escolha: schema do banco em INGLÊS, sem bridge. Supersede ADR-0004 → criada ADR-0005. Estratégia: reescrever migrations 000001-000005 (greenfield, sem produção). Processo: direto (sem gates formais).
- Schema migrado: `selecoes→national_teams`, `usuarios→users`, `usuario_selecoes→user_national_teams`; colunas `nome→name`, `bandeira_url→flag_url`, `nome_completo→full_name`, `senha_hash→password_hash`, `criado_em→created_at`, `usuario_id→user_id`, `selecao_id→national_team_id`. Dados (nomes de seleções) mantidos em pt-BR.
- `sqlc.yaml`: bloco `rename` removido (mantido só `overrides` de TIMESTAMP). sqlc regenerado; structs: `NationalTeam{ID,Name,FlagUrl}`, `User{...}`, `UserNationalTeam{UserID,NationalTeamID}`.
- Docs: ADR-0004 status=superseded (superseded_by 0005), ADR-0005 criada, INDEX reindexado, rules `language-naming.md`/`persistence-sqlite.md` e `CLAUDE.md` atualizados.
- Verificação: `make test` verde (suíte completa), `make build-all` verde (4 targets CGO off), `go vet` limpo. Nenhum resíduo pt-BR de schema em código/SQL.
