# PRD -- Product Requirements Document (O QUE / POR QUÊ)

## 1. Metadados
- **Nome da Feature/Projeto**: Arquitetura Base — API do Álbum da Copa do Mundo
- **Responsável/Autor**: Rodrigo Rahman
- **Data**: 2026-05-27
- **Versão**: v1
- **Status**: Draft
- **Relacionados**: `docs/specs/features/arquitetura-base/v1/pre-refinement.md` (Discovery / Pré-Refinamento); `docs/specs/features/arquitetura-base/v1/prompt.md` (fonte da ideia)

---

## 2. Contexto & Motivação
- **Qual problema ou dor existe hoje?** Não existe uma fundação sobre a qual o produto "Álbum da Copa do Mundo" possa ser construído. Sem uma base padronizada (estrutura do projeto, configuração, registro de eventos, persistência de dados, contrato de comunicação e autenticação), cada módulo futuro reinventaria a estrutura e divergiria, gerando retrabalho e inconsistência.
- **Como funciona atualmente?** Não funciona — é um ponto de partida do zero. Nenhum fluxo de usuário existe e nenhum módulo de domínio pode começar antes da fundação e do mecanismo de autenticação existirem.
- **Por que isso precisa ser resolvido agora?** É o pré-requisito do projeto inteiro: os módulos seguintes (figurinhas e trocas) dependem desta base e do fluxo de autenticação para começarem.
- **Quem sofre o impacto do problema?** Principalmente o time de desenvolvimento, que perde tempo decidindo padrões ad-hoc a cada módulo; e, indiretamente, o usuário final colecionador, cuja experiência de entrada no produto (cadastro/login) depende desta entrega.

---

## 3. Objetivo da Feature
- **O que se deseja alcançar?** Entregar uma fundação reutilizável e padronizada para o produto, acompanhada de um fluxo de autenticação funcional (cadastro e login) e de um primeiro módulo de referência (Seleção Favorita) que sirva de exemplo para os módulos futuros.
- **Qual mudança de comportamento esta feature deve gerar?** Passar de um repositório vazio para um sistema executável que sobe corretamente nos três sistemas operacionais (Windows, Linux, macOS) e permite que um usuário crie sua conta e se autentique.
- **Qual o resultado final esperado do ponto de vista do usuário?** O usuário final consegue se cadastrar (informando nome completo, e-mail, senha e escolhendo sua seleção favorita) e, em seguida, fazer login para obter acesso autenticado ao sistema. Para o time de desenvolvimento, o resultado é uma base canônica pronta para receber os próximos módulos.

---

## 4. Escopo
### 4.1 O que está incluído (dentro do O QUE)
- [ ] Cadastro de usuário com nome completo, e-mail, senha e seleção favorita obrigatória.
- [ ] Login de usuário (e-mail + senha) que concede acesso autenticado ao sistema.
- [ ] Sessão autenticada com expiração: ao expirar, o usuário precisa fazer login novamente.
- [ ] Armazenamento seguro da senha (nunca em texto legível).
- [ ] Módulo de Seleção Favorita: lista pré-cadastrada de seleções disponíveis para o usuário escolher no cadastro, servindo também de módulo de referência arquitetural.
- [ ] Fundação técnica padronizada que servirá de base para os módulos futuros `[DELEGAR_TECH_SPEC]`.
- [ ] Capacidade de o sistema ser executado nos três sistemas operacionais (Windows, Linux, macOS) `[DELEGAR_TECH_SPEC]`.

### 4.2 O que está explicitamente fora do escopo
- [ ] Módulo de Figurinhas (cadastro da coleção do usuário) — adiado para feature futura.
- [ ] Organização de figurinhas repetidas e sistema de trocas entre usuários — adiado.
- [ ] Gestão administrativa de seleções (a lista vem pré-cadastrada e fixa nesta versão).
- [ ] Recuperação de senha — fora da v1.
- [ ] Verificação/confirmação de e-mail — fora da v1.
- [ ] Renovação automática de sessão (manter o usuário logado sem refazer login) — fora da v1.
- [ ] Edição da seleção favorita após o cadastro — fora da v1.
- [ ] Meta formal de cobertura de testes e automação de entrega contínua — fora da v1.

---

## 5. Usuários & Personas
- **Quem é o usuário principal?** Duas personas:
  - **Persona primária — Time de desenvolvimento backend**: construirá os módulos seguintes (figurinhas, trocas) sobre esta base.
  - **Persona secundária — Usuário final colecionador**: consumidor do fluxo de autenticação entregue aqui (cadastro/login).
- **Qual é seu objetivo ao usar essa feature?** O time quer uma base sólida e padronizada para não reinventar estrutura a cada módulo. O usuário final quer criar sua conta e entrar no produto associando sua seleção favorita.
- **Quais dores/dificuldades essa feature resolve pra ele?** Para o time, elimina decisões ad-hoc e divergência entre módulos. Para o usuário final, viabiliza a porta de entrada (identidade e acesso) ao produto.

### 5.1 Histórias de Usuário (User Stories)
- **US-01**: Como usuário final colecionador, quero criar minha conta informando nome completo, e-mail, senha e minha seleção favorita, para ter acesso ao produto.
- **US-02**: Como usuário final colecionador, quero fazer login com meu e-mail e senha, para acessar o sistema de forma autenticada.
- **US-03**: Como usuário final colecionador, quero escolher minha seleção favorita a partir de uma lista pré-definida durante o cadastro, para personalizar minha identidade no produto.
- **US-04**: Como usuário final colecionador, quero que minha senha seja guardada de forma segura, para que minha conta não fique exposta.
- **US-05**: Como time de desenvolvimento, quero uma fundação técnica padronizada e executável nos três sistemas operacionais, para construir os módulos seguintes sem reinventar estrutura.
- **US-06**: Como time de desenvolvimento, quero um módulo de referência (Seleção Favorita) já implementado, para usá-lo como exemplo de padrão arquitetural nos próximos módulos.

---

## 6. Regras de Negócio (alto nível)
- RN1 — O e-mail é o identificador único do usuário: não pode haver duas contas com o mesmo e-mail.
- RN2 — A senha deve ser armazenada de forma protegida, nunca em texto legível.
- RN3 — O cadastro só é concluído se uma seleção favorita válida (existente na lista pré-cadastrada) for escolhida.
- RN4 — A seleção favorita escolhida deve pertencer à lista de seleções pré-cadastradas; valores fora dessa lista não são aceitos.
- RN5 — O login só é concedido quando e-mail e senha correspondem a uma conta existente.
- RN6 — O acesso autenticado tem validade temporária; após a expiração, é necessário um novo login.
- RN7 — Os dados informados no cadastro e no login devem ser validados antes de qualquer processamento (campos obrigatórios e formato de e-mail).

---

## 7. Fluxo Comportamental (não técnico)

### 7.1 Fluxo Principal — Cadastro
1. O usuário inicia o cadastro informando nome completo, e-mail e senha.
2. O sistema apresenta a lista de seleções disponíveis para escolha.
3. O usuário escolhe sua seleção favorita.
4. O sistema valida os dados (campos obrigatórios, formato de e-mail, e-mail ainda não utilizado, seleção válida).
5. O sistema cria a conta guardando a senha de forma protegida e confirma o cadastro.

### 7.2 Fluxo Principal — Login
1. O usuário informa e-mail e senha.
2. O sistema valida as credenciais contra a conta existente.
3. Em caso de sucesso, o sistema concede acesso autenticado com validade temporária.
4. O usuário passa a acessar áreas que exigem autenticação enquanto a sessão estiver válida.

### 7.3 Fluxos Alternativos
- Se o e-mail já estiver cadastrado, o sistema impede o cadastro e avisa o usuário.
- Se algum campo obrigatório faltar ou o e-mail tiver formato inválido, o sistema avisa e não conclui a operação.
- Se a seleção favorita escolhida não existir na lista, o sistema rejeita o cadastro.
- Se as credenciais de login estiverem incorretas, o sistema nega o acesso sem revelar qual campo falhou.
- Se a sessão expirar, o sistema passa a negar o acesso autenticado e o usuário precisa fazer login novamente.

---

## 8. Critérios de Aceite (O QUE deve acontecer)
- [ ] CA-01: DADO um visitante sem conta QUANDO informa nome completo, e-mail, senha e uma seleção favorita válida ENTÃO o sistema cria a conta e confirma o cadastro.
- [ ] CA-02: DADO um e-mail já cadastrado QUANDO um novo cadastro tenta usar o mesmo e-mail ENTÃO o sistema impede a criação e avisa que o e-mail já está em uso.
- [ ] CA-03: DADO um cadastro QUANDO falta um campo obrigatório ou o e-mail tem formato inválido ENTÃO o sistema rejeita a operação e informa o problema.
- [ ] CA-04: DADO um cadastro QUANDO a seleção favorita escolhida não existe na lista pré-cadastrada ENTÃO o sistema rejeita o cadastro.
- [ ] CA-05: DADO um usuário cadastrado QUANDO informa e-mail e senha corretos ENTÃO o sistema concede acesso autenticado com validade temporária.
- [ ] CA-06: DADO um usuário QUANDO informa e-mail ou senha incorretos ENTÃO o sistema nega o acesso sem indicar qual campo falhou.
- [ ] CA-07: DADO um acesso autenticado QUANDO a sessão expira ENTÃO o sistema passa a negar o acesso protegido até um novo login.
- [ ] CA-08: DADO qualquer conta criada QUANDO a senha é armazenada ENTÃO ela nunca fica em texto legível.
- [ ] CA-09: DADO o sistema entregue QUANDO executado em Windows, Linux ou macOS ENTÃO ele sobe corretamente e o fluxo de autenticação funciona nos três.
- [ ] CA-10: DADO um usuário no cadastro QUANDO consulta as seleções disponíveis ENTÃO o sistema apresenta a lista pré-cadastrada de seleções para escolha.

---

## 9. Restrições & Considerações
- A comunicação entre o cliente (frontend) e este sistema ocorre por um canal de comunicação único e padronizado, que é decisão estruturante do projeto `[DELEGAR_TECH_SPEC]`.
- A stack técnica e o padrão de fundação foram fixados no Discovery (decisões fora de negociação) e devem ser detalhados na Tech Spec, incluindo a escolha do mecanismo de persistência que favorece a portabilidade entre os três sistemas operacionais `[DELEGAR_TECH_SPEC]`.
- Área crítica de segurança: tratamento de senha (proteção no armazenamento), gestão do segredo de assinatura da credencial de acesso e expiração da sessão `[DELEGAR_TECH_SPEC]`.
- O cliente (frontend) é um consumidor separado, fora do escopo deste backend.
- A lista de seleções é fixa nesta versão (sem gestão administrativa).
- Sem prazo/janela de entrega formal definido.
- Considerar, ao modelar a conta do usuário, que módulos futuros (figurinhas/trocas) se relacionarão a ela — manter o modelo coeso e isolado, sem antecipar esses módulos, mas sem inviabilizar a evolução `[DELEGAR_TECH_SPEC]`.

---

## 10. Métricas de Sucesso
- O sistema sobe e executa com sucesso nos três sistemas operacionais (Windows, Linux, macOS).
- O fluxo de autenticação funciona de ponta a ponta: um usuário consegue se cadastrar e, depois, fazer login com sucesso.
- O fluxo de autenticação está coberto por testes que validam os cenários principais (cadastro, login e rejeições esperadas).
- O módulo de Seleção Favorita está disponível e utilizável como referência para os próximos módulos.
- (Qualitativa) Os módulos futuros conseguem seguir a fundação sem precisar redefinir estrutura.

---

## 11. Roadmap / Fases
- **Fase 1 (esta entrega):** Fundação técnica padronizada + autenticação (cadastro/login com senha protegida e sessão temporária) + módulo de Seleção Favorita de referência, executável nos três sistemas operacionais.
- **Fase 2 (futura):** Módulo de Figurinhas — cadastro da coleção do usuário.
- **Fase 3 (futura):** Organização de figurinhas repetidas e sistema de trocas entre usuários.

---

## 12. Rastreabilidade de User Stories

| User Story | Descrição Resumida                              | Critério de Aceite Relacionado |
| ---------- | ----------------------------------------------- | ------------------------------ |
| US-01      | Criar conta com dados + seleção favorita        | CA-01, CA-03                   |
| US-02      | Fazer login com e-mail e senha                  | CA-05, CA-06, CA-07            |
| US-03      | Escolher seleção favorita de lista pré-definida | CA-04, CA-10                   |
| US-04      | Senha armazenada de forma segura                | CA-08                          |
| US-05      | Fundação executável nos três SOs                | CA-09                          |
| US-06      | Módulo de referência (Seleção Favorita)         | CA-10                          |

---

## 13. Checklist Final
- [x] PRD descreve apenas O QUE / POR QUÊ
- [x] Escopo fechado
- [x] User Stories definidas e numeradas (US-XX)
- [x] Critérios de aceite claros
- [x] Tabela de rastreabilidade preenchida
- [x] Pronto para criar o TECH_SPEC (COMO)
