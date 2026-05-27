# TASK – Detalhamento da Task

## 1. Identificacao
- **ID**:
- **Nome da Task**:
- **model**: sonnet            <!-- sonnet (default) | opus (área crítica/alta complexidade). Ver minispec-tasks-expert/SKILL.md → "Heurística de modelo e risk". NUNCA haiku. -->
- **risk**: low                 <!-- low | medium | high -->
- **gates**: [qa, tech_review]  <!-- [qa, tech_review] (default) | [qa] | none (task trivial) -->
- **Status**: A Fazer | Em Progresso | Revisao | Concluido
- **Fase**:
- **Dependencias**:
- **Criterio de Conclusao**: Como saber que esta pronta

---

## 2. Objetivo da Task
O que esta task entrega (resultado tecnico direto).

---

## 3. Arquivos Impactados

### 3.1 Arquivos a Criar
| Arquivo | Descricao |
|---------|-----------|
|         |           |

### 3.2 Arquivos a Modificar
| Arquivo | Modificacao |
|---------|------------|
|         |            |

### 3.3 Arquivos de Referencia
| Arquivo | Motivo da Consulta |
|---------|-------------------|
|         |                   |

---

## 4. Detalhes de Implementacao
- [ ] Subtask 1
- [ ] Subtask 2

---

## 5. Testes

<!-- LLM-ONLY: Coluna "Objetivo": Descreva em 1 frase O QUE o teste valida e POR QUE importa. Use o padrao: Verbo + comportamento especifico + condicao. Exemplo: "Verificar que apenas categorias com ativo=1 sao retornadas, ordenadas pelo campo 'ordem'". NAO repita o nome do teste. -->

### 5.1 Testes Unitarios

#### [Camada]: [NomeComponente] (`arquivo_test.go`)

Mock: [interfaces mockadas]

| CT | Teste | Objetivo | Input | Expected | Mock |
|----|-------|----------|-------|----------|------|
| CT-XX | TestMetodo_Cenario | Verificar que [comportamento] quando [condicao] | dados entrada | resultado esperado | dependencias mockadas |

### 5.2 Testes de Integracao

#### [CamadaA + CamadaB] (`arquivo_test.go`)

Setup: [banco in-memory, migracoes, fixtures]

| CT | Teste | Objetivo | Fluxo | Validacao |
|----|-------|----------|-------|-----------|
| CT-XX | TestIntegracao_Cenario | Verificar que [comportamento] quando [condicao] | Passos do fluxo | Assertions esperadas |

### 5.3 Testes E2E

#### Fluxo: [Nome do Fluxo] (CT-XX)
- **Objetivo**: (1 frase descrevendo o que este fluxo E2E valida de ponta a ponta)
- **Pre-condicoes**: (estado inicial do sistema)
- **Passos**:
  1. Passo 1
  2. Passo 2
- **Validacoes**: (assertions sobre dados e estado final)

### 5.4 Cenarios de Erro

| Cenario | Objetivo | Trigger | Codigo/Status | Log Esperado |
|---------|----------|---------|---------------|-------------|
| Descricao do cenario | Verificar que [constraint] impede [operacao] | Acao trigger | Codigo erro | Mensagem log |

### Testes Existentes a Modificar

| Arquivo | Motivo da Modificacao |
|---------|----------------------|
|         |                      |

<!-- LLM-ONLY: Se nenhum teste existente precisa ser modificado, escreva: "Nenhum teste existente impactado." -->

---

## 6. Notas / Observacoes
Anotacoes tecnicas, decisoes, pontos relevantes.

---

## 7. Checklist Final
- [ ] Implementada conforme Scope
- [ ] Testes unitarios criados/atualizados
- [ ] Testes de integracao criados/atualizados
- [ ] Criterio de conclusao atendido
- [ ] Revisada
