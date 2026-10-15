# LexBot

LexBot é um bot de WhatsApp desenvolvido em Go que utiliza Inteligência Artificial para auxiliar no registro e revisão de vocabulário em inglês. O usuário envia palavras desconhecidas diretamente no chat, e o sistema armazena, contextualiza e gera revisões dinâmicas baseadas nesse histórico.

## Funcionalidades

* **Processamento de Vocabulário:** Ao receber uma palavra ou expressão em inglês, o bot consulta uma API de IA e retorna:
  * Tradução principal.
  * Pronúncia aproximada (aportuguesada).
  * Classe gramatical.
  * Frases de exemplo em inglês e português.
* **Sistema de Quizzes:** Geração de perguntas a partir do vocabulário salvo do próprio usuário. Inclui formatos de múltipla escolha, completar frases e tradução reversa.
* **Revisão Adaptativa:** O bot prioriza palavras no quiz baseando-se na taxa de erro do usuário e no tempo desde a última revisão.
* **Meu vocabulário:** Interface web para gerenciar palavras e expressões, acompanhar o progresso e ajustar as dicas do quiz.

## Comandos

Envie `/ajuda` ou `/help` no WhatsApp para consultar todos os comandos e exemplos. Use
`/ajuda <comando>` para ver os detalhes de uma opção, como `/ajuda lista`.

* `/lista` ou `/list`: lista as palavras salvas; aceita `/lista novas` (`/list new`)
  e `/lista dificeis` (`/list difficult`).
* `/quiz`: inicia ou retoma uma revisão.
* `/status`: mostra seu progresso e sua taxa de acerto.
* `/vocabulario` ou `/vocabulary`: abre **Meu vocabulário** no navegador, com um
  link de uso único válido por 15 minutos. `/dashboard` continua funcionando.

Comandos aceitam maiúsculas e minúsculas, e os filtros aceitam variações com ou
sem acento (por exemplo, `difíceis` e `dificeis`). Filtros desconhecidos recebem
orientação de uso. Os comandos em PT e EN têm o mesmo comportamento, com respostas
em português. `/vocabulário` também é aceito. As URLs existentes em `/dashboard`
permanecem compatíveis.

Para cadastrar vocabulário, envie uma palavra ou expressão em inglês por texto.
Para cadastrar várias de uma vez, use uma por linha, até 10 por mensagem. Durante
um quiz, o texto enviado é tratado como resposta à pergunta atual; os comandos
continuam disponíveis.

Áudios (incluindo mensagens de voz), imagens e outras mídias recebem orientação
para enviar o conteúdo por texto. As mídias e suas legendas não cadastram palavras,
não contam como respostas e mantêm a configuração ou pergunta atual do quiz.
Reações e notificações técnicas são ignoradas.

## Níveis de aprendizado

Os níveis são explicados em `/ajuda status` (ou `/help status`) e na seção
**Como funcionam os níveis?** de **Meu vocabulário**.

| Nível | Critério |
| --- | --- |
| Nova | Nenhuma resposta registrada em um quiz. |
| Aprendendo | Já foi revisada e tem menos de 60% de acertos. |
| Familiar | Pelo menos 60% de acertos, sem atender ao critério de Dominada. |
| Dominada | Pelo menos 5 revisões e 85% de acertos. |

Cada resposta conta como uma revisão. A taxa considera todo o histórico da
palavra, e o nível pode subir ou descer a cada resposta. Um primeiro acerto já
pode tornar uma palavra Familiar.

## Stack Tecnológica

* **Linguagem:** [Go](https://go.dev/) - Utilizado para processamento concorrente e deploy facilitado em binário estático.
* **Mensageria:** [whatsmeow](https://github.com/tulir/whatsmeow) - Comunicação direta com o protocolo do WhatsApp Web.
* **Banco de Dados:** [SQLite](https://sqlite.org/) - Armazenamento embutido e persistência local da sessão do bot e dos dados dos usuários.
* **Inteligência Artificial:** Integração via API (OpenAI / Gemini) responsável apenas pela estruturação de contexto e exemplos gerados a partir da inserção da palavra.
* **Arquitetura:** Estruturado no padrão **Ports and Adapters** (Arquitetura Hexagonal) para isolar a lógica de negócio (Core) das interfaces externas, facilitando a substituição de provedores de IA, banco de dados ou libs de whatsapp e telegram.
