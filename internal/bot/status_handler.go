package bot

import (
	"context"
	"log/slog"

	"lexbot/internal/service"
)

// StatusHandler handles the "/status" command.
type StatusHandler struct {
	messenger service.Messenger
	wordRepo  service.WordRepository
	quizRepo  service.QuizRepository
}

func NewStatusHandler(messenger service.Messenger, wordRepo service.WordRepository, quizRepo service.QuizRepository) *StatusHandler {
	return &StatusHandler{messenger: messenger, wordRepo: wordRepo, quizRepo: quizRepo}
}

// Handle replies with a snapshot of the user's progress: word counts by
// difficulty, quizzes completed and overall accuracy. Everything is derived
// directly from existing data — no AI call involved.
func (h *StatusHandler) Handle(ctx context.Context, chat string, userID int64) {
	wordStats, err := h.wordRepo.GetStats(ctx, userID)
	if err != nil {
		slog.Error("failed to get word stats", "user_id", userID, "error", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao buscar seu status. Tente novamente.")
		return
	}

	quizzesCompleted, lastQuizAt, err := h.quizRepo.GetCompletedStats(ctx, userID)
	if err != nil {
		slog.Error("failed to get quiz stats", "user_id", userID, "error", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao buscar seu status. Tente novamente.")
		return
	}

	if err := h.messenger.Send(chat, formatStatus(wordStats, quizzesCompleted, lastQuizAt)); err != nil {
		slog.Error("failed to send status", "user_id", userID, "error", err)
	}
}
