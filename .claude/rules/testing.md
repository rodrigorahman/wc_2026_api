---
description: Disciplina de testes da WC 2026 API — boundary real (SQLite efêmero, bufconn), determinismo via clock injetado, asserções sobre comportamento observável (sem mock-driven confidence) e helpers compartilhados. Carregada ao editar qualquer arquivo de teste ou os helpers de teste.
paths:
  - "**/*_test.go"
  - "test/**"
  - "internal/testutil/**"
---

# WC 2026 API — Disciplina de Testes

## Cobertura mínima

- `make test` (`go test ./...`) verde é condição para "pronto".
- Toda função pública: **caminho feliz + ≥1 erro real**.

## Boundary real

- **Integração**: SQLite real efêmero via `internal/testutil.TestNewDB` (migrations + seed + `foreign_keys=ON`). Não mocke o banco em testes de repositório.
- **E2E**: servidor completo em memória via `internal/testutil.TestNewBufconnServer` (`test/e2e/`), com a cadeia de interceptors **real**. Não duplique a montagem da cadeia — reuse `internal/server` / o helper bufconn.

## Determinismo

- Injete `clock.Clock` fixo onde o tempo importa; sem `time.Sleep`, sem relógio real, sem UUID real cujo valor seja asserido.
- Ex.: para expirar token em teste, avance o clock injetado, não durma.

## Qualidade das asserções

- **Sem mock-driven confidence**: asserte o comportamento observável, não o valor que o próprio mock plantou. Ex.: capture o argumento passado a `CreateUser` e verifique o hash com `bcrypt.CompareHashAndPassword`, em vez de confiar no retorno do mock.
- Erros conferidos com `errors.Is` (ou `require.ErrorIs`), nunca por substring.
- Tokens para alg-confusion (`none`/RS256) são forjados **fora** do SUT (base64 manual / RSA efêmera), não via `Generate`.
- Para todo teste com mock, exista companheiro de integração sem mock cobrindo o mesmo caminho real (mock budget).

> Padrões de auth testados aqui referenciam [`auth-security.md`](auth-security.md).
