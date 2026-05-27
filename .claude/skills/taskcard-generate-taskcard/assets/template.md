# TASKCARD - Execucao Rapida (com Guardrails LLM)

## 1. Identificacao
- **ID**: TC-XXX
- **Nome da Task**: [nome descritivo]
- **model**: sonnet            <!-- sonnet (default) | opus (área crítica/complexa). Ver taskcard-expert/SKILL.md → "Heurística de modelo e risk". NUNCA haiku. -->
- **risk**: low                 <!-- low | medium | high -->
- **gates**: [qa, tech_review]  <!-- [qa, tech_review] (default) | [qa] | none (task trivial: docs/config sem código executável) -->
- **Responsavel**: [quem executa]
- **Data**: [data de criacao]
- **Status**: A Fazer | Em Progresso | Revisao | Concluido
- **Dependencias**: (IDs de outras tasks, se houver)
- **Relacionados**: (Issue, PR, Discussao, Link, Documento...)

---

## 2. Contexto
Explique em 2-4 linhas por que essa task existe e o que motivou a execucao.

---

## 3. Objetivo da Task
Explique o que deve ser entregue ao final desta task (resultado tecnico direto, sem "historia de produto").

---

## 4. Escopo
### 4.1 Inclui
- [ ] Item incluido A
- [ ] Item incluido B

### 4.2 Fora do escopo
- [ ] Item fora A
- [ ] Item fora B

---

## 5. Descricao de Execucao (COMO fazer)
Explique como implementar:
- O que criar
- O que modificar
- Onde mexer
- Regras tecnicas relevantes (curtas e objetivas)

### 5.1 Exemplo de Payload (OBRIGATÓRIO se a TaskCard expõe endpoint com payload parcial)

> Quando a TaskCard implementa um endpoint `PUT`/`PATCH` com atualização parcial (qualquer campo pode estar ausente, multipart parcial), registre aqui:
>
> 1. Exemplo de body/form **mínimo** (só com o campo mais comumente atualizado isoladamente).
> 2. Observação literal: "campos ausentes são ignorados; **sem `binding:"required"`** / **sem `@NotNull`** / **sem `validates_presence_of`** no Request — apenas o ID na URL é obrigatório".
> 3. Diferenciar do verbo `POST` correspondente — no `POST` campos obrigatórios continuam obrigatórios; no `PUT`/`PATCH` parcial, **não**.
>
> Marque "N/A — sem payload parcial" se não aplicável.

---

## 6. Guardrails de Execucao (LLM) - DEVE / NAO DEVE
> Quebrar qualquer item aqui **invalida a task**.

### 6.1 DEVE
- Seguir padroes ja existentes no projeto
- Alterar apenas arquivos listados em "Arquivos/Areas Impactadas"
- Manter contratos publicos (APIs, assinaturas) inalterados

### 6.2 NAO DEVE
- Nao refatorar fora do escopo
- Nao criar abstracoes genericas "por precaucao"
- Nao introduzir novas dependencias sem justificar e registrar

---

## 7. Passos Sugeridos (checklist executavel)
- [ ] Passo 1
- [ ] Passo 2
- [ ] Passo 3

---

## 8. Arquivos Envolvidos

### 8.0 Visão em Árvore

<!-- LLM-ONLY: Gere uma árvore ASCII de TODOS os arquivos das seções 8.1–8.3 organizados por diretório.
  Marque cada arquivo com: [N] Novo  [M] Modificado  [R] Referência (somente leitura)
  Use os caracteres: ├── └── │ (não use + ou -)
  Exemplo:

  internal/
  ├── user/
  │   ├── handler.go      [N]
  │   └── service.go      [M]
  └── db/
      └── 000002.up.sql   [N]
  go.mod                  [R]

  Legenda: [N] Novo  [M] Modificado  [R] Referência
-->

```
(treeview gerado pelo LLM aqui)
```

Legenda: `[N]` Novo &nbsp; `[M]` Modificado &nbsp; `[R]` Referência

---

### 8.1 Arquivos Existentes (leitura/referencia)
Arquivos que o executor DEVE ler antes de comecar. Evita scans desnecessarios no codebase:
- `path/to/existing1` — [por que ler: padrao a seguir, interface a implementar, etc.]
- `path/to/existing2` — [por que ler]

### 8.2 Arquivos a Criar
- `path/to/new1` — [descricao do que sera criado]

### 8.3 Arquivos a Modificar
- `path/to/modify1` — [o que sera alterado]

---

## 9. Aceite Tecnico (criterios objetivos)
A task estara concluida quando:
- [ ] Objetivo atingido conforme secao 3
- [ ] Guardrails respeitados (secao 6)
- [ ] Codigo compila / roda sem erros
- [ ] Nenhuma quebra nos fluxos existentes
- [ ] Padroes do projeto respeitados
- [ ] Revisao realizada (quando aplicavel)

---

## 10. Testes

### 10.1 Testes Existentes a Modificar
Testes que ja existem e precisam ser atualizados por causa das mudancas desta task:
- `path/to/existing_test_file` — [o que precisa mudar: novos cenarios, mocks atualizados, fixtures alteradas, etc.]

### 10.2 Testes a Criar
Novos testes que devem ser criados para cobrir as mudancas desta task:
- `path/to/new_test_file` — [descricao: o que testar, cenarios de sucesso e erro]

### 10.3 Cenarios Obrigatorios
Lista de cenarios que DEVEM ser cobertos pelos testes:
- [ ] Cenario de sucesso (caminho feliz)
- [ ] Cenario de erro (validacao, not found, etc.)
- [ ] Cenarios de borda (limites, valores nulos, etc.)

### 10.4 Padroes de Teste
Referencia dos padroes de teste a seguir (identificados no Passo Zero):
- **Framework**: [detectado — ex: testify, jest, pytest, flutter_test]
- **Convencao de nomes**: [detectada — ex: Test<Layer>_<Function>_<Scenario>, describe/it, test_<function>]
- **Fixture/Setup**: [detectado — ex: banco in-memory, factory functions, fixtures]
- **Mocks**: [detectado — ex: interfaces com mock, jest.mock, mocktail, mockito]

### 10.5 Cenarios de Erro
Mapeamento de cenarios de erro com detalhes tecnicos:

| Cenario | Trigger | Expected | Codigo/Status |
|---------|---------|----------|---------------|
| [descricao do erro] | [o que causa] | [comportamento esperado] | [codigo gRPC, HTTP status, excecao] |

### 10.6 Rastreabilidade: Aceite Tecnico -> Testes
Mapeamento entre criterios de aceite (secao 9) e testes que os validam:

| # | Criterio de Aceite (secao 9) | Teste(s) Correspondente(s) | Tipo |
|---|------------------------------|---------------------------|------|
| 1 | [criterio] | [TestNome] | [Unitario/Integracao/E2E] |

---

## 11. Notas / Observacoes
Decisoes rapidas, alertas, trade-offs ou qualquer detalhe que ajude o reviewer.
