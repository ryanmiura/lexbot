package bot

import (
	"context"
	"fmt"
	"strings"

	"lexbot/internal/service"
)

// Handler receives raw events from the messaging channel and routes them to
// the right flow (word insertion or command).
type Handler struct {
	messenger service.Messenger
	users     service.UserRepository
	words     *WordHandler
	list      *ListHandler
	quiz      *QuizHandler
}

func NewHandler(messenger service.Messenger, users service.UserRepository, wordService *service.WordService, wordRepo service.WordRepository, quizRepo service.QuizRepository, quizService *service.QuizService) *Handler {
	return &Handler{
		messenger: messenger,
		users:     users,
		words:     NewWordHandler(messenger, wordService),
		list:      NewListHandler(messenger, wordRepo),
		quiz:      NewQuizHandler(messenger, wordRepo, quizRepo, quizService),
	}
}

// HandleMessage processes a single incoming text message, identified by the
// chat to reply to and the sender's phone number.
func (h *Handler) HandleMessage(ctx context.Context, chat string, phone string, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}

	fmt.Printf("Message received from %s: %s\n", phone, msg)

	if strings.EqualFold(msg, "ping") {
		if err := h.messenger.Send(chat, "pong"); err != nil {
			fmt.Printf("Error sending message: %v\n", err)
		}
		return
	}

	user, err := h.users.Upsert(ctx, phone)
	if err != nil {
		fmt.Printf("Error upserting user: %v\n", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao identificar seu usuário. Tente novamente.")
		return
	}

	msgType, cmd := Classify(msg)
	switch msgType {
	case MessageTypeCommand:
		h.handleCommand(ctx, chat, user, cmd)
	default:
		// A quiz in progress claims plain-text input as an answer (or as the
		// round-count reply); only words fall through to word processing.
		if h.quiz.HandleActiveInput(ctx, chat, user, msg) {
			return
		}
		h.words.Handle(ctx, chat, user.ID, msg)
	}
}

func (h *Handler) handleCommand(ctx context.Context, chat string, user *service.User, cmd *Command) {
	switch cmd.Name {
	case "lista":
		h.list.Handle(ctx, chat, user.ID, cmd.Args)
	case "quiz":
		h.quiz.Start(ctx, chat, user)
	default:
		h.messenger.Send(chat, "Comando não implementado ainda.")
	}
}
