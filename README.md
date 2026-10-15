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

## Comandos

Envie `/ajuda` no WhatsApp para consultar todos os comandos e exemplos. Use
`/ajuda <comando>` para ver os detalhes de uma opção, como `/ajuda lista`.

* `/lista`: lista as palavras salvas; aceita `/lista novas` e `/lista dificeis`.
* `/quiz`: inicia ou retoma uma revisão.
* `/status`: mostra seu progresso e sua taxa de acerto.
* `/dashboard`: abre o gerenciamento das palavras no navegador.

Para cadastrar vocabulário, envie uma palavra ou expressão em inglês por texto.
Para cadastrar várias de uma vez, use uma por linha, até 10 por mensagem. Durante
um quiz, o texto enviado é tratado como resposta à pergunta atual; os comandos
continuam disponíveis.

## Stack Tecnológica

* **Linguagem:** [Go](https://go.dev/) - Utilizado para processamento concorrente e deploy facilitado em binário estático.
* **Mensageria:** [whatsmeow](https://github.com/tulir/whatsmeow) - Comunicação direta com o protocolo do WhatsApp Web.
* **Banco de Dados:** [SQLite](https://sqlite.org/) - Armazenamento embutido e persistência local da sessão do bot e dos dados dos usuários.
* **Inteligência Artificial:** Integração via API (OpenAI / Gemini) responsável apenas pela estruturação de contexto e exemplos gerados a partir da inserção da palavra.
* **Arquitetura:** Estruturado no padrão **Ports and Adapters** (Arquitetura Hexagonal) para isolar a lógica de negócio (Core) das interfaces externas, facilitando a substituição de provedores de IA, banco de dados ou libs de whatsapp e telegram.
