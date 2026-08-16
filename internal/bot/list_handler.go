package bot

import (
	"context"
	"fmt"

	"lexbot/internal/service"
)

// ListHandler handles the "/lista" command.
type ListHandler struct {
	messenger service.Messenger
	wordRepo  service.WordRepository
}

func NewListHandler(messenger service.Messenger, wordRepo service.WordRepository) *ListHandler {
	return &ListHandler{messenger: messenger, wordRepo: wordRepo}
}

// Handle replies with the user's saved words, optionally narrowed by filter
// (e.g. "novas", "dificeis").
func (h *ListHandler) Handle(ctx context.Context, chat string, userID int64, filter string) {
	words, err := h.wordRepo.ListByUser(ctx, userID, filter)
	if err != nil {
		fmt.Printf("Error listing words: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao buscar sua lista. Tente novamente.")
		return
	}

	if err := h.messenger.Send(chat, formatWordList(words)); err != nil {
		fmt.Printf("Error sending word list: %v\n", err)
	}
}
