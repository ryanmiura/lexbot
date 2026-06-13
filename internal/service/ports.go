package service

import "context"

// Messenger abstracts sending messages to the end user,
// allowing integration with WhatsApp, Telegram, etc.
type Messenger interface {
	Send(to string, message string) error
	// SendDocument(to string, filename string, data []byte) error // future
}

// AIProvider abstracts the intelligence service (e.g. OpenAI, Groq, Gemini)
// that receives a raw word and returns structured contextual information.
type AIProvider interface {
	ProcessWord(ctx context.Context, word string) (*Word, error)
}
