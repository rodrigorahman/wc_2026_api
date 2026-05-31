# QA / Tech Review Observations — esqueci-a-senha/v1

## Challenge Session — 2026-05-31 (artifact: tech_spec.md)

- Questões processadas: 4 (interrogadas) + verificações de código (national_team_id, adapter, migrations)
- Conflitos de terminologia resolvidos: 0
- Decisões implícitas explicitadas: 3
  - `password_change_required` é derivado (sem coluna de flag) — reconcilia "flag de troca" do tech-alignment §5
  - "Troca obrigatória" = sinalização ao cliente, sem enforcement de backend (RN9-coerente)
  - Timing leak do 2º branch do Login: equalização total avaliada e rejeitada (custo bcrypt), risco aceito mantido
- Contradições com código corrigidas (inline): 1 (alto impacto)
  - `userRepositoryAdapter` (module.go) copia campo a campo repository.User→service.User; spec não listava propagação dos campos temp nem os campos no struct service.User. Sem isso o 2º branch do Login nunca enxerga a temporária (CT-023/CT-032 quebrariam). Atualizado §3.2, §6.2, §22.2.
- Verificações que confirmaram a spec (sem mudança):
  - §7.2 correta: migration 000004 fez `ALTER TABLE users DROP COLUMN national_team_id`; colunas restantes batem.
  - resend-go ausente do go.mod (esperado — dependência a adicionar).
- Termos canonizados no glossário: 3 (nível FEATURE — arquivo criado)
  - senha temporária, troca obrigatória, recuperação de acesso
- Candidatos a ADR sinalizados: 0 (decisões do challenge são feature-scoped — falham C1)
- ADRs sugeridos para criação: 0
