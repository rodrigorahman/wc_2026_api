# INTENT – Próximos jogos das seleções favoritas

## 1. Identificação
- **Nome da Tarefa / Feature**: Próximos jogos das seleções favoritas
- **Autor**: Rodrigo Rahman
- **Data**: 2026-05-29
- **Status**: Draft
- **Relacionados**: `usuario-multiplas-selecoes/v1` (relação usuário↔seleções favoritas, consumida por esta feature), `national-team-flag-url/v1` (bandeira da seleção exibida no card)

---

## 2. Contexto & Motivação
- A home do app não tem como mostrar ao usuário os próximos jogos das seleções que ele escolheu — hoje não existe nenhum dado de partida da Copa 2026 no sistema. O torcedor que favoritou Brasil e Argentina não consegue ver "o próximo jogo do Brasil é dia 12/06 contra o México, no Azteca, Grupo A".
- Precisa ser feito agora porque é insumo direto da home durante a Copa 2026 — a janela de valor é definida pelo calendário do torneio, com começo e fim atrelados a ele.
- Sem isso, o produto perde uma seção natural de engajamento na home e o torcedor não tem onde acompanhar suas seleções dentro do app durante o evento.

---

## 3. Objetivo
- Passar a existir, no sistema, um calendário de partidas da Copa 2026 e a capacidade de devolver, para cada usuário autenticado, os próximos jogos das suas seleções favoritas.
- Ao final, a home do app consegue popular um card com os próximos jogos das favoritas de cada usuário, contendo data/hora, as duas seleções (mandante e visitante), estádio, cidade e fase.

---

## 4. Resultado Esperado
Um usuário autenticado que favoritou seleções, ao consultar o serviço, recebe a lista cronológica (do jogo mais próximo ao mais distante) das partidas futuras dessas seleções. Cada partida traz: data/hora, seleção mandante e seleção visitante (com nome, bandeira e sigla de 3 letras — ex.: BRA, MEX), estádio, cidade e fase (ex.: "Grupo A", "Oitavas de final", "Final").

Comportamentos observáveis:
- Partidas já ocorridas não aparecem — apenas as com data/hora futura em relação ao momento atual.
- Uma partida que envolve duas seleções favoritas do mesmo usuário aparece uma única vez.
- Um usuário cujas seleções não têm jogos futuros (ex.: já eliminadas) recebe lista vazia.

---

## 5. Restrições
- **Sem placar e sem status** da partida (ao vivo/encerrado) — apenas os dados cadastrados. Resultados ao vivo ficam fora desta versão (eventual v2).
- **Cadastro manual, fase a fase**: as partidas são inseridas conforme a Copa avança (grupos, oitavas, quartas, semi, final). Não há serviço de criação/edição de partidas nem papel de administrador no produto.
- **Serviço autenticado que infere as favoritas do próprio usuário** — não recebe a lista de seleções por parâmetro; usa as seleções já escolhidas no cadastro.
- **Sem limite nem paginação** na v1 — retorna todos os jogos futuros das favoritas. Paginação fica adiada para quando o volume justificar.
- **Depende** da relação "seleções favoritas" já persistida pela feature `usuario-multiplas-selecoes/v1`.
- "Próximo jogo" = partida com data/hora futura em relação ao momento atual; jogos passados não retornam.

---

## 6. Checklist Final
- [x] INTENT descreve apenas O QUE / POR QUE
- [x] Objetivo claro e mensurável
- [x] Sem detalhes de implementação ou arquitetura
- [x] Resultado esperado específico
- [x] Restrições explícitas
- [x] Pronto para definição de SCOPE
