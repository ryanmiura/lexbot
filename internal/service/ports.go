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
	// FindByID loads a word (with its quiz questions) by its primary key,
	// used by the quiz flow to render the question for a selected word.
	FindByID(ctx context.Context, id int64) (*Word, error)
	// ListByUser returns the user's words, most recently added first.
	// filter narrows the result: "novas" (difficulty=new), "dificeis" (difficulty=learning),
	// or "" for no filtering.
	ListByUser(ctx context.Context, userID int64, filter string) ([]*Word, error)
	// UpdateAfterQuiz records the outcome of a quiz answer for a word,
	// bumping times_reviewed/times_correct, last_reviewed_at and difficulty.
	UpdateAfterQuiz(ctx context.Context, wordID int64, correct bool) error
}

// QuizRepository abstracts access to quiz session and answer data.
type QuizRepository interface {
	// SaveSession persists a new quiz session and sets session.ID.
	SaveSession(ctx context.Context, session *QuizSession) error
	// GetActiveSession returns the user's active session, or nil if none exists.
	GetActiveSession(ctx context.Context, userID int64) (*QuizSession, error)
	UpdateSession(ctx context.Context, session *QuizSession) error
	SaveAnswer(ctx context.Context, answer *QuizAnswer) error
	// GetQuestionsByWordID returns the three quiz questions generated for a word.
	GetQuestionsByWordID(ctx context.Context, wordID int64) ([]*QuizQuestion, error)
}
