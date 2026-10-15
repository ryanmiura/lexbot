package whatsmeow

import (
	"lexbot/internal/service"

	"go.mau.fi/whatsmeow/proto/waE2E"
)

// ParseIncomingMessage consumes the unwrapped Message supplied by whatsmeow
// events. Reactions and protocol notifications have no user input to process.
func ParseIncomingMessage(message *waE2E.Message) service.IncomingMessage {
	switch {
	case message.GetAudioMessage() != nil:
		return service.IncomingMessage{Kind: service.IncomingMessageAudio}
	case message.GetImageMessage() != nil:
		return service.IncomingMessage{Kind: service.IncomingMessageImage}
	case message.GetVideoMessage() != nil || message.GetDocumentMessage() != nil || message.GetStickerMessage() != nil:
		return service.IncomingMessage{Kind: service.IncomingMessageUnsupported}
	case message.GetExtendedTextMessage().GetText() != "":
		return service.IncomingMessage{Kind: service.IncomingMessageText, Text: message.GetExtendedTextMessage().GetText()}
	case message.GetConversation() != "":
		return service.IncomingMessage{Kind: service.IncomingMessageText, Text: message.GetConversation()}
	default:
		return service.IncomingMessage{Kind: service.IncomingMessageIgnored}
	}
}
