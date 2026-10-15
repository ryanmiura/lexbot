package bot

import (
	"context"
	"log/slog"

	"lexbot/internal/service"
)

// HandleIncomingMessage rejects unsupported media before text routing, so a
// media message can never consume a quiz answer or change its configuration.
func (h *Handler) HandleIncomingMessage(ctx context.Context, chat, phone string, message service.IncomingMessage) {
	var reply string
	switch message.Kind {
	case service.IncomingMessageText:
		h.HandleMessage(ctx, chat, phone, message.Text)
		return
	case service.IncomingMessageAudio:
		reply = "🎤 Recebi seu áudio! Por enquanto, envie a palavra ou sua resposta por texto. Digite /ajuda para ver as opções."
	case service.IncomingMessageImage:
		reply = "🖼️ Recebi sua imagem! Por enquanto, digite a palavra ou expressão que você quer salvar, ou sua resposta ao quiz. Digite /ajuda para ver as opções."
	case service.IncomingMessageUnsupported:
		reply = "Recebi uma mídia que ainda não consigo processar. Envie a palavra ou sua resposta por texto. Digite /ajuda para ver as opções."
	default:
		return
	}
	if err := h.messenger.Send(chat, reply); err != nil {
		slog.Error("failed to send media guidance", "kind", message.Kind, "error", err)
	}
}
