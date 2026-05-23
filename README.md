# LexBot

LexBot é um bot de WhatsApp desenvolvido em Go que utiliza Inteligência Artificial para auxiliar no registro e revisão de vocabulário em inglês. O usuário envia palavras desconhecidas diretamente no chat, e o sistema armazena, contextualiza e gera revisões dinâmicas baseadas nesse histórico.

## Funcionalidades

* **Processamento de Vocabulário:** Ao receber uma palavra ou expressão em inglês, o bot consulta uma API de IA e retorna:
  * Tradução principal.
  * Transcrição fonética (IPA).
  * Classe gramatical.
  * Frases de exemplo em inglês e português.
* **Sistema de Quizzes:** Geração de perguntas a partir do vocabulário salvo do próprio usuário. Inclui formatos de múltipla escolha, completar frases e tradução reversa.
* **Revisão Adaptativa:** O bot prioriza palavras no quiz baseando-se na taxa de erro do usuário e no tempo desde a última revisão.

## Stack Tecnológica

* **Linguagem:** [Go](https://go.dev/) - Utilizado para processamento concorrente e deploy facilitado em binário estático.
* **Mensageria:** [whatsmeow](https://github.com/tulir/whatsmeow) - Comunicação direta com o protocolo do WhatsApp Web.
* **Banco de Dados:** [SQLite](https://sqlite.org/) - Armazenamento embutido e persistência local da sessão do bot e dos dados dos usuários.
* **Inteligência Artificial:** Integração via API (OpenAI / Gemini) responsável apenas pela estruturação de contexto e exemplos gerados a partir da inserção da palavra.
* **Arquitetura:** Estruturado no padrão **Ports and Adapters** (Arquitetura Hexagonal) para isolar a lógica de negócio (Core) das interfaces externas, facilitando a substituição de provedores de IA, banco de dados ou libs de whatsapp e telegram.
