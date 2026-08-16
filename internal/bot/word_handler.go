package bot

import (
	"context"
	"fmt"

	"lexbot/internal/service"
)

// WordHandler orchestrates the word-insertion flow: AI processing,
// persistence and the formatted response to the user.
type WordHandler struct {
	messenger   service.Messenger
	wordService *service.WordService
}

func NewWordHandler(messenger service.Messenger, wordService *service.WordService) *WordHandler {
	return &WordHandler{messenger: messenger, wordService: wordService}
}

// Handle processes a single word sent by the user.
func (h *WordHandler) Handle(ctx context.Context, chat string, userID int64, word string) {
	if err := h.messenger.Send(chat, "⏳ Processando..."); err != nil {
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
