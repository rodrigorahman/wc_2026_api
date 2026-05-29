Vamos iniciar com a construção da arquitetura base do projeto, o projeto consiste em uma api para atender o projeto de Album da copa do mundo, onde o usuário poderá cadastrar as figurinhas de sua coleção e organizar as figurinhas repetidas para possíveis trocas.

## Requisitos
- Vamos utilizar GRPC como protocolo de comunicação entre o frontend e o backend.
- Vamos utilizar o Sqlite como banco de dados.
- Vamos utilizar uf.validate para trabalhar com validações de dados.
- Para autenticação deve ser usado golang-jwt e como middleware deve ser usado go-grpc-middleware/auth
- Para configurações de env vamos utilizar o viper
- Para logging vamos utilizar o zap
- Para testes vamos utilizar o testify
- Para trabalhar com queries vamos utilizar o sqlc
- Para trabalhar com migrations vamos utilizar o golang-migrate
- Projeto será utilizado compilado rodando no sistema operacional windows,linux e macos.
- Para estrutura de pastas vamos utilizar o conceito de go-standard-layout
- Para trabalhar com gRPC vamos utilizar o protoc-gen-go e protoc-gen-go-grpc
- Os IDs de todas as tabelas serão UUID v4
- Para Dependency Injection vamos utilizar uber-fx

## Autenticação
O usuário terá os campos:
- Nome Completo
- E-mail
- Seleção favorita

## Seleção favorita terá os campos:
- ID
- Nome
