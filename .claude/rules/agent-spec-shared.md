---
description: Regras compartilhadas entre os workflows do framework agent-spec (SDD, miniSpec, TaskCard, ADR e skills auxiliares de spec) — apenas conteúdo cross-workflow. Regras específicas de cada workflow ficam nos respectivos arquivos workflow-rules. Carregada automaticamente quando qualquer skill desses workflows está em execução.
paths:
  - ".claude/skills/sdd-*/**"
  - ".claude/skills/minispec-*/**"
  - ".claude/skills/taskcard-*/**"
  - ".claude/skills/adr-*/**"
  - ".claude/skills/pre-refinement/**"
  - ".claude/skills/generate-tech-alignment/**"
  - ".claude/skills/challenge-spec/**"
  - ".claude/skills/backend-contract-handoff/**"
---

# Regras Compartilhadas — Workflows agent-spec

> Carregada automaticamente quando qualquer skill SDD, miniSpec, TaskCard, ADR ou skill auxiliar de spec (pre-refinement, generate-tech-alignment, challenge-spec, backend-contract-handoff) está em execução. Centraliza apenas conteúdo **cross-workflow** (válido em qualquer um deles). Conteúdo específico de cada workflow fica nos respectivos `agent-spec-{workflow}-workflow-rules.md`.

---

## Regra de Acentuação (pt-BR)

Todo artefato gerado pelos skills (`prd.md`, `tech_spec.md`, `task_plan.md`, `tasks/TN.md`, `intent.md`, `scope.md`, `taskcard.md`, ADRs em `docs/adr/`, `domain-glossary.md`, `pre-refinement.md`, `tech-alignment.md`) é em português brasileiro com acentuação correta:

- Títulos/seções: `Descrição`, `Restrições`, `Instruções`, `Validação`, `Configuração`
- Corpo: `não`, `é`, `está`, `será`, `também`, `através`, `após`, `até`, `único`
- Termos técnicos em pt-BR: `autenticação`, `paginação`, `migração`, `funcionalidade`

Apenas nomes de código (funções, variáveis, structs, pacotes) permanecem em inglês sem acento.
