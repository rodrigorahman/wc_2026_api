# CLAUDE.md — WC 2026 API

Guia de trabalho para qualquer agente (ou pessoa) que altere este repositório. Quatro princípios + um resumo de stack. Mantenha curto; o detalhe técnico vive nas [ADRs](docs/adr/INDEX.md) e nas [rules](.claude/rules/).

---

## O projeto em uma frase

API gRPC em Go (uber-fx + SQLite pure-Go + sqlc) com autenticação JWT/bcrypt, compilável cross-platform sem CGO. Veja o [README](README.md) para rodar.

---

## 1. Pense antes de codar

Não assuma. Não esconda confusão. Exponha trade-offs.

- Se a tarefa admite **≥2 interpretações plausíveis**, pare e pergunte — apresente as opções e recomende a mais simples.
- Se implementar como pedido exige **tocar arquivo fora do escopo combinado**, pare e pergunte.
- Termo de domínio ambíguo → cheque os [ADRs](docs/adr/INDEX.md) e a spec antes de inventar.

## 2. Simplicidade primeiro (YAGNI / KISS)

Código mínimo que resolve o problema. Nada especulativo.

- Sem features, parâmetros ou configurabilidade "que podem ser úteis um dia".
- Sem abstração antecipada (interface com uma só implementação, factory de um tipo só, generics para 1 caso).
- Sem try/catch, retry, cache ou validação defensiva para casos que a tarefa não declarou.
- Repetição local pequena > abstração prematura. Na dúvida, o caminho mais simples.
- **Código auto-explicativo, não comentado.** O código deve se explicar por nomes claros e funções pequenas. Comente **apenas** quando for extremamente necessário e realmente agregue: o *porquê* de uma decisão não óbvia, um workaround, uma invariante sutil. Nunca um comentário que só repete o que o código já diz.

## 3. Mudanças cirúrgicas

Toque só no necessário. Limpe só a sua própria bagunça.

- Toda linha alterada rastreia a um requisito. Se não justifica, reverta.
- **Imite o estilo da vizinhança** (naming, ordem de imports, formato de log/retorno) — não imponha preferências.
- Dead code preexistente **não** é seu escopo. Remova apenas símbolos que **suas** mudanças tornaram órfãos.

## 4. Execução orientada a objetivo

Defina o critério de sucesso. Itere até verificar.

- Critério vago vira teste concreto. Escreva o teste, faça passar, refatore só se ainda couber nas regras 2 e 3.
- **Testes não são opcionais.** Toda função pública tem caminho feliz + ≥1 erro real.
- Não reporte "pronto" sem `make test` verde para o que você tocou.

---

## Regras de stack (invioláveis neste projeto)

> O detalhe e o "porquê" estão nas ADRs e nas rules por tema em [`.claude/rules/`](.claude/rules/) — carregadas automaticamente conforme os arquivos que você toca (persistência, idioma, DI/camadas, gRPC, auth/segurança, testes). Aqui o resumo que pesa em toda alteração:

1. **Sem CGO** — driver SQLite é sempre `modernc.org/sqlite`, nunca `sqlite3`/mattn. Build é `CGO_ENABLED=0` (ADR-0001).
2. **Idioma** — schema do banco, código Go e proto todos em **inglês**, sem bridge de tradução (sqlc gera direto do schema; sem `rename`). Dados de domínio (ex.: nomes de seleções) e mensagens ao usuário podem ser pt-BR (ADR-0005, supersede ADR-0004).
3. **DI por uber-fx** — wiring vive em `fx.Module` por domínio; o bind concreto→interface acontece no módulo, não no consumidor (ADR-0002).
4. **Interface no consumidor** — quem usa declara a interface; o pacote concreto não conhece o consumidor.
5. **Auth** — JWT HS256 com `WithValidMethods(["HS256"])` (rejeita alg-confusion), bcrypt cost 12, TTL via config (ADR-0003). Token/senha **nunca** logados.
6. **Erros** — service retorna `status.Error(codes.X)`; handler é mapper puro (não retraduz). Sentinelas comparados com `errors.Is`, nunca por string.
7. **Tempo determinístico** — use a interface `clock.Clock` injetada; nunca `time.Now()` direto em código testável.
8. **Código gerado não se edita** — `internal/pb/**` (buf) e `internal/db/sqlc/**` (sqlc) são regenerados via `make proto` / `make sqlc`.

---

## Comandos

```bash
make test        # go test ./...  (rode antes de reportar conclusão)
make build-all   # cross-compile dos 4 targets (CGO off)
make proto       # regenera stubs proto
make sqlc        # regenera queries tipadas
```

---

**Estas regras funcionam quando** reduzem diff desnecessário, evitam reescrita por excesso de engenharia e trazem a pergunta de esclarecimento **antes** da implementação, não depois.
