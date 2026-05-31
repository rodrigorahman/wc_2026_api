# Architecture Decision Records — INDEX

> Ultima atualizacao: 2026-05-31 (6 ADRs)

<!-- ADR-INDEX-START -->
| ID | Titulo | Status | Tags | Problema (1-linha) | Decisao (1-linha) |
|----|--------|--------|------|---------------------|--------------------|
| 0001 | Uso do driver modernc.org/sqlite (pure-Go) para portabilidade cross-platform | accepted | data, build, architecture | A API precisa de SQLite embarcado e roda/compila em múltiplos SOs (macOS, Linux, Windows) e ambie... | Adotar o driver modernc.org/sqlite (implementação pure-Go, sem CGO) como driver SQLite padrão da ... |
| 0002 | Injeção de dependências com uber-fx e go-standard-layout | accepted | architecture, build, cross-cutting | A API cresce em número de módulos (handlers, services, repos) e precisa de wiring consistente, ge... | Adotar uber-fx como container de injeção de dependências e lifecycle, organizando o código segund... |
| 0003 | Autenticação via JWT HS256 com bcrypt e TTL de 1h sem refresh | accepted | auth, security, cross-cutting | A API é stateless e roda como serviço único, sem necessidade de compartilhar | Adotamos JWT assinado com HS256 (segredo compartilhado), senhas com hash bcrypt |
| 0004 | Convenção de idioma — schema em pt-BR e código Go/proto em inglês com bridge via sqlc rename | superseded | data, cross-cutting, architecture | O domínio do projeto (Copa do Mundo 2026) e os stakeholders/DBA operam em pt-BR, então o schema d... | O schema do banco (tabelas e colunas) é nomeado em português brasileiro; o código Go e os contrat... |
| 0005 | Schema do banco em inglês, sem bridge de idioma | accepted | data, cross-cutting, architecture | A ADR-0004 estabeleceu o schema do banco em pt-BR e o código Go/proto em inglês, com a tradução f... | O schema do banco (tabelas e colunas) passa a ser nomeado em inglês, igual ao código Go e aos con... |
| 0006 | Envio de e-mail transacional via provider externo (Resend) com interface no consumidor | accepted | architecture, http, cross-cutting | O projeto não possui nenhuma capacidade de envio de e-mail. A primeira feature a | Adotamos uma interface EmailSender declarada no pacote consumidor, com uma |
<!-- ADR-INDEX-END -->
