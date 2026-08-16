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
}

func NewHandler(messenger service.Messenger, users service.UserRepository, wordService *service.WordService) *Handler {
	return &Handler{
		messenger: messenger,
		users:     users,
		words:     NewWordHandler(messenger, wordService),
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
		h.handleCommand(ctx, chat, user.ID, cmd)
	default:
		h.words.Handle(ctx, chat, user.ID, msg)
	}
}

func (h *Handler) handleCommand(_ context.Context, chat string, _ int64, cmd *Command) {
	switch cmd.Name {
	default:
		h.messenger.Send(chat, "Comando não implementado ainda.")
	}
}
