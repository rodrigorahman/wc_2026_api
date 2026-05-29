Vamos iniciar com a construção da arquitetura base do projeto, o projeto consiste em uma api para atender o projeto de Album da copa do mundo, onde o usuário poderá cadastrar as figurinhas de sua coleção e organizar as figurinhas repetidas para possíveis trocas.

## Requisitos
- Vamos utilizar GRPC como protocolo de comunicação entre o frontend e o backend.
- Vamos utilizar o Sqlite como banco de dados, com o driver modernc.org/sqlite (pure-Go, sem CGO, para compilar nos três sistemas operacionais com CGO_ENABLED=0).
- Vamos utilizar buf.validate (protovalidate) para trabalhar com validações de dados, com as regras declaradas no .proto.
- Para autenticação deve ser usado golang-jwt e como middleware deve ser usado go-grpc-middleware/auth. O token será assinado com HS256, o segredo virá do viper/env e a sessão expira em 1 hora (sem refresh token nesta versão).
- A senha do usuário deve ser guardada com hash bcrypt (cost 12), nunca em texto legível.
- Para configurações de env vamos utilizar o viper.
- Para logging vamos utilizar o zap.
- Para testes vamos utilizar o testify (testes de integração com Sqlite in-memory e testes E2E via bufconn).
- Para trabalhar com queries vamos utilizar o sqlc.
- Para trabalhar com migrations vamos utilizar o golang-migrate. A lista de seleções é fixa e populada por seed via migration.
- Projeto será utilizado compilado rodando no sistema operacional windows, linux e macos.
- Para estrutura de pastas vamos utilizar o conceito de go-standard-layout, organizado por domínio (feature-first): cada domínio é um pacote em internal/<dominio>/ com sub-pastas handler, service e repository, e expõe um fx.Module.
- Para trabalhar com gRPC vamos utilizar o protoc-gen-go e protoc-gen-go-grpc. Os contratos .proto serão versionados por pacote (wc2026.<dominio>.v1) e os stubs gerados ficam em internal/pb. O servidor terá reflection e health-check habilitados.
- Os IDs de todas as tabelas serão UUID v4.
- Para Dependency Injection vamos utilizar uber-fx.
- O banco de dados será escrito 100% em português (tabelas e colunas), enquanto o código Go e os .proto serão 100% em inglês; o mapeamento entre os dois é feito com rename/overrides no sqlc.

## Autenticação
O usuário terá os campos:
- Nome Completo
- E-mail (identificador único de login)
- Senha
- Seleção favorita

## Seleção favorita terá os campos:
- ID
- Nome
