package whatsmeow

import (
	"testing"

	"lexbot/internal/service"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestParseIncomingMessage(t *testing.T) {
	for _, test := range []struct {
		name    string
		message *waE2E.Message
		kind    service.IncomingMessageKind
		text    string
	}{
		{"text", &waE2E.Message{Conversation: proto.String("resilient")}, service.IncomingMessageText, "resilient"},
		{"extended text", &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{Text: proto.String("/help")}}, service.IncomingMessageText, "/help"},
		{"voice note", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{PTT: proto.Bool(true)}}, service.IncomingMessageAudio, ""},
		{"audio file", &waE2E.Message{AudioMessage: &waE2E.AudioMessage{}}, service.IncomingMessageAudio, ""},
		{"image", &waE2E.Message{ImageMessage: &waE2E.ImageMessage{}}, service.IncomingMessageImage, ""},
		{"image caption is not a command", &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("/quiz")}}, service.IncomingMessageImage, ""},
		{"video", &waE2E.Message{VideoMessage: &waE2E.VideoMessage{}}, service.IncomingMessageUnsupported, ""},
		{"document", &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{}}, service.IncomingMessageUnsupported, ""},
		{"sticker", &waE2E.Message{StickerMessage: &waE2E.StickerMessage{}}, service.IncomingMessageUnsupported, ""},
		{"reaction", &waE2E.Message{ReactionMessage: &waE2E.ReactionMessage{}}, service.IncomingMessageIgnored, ""},
		{"protocol event", &waE2E.Message{ProtocolMessage: &waE2E.ProtocolMessage{}}, service.IncomingMessageIgnored, ""},
		{"empty", &waE2E.Message{}, service.IncomingMessageIgnored, ""},
		{"nil", nil, service.IncomingMessageIgnored, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ParseIncomingMessage(test.message)
			if got.Kind != test.kind || got.Text != test.text {
				t.Fatalf("got %+v, want kind=%v text=%q", got, test.kind, test.text)
			}
		})
	}
}

func TestParseUnwrappedMedia(t *testing.T) {
	message := &waE2E.Message{ImageMessage: &waE2E.ImageMessage{Caption: proto.String("1")}}
	for _, raw := range []*waE2E.Message{
		{EphemeralMessage: &waE2E.FutureProofMessage{Message: message}},
		{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: message}},
	} {
		event := (&events.Message{RawMessage: raw}).UnwrapRaw()
		if got := ParseIncomingMessage(event.Message); got.Kind != service.IncomingMessageImage || got.Text != "" {
			t.Fatalf("wrapped media misclassified: %+v", got)
		}
	}
}
