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

// UserRepository abstracts access to user data, keyed by WhatsApp phone number.
type UserRepository interface {
	Upsert(ctx context.Context, phone string) (*User, error)
}

// WordRepository abstracts access to word data and its associated quiz questions.
type WordRepository interface {
	// Save persists a word and its quiz questions atomically, and sets word.ID.
	Save(ctx context.Context, word *Word) error
	FindByUserAndWord(ctx context.Context, userID int64, word string) (*Word, error)
}
