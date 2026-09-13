package bot

import (
	"context"
	"log/slog"

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
		slog.Error("failed to list words", "user_id", userID, "filter", filter, "error", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao buscar sua lista. Tente novamente.")
		return
	}

	slog.Info("word list requested", "user_id", userID, "filter", filter, "count", len(words))

	if err := h.messenger.Send(chat, formatWordList(words)); err != nil {
		slog.Error("failed to send word list", "user_id", userID, "error", err)
	}
}
