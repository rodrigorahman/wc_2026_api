# Glossário de Domínio — Projeto

> Fonte canônica do vocabulário do produto **Álbum da Copa do Mundo** (cross-feature). Termos de entidade de negócio vivem aqui; termos operacionais restritos a uma feature ficam no glossário-feature correspondente.
>
> **Convenção de idioma do projeto**: identificadores de **dado/banco** ficam em **pt-BR**; identificadores de **código Go/proto** ficam em **inglês**. A ponte é o `sqlc` (`rename`/`overrides`). Cada termo abaixo registra o par pt-BR (DB) ↔ inglês (código).

## Termos

**User** (`usuarios` no DB):
Pessoa que possui uma conta no produto, identificada unicamente pelo e-mail.
_pt-BR (DB)_: `usuarios` · `nome_completo` · `email` · `senha_hash` · `selecao_id` · `criado_em`
_inglês (código)_: `User` · `FullName` · `Email` · `PasswordHash` · `NationalTeamID` · `CreatedAt`
_Evitar_: Usuario (sem acento como termo de domínio), Account, Member, Customer.

**NationalTeam** (`selecoes` no DB):
Seleção nacional de futebol que disputa a Copa do Mundo; pertence a uma lista fixa pré-cadastrada por seed.
_pt-BR (DB)_: `selecoes` · `nome`
_inglês (código)_: `NationalTeam` · `Name`
_Evitar_: Selection (tradução literal incorreta — "seleção" de futebol é "national team", não "selection"), Team, Country.

**Seleção Favorita** (não é entidade — é relacionamento):
A `NationalTeam` escolhida por um `User` no cadastro. Materializa-se como o atributo `usuarios.selecao_id` (FK → `selecoes.id`), **não** como uma tabela/entidade separada.
_Evitar_: tratar "Seleção Favorita" como entidade própria; criar tabela `selecoes_favoritas`.

**Sessão** (representada por JWT):
Acesso autenticado de um `User`, com validade temporária (TTL 1h, sem refresh na v1). Representada por um token JWT HS256 (claim `sub` = `user_id`, `exp` = expiração), validado pelo interceptor de auth nos RPCs protegidos.
_inglês (código)_: `TokenManager`, `access_token`, claims `sub`/`exp`.
_Evitar_: Token (use "Sessão" para o conceito de acesso; "token"/JWT é a representação), Login (Login é a ação que cria a Sessão).

## Relacionamentos

- Um **User** escolhe exatamente uma **NationalTeam** como Seleção Favorita (FK `selecao_id`, obrigatória).
- Uma **NationalTeam** pode ser a favorita de zero ou muitos **Users** (1:N).
- Um **Login** bem-sucedido de um **User** produz uma **Sessão** (JWT) temporária.

## Ambiguidades resolvidas

- **"Seleção Favorita"** era passível de ser lida como entidade separada — resolvido: é a **NationalTeam escolhida por um User** (relacionamento via `selecao_id`), não uma entidade nova.
- **"Seleção" → "Selection"** (tradução literal) — resolvido: o termo de futebol "seleção" canoniza para **`NationalTeam`** em inglês; "selection" é proibido para esta entidade.
