package bot

import (
	"context"
	"fmt"
	"strings"

	"lexbot/internal/service"
)

// maxWordsPerMessage caps how many words a single message can register in
// one go (words are separated by line breaks).
const maxWordsPerMessage = 10

// WordHandler orchestrates the word-insertion flow: AI processing,
// persistence and the formatted response to the user.
type WordHandler struct {
	messenger   service.Messenger
	wordService *service.WordService
}

func NewWordHandler(messenger service.Messenger, wordService *service.WordService) *WordHandler {
	return &WordHandler{messenger: messenger, wordService: wordService}
}

// Handle splits the message into one word per line and processes each one
// sequentially, capping the batch at maxWordsPerMessage.
func (h *WordHandler) Handle(ctx context.Context, chat string, userID int64, msg string) {
	words := splitWords(msg)
	if len(words) == 0 {
		return
	}

	if len(words) > maxWordsPerMessage {
		h.messenger.Send(chat, fmt.Sprintf(
			"⚠️ Você enviou %d palavras, mas o limite é %d por mensagem. Processando as %d primeiras — envie o restante em outra mensagem.",
			len(words), maxWordsPerMessage, maxWordsPerMessage,
		))
		words = words[:maxWordsPerMessage]
	}

	for _, word := range words {
		h.handleOne(ctx, chat, userID, word)
	}
}

// handleOne processes a single word sent by the user.
func (h *WordHandler) handleOne(ctx context.Context, chat string, userID int64, word string) {
	if err := h.messenger.Send(chat, fmt.Sprintf("⏳ Processando \"%s\"...", word)); err != nil {
		fmt.Printf("Error sending feedback message: %v\n", err)
	}

	aiWord, alreadyExists, err := h.wordService.ProcessNewWord(ctx, userID, word)
	if err != nil {
		fmt.Printf("Error processing word %q: %v\n", word, err)
		h.messenger.Send(chat, fmt.Sprintf("❌ Ocorreu um erro ao processar \"%s\". Tente novamente.", word))
		return
	}

	card := formatWordCard(aiWord)
	if alreadyExists {
		card = "⚠️ Você já tem essa palavra na sua lista!\n\n" + card
	}
	if err := h.messenger.Send(chat, card); err != nil {
		fmt.Printf("Error sending card: %v\n", err)
	}
}

// splitWords breaks a message into non-empty, trimmed lines.
func splitWords(msg string) []string {
	lines := strings.Split(msg, "\n")
	words := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			words = append(words, line)
		}
	}
	return words
}
