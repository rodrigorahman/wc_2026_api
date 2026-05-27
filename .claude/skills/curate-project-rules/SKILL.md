---
name: curate-project-rules
description: |
  Decide se uma convenção, decisão ou padrão merece virar regra de projeto
  (em CLAUDE.md, em `.claude/rules/*.md` ou equivalente do projeto host) —
  e, se merece, com que escopo, matcher e forma. Use SEMPRE que o usuário
  for adicionar, atualizar ou auditar regras do projeto; perguntar "vale
  uma regra?", "documenta isso no CLAUDE.md?", "isso é regra?", "adiciona
  ao .claude/rules"; revisar diff de CLAUDE.md ou rules procurando bloat,
  redundância ou conteúdo apodrecido; ou estiver prestes a escrever uma
  linha nova em qualquer rule. Acione também quando ele descrever uma
  convenção/decisão e pedir para "deixar isso registrado" — mesmo sem
  usar a palavra "regra".
when_to_use: |
  - Decidir se um item entra em rule global, rule com escopo, ou em lugar nenhum.
  - Escrever uma linha nova em CLAUDE.md ou em rule do projeto host.
  - Definir/ajustar o matcher (escopo de carregamento) de uma rule.
  - Revisar/limpar rules existentes (auditoria, "isso ainda faz sentido?").
  - Receber feedback "X foi feito errado" e decidir se vira regra.
  - Migrar conteúdo entre rule global ↔ rule de escopo.
do_not_invoke_for: |
  - Escrever PRD, spec, tech-spec, ADR, taskcard (conteúdo de feature, não regra global).
  - Atualizar README ou docs do site.
  - Editar memória (memória é estado, não regra).
  - Configurações de hook, permissões ou settings.json.
disable-model-invocation: true
---

# curate-project-rules

> Esta skill é parte do framework agent-spec e roda em **qualquer projeto hospedeiro**. Antes de decidir qualquer coisa, **descubra a convenção do projeto host** — onde moram as rules, que frontmatter elas usam, como expressam matchers. Não force a convenção de outro projeto neste.
>
> Uma regra mal-colocada custa mais do que regra nenhuma — ela polui o contexto de toda futura sessão ou, pior, não carrega quando deveria.

---

## Fase 0 — Discovery do projeto host

Antes de propor qualquer coisa, faça uma varredura rápida (≤60s) para descobrir:

| O que descobrir | Onde olhar | Por quê |
|---|---|---|
| **Onde moram as rules** | `.claude/rules/`, `.cursor/rules/`, `.windsurfrules`, `CLAUDE.md`, `AGENTS.md`, `docs/rules/` — o que existir | Convenção do host. Não invente diretório novo. |
| **Que frontmatter as rules usam** | Abra 1–2 rules existentes e leia o YAML do topo | Replicar o estilo. Campos comuns: `description`, `paths`/`globs`/`applies_to`, `when_to_use`, `auto-load`. |
| **Como expressam matchers** | Mesmo frontmatter — campo de glob/path | Decide se a próxima rule vai precisar/herdar esse campo. |
| **Que rules são globais vs com escopo** | Quais não têm matcher (ou têm matcher trivial tipo `"**"`) vs quais têm globs específicos | Você precisa saber em qual bucket sua rule cai. |
| **Idioma e tom** | Estilo das rules existentes (pt-BR/EN, formal/direto) | Combinar — não criar dissonância. |

Se nada existir ainda, **pergunte ao usuário** o destino antes de criar. Não invente convenção sem confirmar.

> ⚠️ Esta skill vive no framework agent-spec, mas o repo onde ela está agora **não é o projeto host** quando estiver instalada — exceto quando você está editando o próprio framework. Quando instalada em outro projeto, "host" = aquele projeto. Quando editando o framework aqui, "host" = `adi_agent_spec`.

---

## Modelo mental

Três camadas, cada uma com dono:

| Camada | Onde mora (típico) | Estabilidade | Função |
|---|---|---|---|
| **Modelo mental + decisões + comandos globais** | `CLAUDE.md` (raiz) | Estável (meses) | Orientação que todo agente precisa em toda sessão. |
| **Detalhes operacionais por domínio/área** | Rule com matcher (path glob) | Vivo (semanas) | Convenções específicas. Só carrega quando relevante. |
| **Estado, feature em andamento, contexto efêmero** | Memória, PRD, spec, ADR, git log | Volátil (dias) | Não é regra. Não entra. |

A camada do meio **só funciona se o matcher for bom**. Matcher errado → rule fica decorativa.

---

## Teste de fricção (obrigatório antes de adicionar)

Faça as 4 perguntas. Se qualquer uma falhar, **descarte ou reformule**.

### 1. Sem isto, alguém faria errado?

Imagine um agente competente lendo o repo pela primeira vez. Ele faria errado? Se "não, descobriria sozinho", a regra não pega peso.

- ✅ Convenção que diverge do default da linguagem/framework.
- ❌ "Use `defer` para fechar arquivos" — qualquer um que sabe Go já faz.

### 2. Está derivável lendo um arquivo do projeto em 1 minuto?

Se sim, **linke o arquivo** em vez de duplicar a regra.

- ✅ "Handler segue padrão de `<path/exemplar.ext>`" — aponte e pare.
- ❌ Copiar 40 linhas da estrutura — sai de sincronia com o código.

### 3. Vai apodrecer em 3 meses?

Datas, números de ticket, branches, "atualmente em X", "estamos migrando para Y" — apodrece. Ou remova a parte volátil, ou aceite que precisa revisitar.

- ✅ Invariante estável ("pool é singleton").
- ❌ "Atualmente em 000045, indo para 000050" — apodrece na próxima migração.

### 4. Tem o "porquê"?

Regra forte sem racional vira ruído. Edge case → o leitor precisa do motivo.

- ✅ "Pool singleton — recriar causa `Error 1040: Too many connections`."
- ❌ "NUNCA recrie o pool" — sem motivo, ninguém julga edge case.

---

## O que documentar

| Tipo | Característica |
|---|---|
| **Convenção não-inferível** | "Um handler = um caso de uso = um arquivo." Sem regra, copia-se legado. |
| **Decisão arquitetural com racional** | Sempre o porquê (incidente, constraint, vendor lock). |
| **Padrão em ≥2-3 domínios** | Se aparece só uma vez, ainda não é padrão. Espere repetir. |
| **Anti-padrão que queimou** | Causa concreta já observada + alternativa. |
| **Default diferente da linguagem/framework** | Ex.: schema em pt-BR num projeto Go. |
| **Comando/fluxo específico** | Ex.: `make X` antes de compilar; ordem dos N passos. |
| **Tradução entre camadas** | SQLC→DTO, validator→400, `errors.Is`→status code. |

---

## O que NÃO documentar

| Tipo | Por quê |
|---|---|
| **Óbvio do código** | Leitura rápida resolve. Não duplique. |
| **Geral da linguagem/framework** | É linguagem, não regra do projeto. |
| **Estado que muda toda hora** | Branch atual, task em andamento, PR aberto → memória. |
| **Detalhe de feature** | Pertence ao PRD/spec/taskcard. |
| **Já dito em outro lugar** | **Linke**. Não duplique. |
| **Comentário narrativo/histórico** | "Decidido em jan/2024 porque..." apodrece. Git log é a fonte. |
| **Regra sem exemplo real** | Sem arquivo apontado, ainda não é regra. |

---

## Decisão de escopo e matcher

Esta é a parte que **mais importa** e que costuma ser ignorada.

### Toda rule tem uma decisão de escopo

| Escopo | O que é | Custo | Quando vale |
|---|---|---|---|
| **Global** (sem matcher, ou no CLAUDE.md) | Carrega em **toda** sessão | Polui contexto sempre | Vale só se a regra é mesmo transversal e estável. |
| **Por área/path** (matcher com globs) | Carrega **só** quando agente toca o escopo | Não polui fora do escopo, mas pode não disparar quando deveria | Default para detalhes operacionais de um workflow/módulo/domínio. |
| **Sem rule, apenas link no código/CLAUDE.md** | Não carrega — só é lida quando o agente segue o link | Custo mínimo | Quando o conteúdo é raro de precisar ou já existe em código que o agente vai ler de qualquer forma. |

### Quando a rule **precisa** de matcher

- Vale só dentro de um **workflow** (ex.: SDD, miniSpec, TaskCard).
- Vale só dentro de um **módulo/domínio** (ex.: `services/payments/**`).
- Vale só para um **tipo de arquivo** (ex.: `**/*.handler.go`, `**/*.sql`).
- Aplica a **paths bem demarcados** que o frontmatter do projeto host suporta expressar.

### Quando a rule **dispensa** matcher (vai global / CLAUDE.md)

- É **modelo mental** do projeto (mapa de onde mora o quê).
- É **comando** que todo agente precisa em toda sessão (`make X` antes de Y).
- É **decisão de arquitetura cross-cutting** (auth, errors, logging).
- A penalidade de carregar sempre é menor do que a penalidade de **não** carregar quando precisa.

### Como escolher os globs do matcher

1. **Comece estreito.** Glob amplo demais transforma rule de escopo em rule global disfarçada.
2. **Cubra os caminhos onde a regra realmente se aplica.** Não inclua path "porque pode ser útil".
3. **Inclua skills/agents que executam o workflow**, se a convenção do host carrega rules por path de skill (ex.: `.claude/skills/sdd-*/**`).
4. **Não use `**` como matcher.** Se a regra vale sempre, ela é global — declare como global, não como "matcher universal".
5. **Espelhe convenção do host.** Se as outras rules usam `paths:` com lista de globs, use `paths:`; se usam `globs:` ou `applies_to:`, use isso.

### Sinais de matcher errado

- A rule carrega em sessões onde claramente não se aplica → glob amplo demais.
- A rule **não** carrega numa sessão onde deveria → glob estreito demais; faltou path; ou conflito com convenção de carregamento do host.
- Duas rules cobrindo o mesmo glob → consolide ou particione melhor o escopo.

---

## Onde colocar (algoritmo)

Depois que o item passou no teste de fricção, decida na ordem:

1. **Já existe rule equivalente?** → propõe **edição** em vez de criar nova.
2. **A regra é global e estável?** → CLAUDE.md (ou rule global do host, se ele tem essa convenção).
3. **A regra pertence a um domínio/workflow já com rule?** → adicione lá; ajuste o matcher se necessário.
4. **A regra pertence a um domínio/workflow **sem** rule ainda E vai ter ≥3 regras?** → crie rule nova, defina matcher estreito.
5. **A regra é só uma**? → adicione na rule do domínio mais próximo, ou aceite que vai para CLAUDE.md como item solto. Não crie arquivo para uma linha.

---

## Forma da regra (como escrever quando entra)

### Corpo
1. **Curto.** Uma linha, no máximo um parágrafo. Mais que isso → é reference, não rule.
2. **Com exemplo do código real** (`path/arquivo.ext:linha`). Sem exemplo → ainda não é regra.
3. **Com "porquê" explícito** em regras críticas (pool, soft delete, segurança, anti-padrão que queimou).
4. **Linkada.** Uma fonte de verdade por tópico. Se outra rule cobre, aponte.
5. **Sem referência a task/PR/data.** "Adicionado para X" apodrece. Git log/blame é a fonte histórica.
6. **Sem ALL-CAPS gratuito.** "NUNCA"/"SEMPRE" só quando o impacto da violação justifica (perda de dados, vulnerabilidade, regressão crítica). Caso contrário, descreva o motivo e deixe o agente decidir o edge case.

### Frontmatter (se o host usa)

Replique os campos do host. Padrão observável em frameworks atuais:

```yaml
---
description: <uma linha do que a rule cobre + quando carrega>
<campo-de-matcher-do-host>:
  - "<glob-estreito>"
  - "<outro-glob>"
---
```

> **`description` do frontmatter também é matcher.** Em alguns hosts, ela informa o modelo sobre quando aplicar a rule mesmo se o glob casou por proximidade. Escreva-a como gatilho ("Carregada quando trabalhando com X ou Y"), não como abstract acadêmico.

### Template — item simples

```markdown
- **<nome curto>.** <o que fazer>. Exemplo: `path/real.ext:linha`. Por quê: <motivo>.
```

### Template — anti-padrão

```markdown
- **Não <ação proibida>.** Causa <consequência observada>. Em vez disso: <alternativa>. Ver: `path/exemplo.ext`.
```

---

## Quando o usuário pede "documenta isso"

1. **Discovery silencioso** (Fase 0). Confirme convenção do host.
2. Aplique o **teste de fricção**. Se falhar, diga por quê em vez de escrever.
3. Se passa, decida **escopo + matcher** explicitamente. Mostre a decisão.
4. Se há regra equivalente, **proponha edição** em vez de adicionar nova.
5. Mostre o diff (corpo + frontmatter) antes de gravar — o usuário pode discordar do escopo, do matcher ou da forma.

---

## Auditoria de rules existentes

Passes ao revisar rules do host:

| Passe | O que procurar | Ação |
|---|---|---|
| **Apodrecimento** | Datas, "atualmente", "em migração", referências a task/PR | Remover parte volátil ou a regra inteira. |
| **Duplicação** | Mesmo conceito em dois lugares | Manter o mais completo, substituir o outro por link. |
| **Sem exemplo** | Regra abstrata sem arquivo apontado | Achar exemplo real ou remover. |
| **Sem porquê** | Regra crítica sem racional | Adicionar motivo ou rebaixar para "convenção" sem o tom forte. |
| **Óbvio do código** | Coisa que `grep` resolve em 30s | Remover. |
| **Matcher amplo demais** | Glob `**` ou path raiz numa rule de detalhe | Estreitar ou promover a global. |
| **Matcher estreito demais** | Rule não carrega onde se aplica | Adicionar paths faltantes. |
| **Matcher inexistente onde devia existir** | Rule global mas conteúdo é só de um domínio | Mover para rule de escopo ou adicionar matcher. |

Pergunta-âncora: *"Se eu apagar esta linha, alguém faria errado num cenário real?"* Se não, apague.

---

## Sinais de alerta

Pare e revise se você se pegar:

- Escrevendo "ALWAYS" / "NEVER" / "MUST" sem justificar o impacto.
- Copiando 20+ linhas de código numa rule (é reference; mova para `references/` ou linke o arquivo).
- Adicionando regra para caso que ainda não aconteceu ("preventivo"). Espere o segundo/terceiro caso.
- Citando task/PR/data no corpo.
- Não conseguindo apontar arquivo real onde o padrão já existe.
- Inventando diretório de rule sem confirmar convenção do host.
- Usando `**` como matcher (se vale sempre, declare global).

---

## Resumo executivo

> Toda regra precisa: (1) evitar erro que o código não evita, (2) não ser derivável em 1min, (3) não apodrecer em 3 meses, (4) ter o porquê, (5) ter exemplo real, (6) ter **escopo declarado** — global ou com matcher honesto. Antes de criar, faça discovery do host: onde moram as rules, que frontmatter usam, que matchers expressam. Não force convenção. Sem isso → não é regra.
