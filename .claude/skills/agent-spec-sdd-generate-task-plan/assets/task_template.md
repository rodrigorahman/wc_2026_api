# TASK – Detalhamento da Task

## 1. Identificação
- **ID**:
- **Nome da Task**:
- **model**: sonnet            <!-- sonnet (default) | opus (área crítica/alta complexidade). Ver agent-spec-sdd-generate-task-plan/SKILL.md → "Heurística de modelo e risk". NUNCA haiku aqui. -->
- **risk**: low                 <!-- low | medium | high. Ver SKILL.md → mesma seção. -->
- **gates**: [qa, tech_review]  <!-- [qa, tech_review] (default) | [qa] (pula Tech Review) | none (task trivial, só docs/config). Ver SKILL.md → "Fast-path gates". -->
- **Responsável**:
- **Status**: A Fazer | Em Progresso | Revisão | Concluído
- **Fase**:
- **Dependências**:
- **Símbolos públicos criados**:        <!-- tipos/funções/interfaces/constantes que OUTRAS tasks podem consumir (ex.: `service.EmailSender`, `dto.CartItem`). N/A se nenhum. Alimenta a derivação do flag de paralelismo e o guard de disjunção de símbolo do executor (ver agent-spec-workflow-rules.md → "Invariante de Paralelismo"). -->
- **Símbolos consumidos de outras tasks**: <!-- símbolo → task de origem (ex.: `service.EmailSender ← T5`). N/A se nenhum. Se consumir um símbolo nascido em task posterior, REORDENE (Regra 10a). -->
- **User Stories Relacionadas**: (US-XX do PRD)

---

## 2. Objetivo da Task
Explique o que deve ser entregue ao final desta task (resultado técnico direto, não comportamento do usuário).

---

## 3. Descrição Detalhada
Explique COMO implementar, baseado no TECH_SPEC:
- O que deve ser criado
- O que deve ser modificado
- Fluxo técnico envolvido
- Regras de implementação específicas
- Decisões técnicas já tomadas

<!-- LLM-ONLY: A descricao deve ser objetiva, clara e de engenharia. -->

---

## 4. Aceite Técnico (critérios objetivos)
A task estará concluída quando:
- [ ] Estrutura implementada conforme SPEC
- [ ] Fluxo técnico funcional
- [ ] Erros corretamente tratados
- [ ] Testes da task criados (quando aplicável)
- [ ] Código revisado e aprovado
- [ ] Nenhuma quebra nos fluxos existentes

---

## 5. Arquivos Impactados

### 5.1 Arquivos a Criar
| Arquivo | Descrição |
|---------|-----------|
|         |           |

### 5.2 Arquivos a Modificar
| Arquivo | Modificação |
|---------|------------|
|         |            |

### 5.3 Arquivos de Referência
| Arquivo | Motivo da Consulta |
|---------|-------------------|
|         |                   |

---

## 6. Testes

<!-- LLM-ONLY: Coluna "Objetivo": Descreva em 1 frase O QUE o teste valida e POR QUE importa. Use o padrao: Verbo + comportamento especifico + condicao. Exemplo: "Verificar que apenas categorias com ativo=1 sao retornadas, ordenadas pelo campo 'ordem'". NAO repita o nome do teste — o objetivo deve dar contexto que o nome sozinho nao da. -->

### 6.1 Testes Unitários

#### [Camada]: [NomeComponente] (`arquivo_test.go`)

Mock: [interfaces mockadas]

| CT | Teste | CA | Objetivo | Input | Expected | Mock |
|----|-------|----|----------|-------|----------|------|
| CT-XX | TestMetodo_Cenario | CA-XX | Verificar que [comportamento esperado] quando [condição] | dados de entrada | resultado esperado | dependências mockadas |

### 6.2 Testes de Integração

#### [CamadaA + CamadaB] (`arquivo_test.go`)

Setup: [banco in-memory, migrações, fixtures]

| CT | Teste | CA | Objetivo | Fluxo | Validação |
|----|-------|----|----------|-------|-----------|
| CT-XX | TestIntegracao_Cenario | CA-XX | Verificar que [comportamento] quando [condição] | Passos do fluxo | Assertions esperadas |

### 6.3 Testes E2E

#### Fluxo: [Nome do Fluxo] (CT-XX)
- **CA**: CA-XX, CA-YY
- **Objetivo**: (1 frase descrevendo o que este fluxo E2E valida de ponta a ponta)
- **Pré-condições**: (estado inicial do sistema)
- **Passos**:
  1. Passo 1
  2. Passo 2
- **Validações**: (assertions sobre dados e estado final)

### 6.4 Cenários de Erro

| Cenário | CA | Objetivo | Trigger | Código/Status | Log Esperado |
|---------|----|----------|---------|---------------|-------------|
| Descrição do cenário | CA-XX | Verificar que [constraint] impede [operação] e retorna erro adequado | Ação que dispara o erro | Código de erro esperado | Mensagem de log esperada |

---

## 7. Notas / Observações
Anotações técnicas, decisões, pontos relevantes.

---

## 8. Checklist Final
- [ ] Implementada conforme SPEC
- [ ] Testes unitários criados/atualizados
- [ ] Testes de integração criados/atualizados
- [ ] Aceite técnico atendido
- [ ] Revisada
- [ ] Integrada à branch principal
