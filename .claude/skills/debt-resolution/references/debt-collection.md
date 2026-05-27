# Coleta de Débitos — Procedimento de Extração

> Referência consumida por `SKILL.md` na FASE 1. Define como ler `qa-observations.md` e arquivos de task para coletar débitos elegíveis.

---

## Fontes de débito (em ordem de prioridade)

### 1. `qa-observations.md` (fonte primária)

Arquivo principal. Estrutura típica:

```markdown
# QA Observations — <feature> <version>

## Challenge Session — <data>
...

## Execução minispec-run-tasks — <data>
...

### T5 — gate2 rejeitado | categorias=[naming×2,style×1]
       requires_qa_revalidation=false → skip Gate 1 na correção

### T8 — APROVADO_COM_OBSERVACOES
- MED-001 (code_quality): Duplicata semântica CT-014 vs TestX_ListaVaziaNuncaNull
  Arquivo: internal/api/handlers/franchise_dish/list_my_franchise_handler_test.go:271
  Correção: remover o teste autônomo
- BAIXO-001 (style): inconsistência de naming em variável `x` no handler
  Arquivo: internal/api/handlers/franchise_dish/list_my_franchise_handler.go:42
```

#### Padrões de extração

A skill DEVE detectar débitos via:

- **Marcadores explícitos**: linhas começando com `MED-XXX`, `BAIXO-XXX`, `(code_quality)`, `(naming)`, `(style)`, `(documentation)`, `(dead_code)`, `(imports)`.
- **Veredito `APROVADO_COM_OBSERVACOES`**: tasks listadas sob esse veredito carregam débitos nas linhas seguintes (procurar até o próximo `### T` ou `## ` ou fim do arquivo).
- **Blocos `### T{N} — retry classification` com `requires_qa_revalidation: false`**: indicam que problemas de categorias `code_review_only` foram corrigidos sem re-QA — esses débitos JÁ foram resolvidos, **NÃO recoletar**.

#### Como NÃO incluir débitos já resolvidos

- Se um débito aparece em uma seção e há indicação posterior de correção (ex.: "retry classification" mostrando que o executor corrigiu), pular.
- Se a feature está em uma versão posterior (`v{N+1}` existe e foi gerada por esta skill), olhar o `scope.md` da v{N+1} — débitos lá listados como `Inclui` já estão sendo resolvidos; se já listados como `Fora do escopo`, foram ignorados conscientemente. Em ambos os casos, **não recoletar**.

### 2. `tasks/T*.md` — campo "Notas / Observações" (fallback)

Tasks individuais podem ter seção "## Notas / Observações" ou "## Observações" com débitos anotados durante execução. Estrutura:

```markdown
## 8. Notas / Observações

- [DÉBITO] Refatorar helper `parseCpf` — atualmente repetido em 3 arquivos.
- [TODO] Adicionar comentário explicando regra de NULL no índice composto.
```

#### Padrões de extração (fallback)

- Linhas começando com `- [DÉBITO]`, `- [TODO]`, `- [CLEANUP]`, `- [TECH-DEBT]`.
- Só usar como fallback se `qa-observations.md` resultou em poucos débitos (<2). Razão: notas em tasks são ad-hoc e podem misturar débito real com lembretes do executor.

### 3. NÃO usar

- **TODOs no código** (grep `// TODO`/`# TODO`): fora do escopo desta skill. São débitos do projeto, não da feature específica.
- **Issues do GitHub/GitLab**: fora do escopo — esta skill opera só sobre artefatos do framework agent-spec.
- **CRITICOS/ALTOS** em `qa-observations.md`: esses NÃO chegam aqui como débito anotado. Eles bloquearam o pipeline e foram resolvidos via re-execução da task. Se aparecer um nesta fonte, há bug no gate — **logue um warning e pule** (não confunda débito MEDIO/BAIXO com bug crítico não resolvido).

---

## Schema de cada débito coletado

```yaml
id: D-001                                  # contador local sequencial nesta sessão
origem_task: T8                            # ID da task original (a partir do "### T{N}")
origem_arquivo: qa-observations.md         # ou "tasks/T8.md" se fallback
origem_linha: 142                          # linha onde o débito foi encontrado (audit)
severidade: MEDIO                          # ou BAIXO
categoria: code_quality                    # canônica
arquivo: internal/.../x.go                 # path do código a corrigir
linha: 271                                 # opcional
titulo: "Duplicata CT-014 vs TestX..."     # 1 linha
descricao: "Table-driven CT-014 já..."     # 2-3 linhas
correcao_sugerida: "Remover o teste..."    # ação proposta no gate original
```

---

## Procedimento detalhado

### Passo 1 — Ler `qa-observations.md`

```bash
cat <feature_path>/qa-observations.md
```

(Use a ferramenta `Read` no Claude Code — não rode `cat` via Bash.)

### Passo 2 — Parsear seções

Identifique blocos `### T{N}` ou `## <header>` que contenham:

- Marcadores de severidade: `MED-`, `BAIXO-`.
- Categorias canônicas: `code_quality`, `naming`, `style`, `documentation`, `dead_code`, `imports`.
- Veredito `APROVADO_COM_OBSERVACOES` (lista débitos imediatamente abaixo).

### Passo 3 — Extrair cada débito

Para cada item identificado, monte o YAML do schema acima:

- `id`: contador local, comece em `D-001`.
- `origem_task`: extraído do `### T{N}` mais próximo acima.
- `origem_arquivo`: literalmente `qa-observations.md`.
- `origem_linha`: linha em `qa-observations.md` onde o débito foi achado.
- `severidade`: do marcador (`MED-` → `MEDIO`, `BAIXO-` → `BAIXO`).
- `categoria`: do parêntese `(code_quality)`, etc. Se ausente, inferir do contexto (ex.: "duplicata de teste" → `code_quality`; "naming inconsistente" → `naming`).
- `arquivo` + `linha`: extrair de "Arquivo: <path>:<linha>".
- `titulo`: linha do MED-/BAIXO-XXX (cortar em ~80 chars).
- `descricao`: linhas imediatamente após o título (até próximo marcador `MED-`/`BAIXO-`/`###`).
- `correcao_sugerida`: linha que começa com "Correção:".

### Passo 4 — Deduplicação

Se 2 entradas têm mesmo `(arquivo, linha, titulo_normalizado)`, mantenha apenas a primeira ocorrência (severidade maior vence se houver conflito). Log da deduplicação em `qa-observations.md` da v{N+1}-debits (FASE 4.6).

### Passo 5 — Filtro de elegibilidade

Pule entradas que:

- Já estão em `<feature_path>/../v{N+1}-debits/scope.md` (se a versão existir) — já tratadas.
- Severidade `CRITICO` ou `ALTO` — não deveriam estar aqui; logue warning e pule.
- Categoria fora de `{code_quality, naming, style, documentation, dead_code, imports}` (ex.: `architecture`, `security`, `tests`, `logic`, `data_handling`, `error_handling`, `performance`, `concurrency`, `adr_compliance`) — essas são `revalidation_required` e deveriam ter bloqueado o pipeline. Logue warning e pule.

### Passo 6 — Resultado

Lista ordenada por `(origem_task ascendente, id ascendente)`. Apresente count + breakdown por categoria ao usuário antes de invocar o especialista (FASE 2).

---

## Casos especiais

### `qa-observations.md` muito grande (>500 linhas)

Use `Grep` antes de `Read` para localizar marcadores de débito:

```
Grep(pattern="MED-|BAIXO-|APROVADO_COM_OBSERVACOES|code_quality|naming|style|documentation|dead_code|imports", path=<qa-observations-path>, output_mode="content", -n=true, -B=2, -A=10)
```

Resultado: linhas com contexto suficiente para extrair cada débito sem carregar o arquivo inteiro.

### Múltiplas execuções da mesma task (retries)

Se T8 aparece 3 vezes (3 retries), considere apenas a **última** execução (veredito final). Débitos das tentativas intermediárias foram corrigidos ou viraram parte do veredito final.

### `qa-observations.md` sem estrutura padrão

Algumas features podem ter `qa-observations.md` mais livre (ex.: Challenge Sessions, anotações manuais). Nesses casos:

- Procure por listas com `[DÉBITO]`, `[TODO]`, `[CLEANUP]`.
- Se não achar nada estruturado → recorra ao fallback (Fonte 2: notas nas tasks).
- Se ainda zero → aborte limpamente: "Sem débitos elegíveis em `<feature_path>/qa-observations.md`."

---

## Output esperado da FASE 1

JSON em memória do orquestrador (não escreve arquivo) para passar à FASE 2:

```json
{
  "feature": "cadastro-pratos-franquia",
  "version_origem": "v1",
  "total_coletado": 7,
  "debitos": [
    {
      "id": "D-001",
      "origem_task": "T8",
      "origem_arquivo": "qa-observations.md",
      "origem_linha": 142,
      "severidade": "MEDIO",
      "categoria": "code_quality",
      "arquivo": "internal/api/handlers/franchise_dish/list_my_franchise_handler_test.go",
      "linha": 271,
      "titulo": "Duplicata semântica CT-014 vs TestX_ListaVaziaNuncaNull",
      "descricao": "Table-driven CT-014 já cobre o cenário; teste autônomo é redundante.",
      "correcao_sugerida": "Remover o teste autônomo TestX_ListaVaziaNuncaNull"
    }
  ]
}
```

Esse JSON alimenta diretamente o `<DÉBITOS_JSON>` do prompt do especialista (FASE 2).
