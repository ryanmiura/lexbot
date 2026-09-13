package service

import (
	"context"
	"time"
)

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
	// FindUserByID loads a user by primary key, used by the web dashboard
	// (sessions are keyed by user ID, not phone). Returns nil, nil if no
	// user with that ID exists.
	FindUserByID(ctx context.Context, userID int64) (*User, error)
	// UpdatePreferences persists the user's quiz-hint preference, used by
	// the web dashboard.
	UpdatePreferences(ctx context.Context, userID int64, quizHintsEnabled bool) error
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
	// GetStats returns aggregated word counts (by difficulty) and review
	// totals for the user, used by the /status command.
	GetStats(ctx context.Context, userID int64) (*WordStats, error)
	// Delete removes a word (and its quiz questions/answers) if it belongs
	// to userID, used by the web dashboard. Deleting a word that doesn't
	// belong to userID (or doesn't exist) is a no-op, not an error — the
	// scoped WHERE clause is what prevents deleting someone else's word.
	// deleted reports whether a row was actually removed, so callers can
	// tell a genuine deletion apart from a no-op.
	Delete(ctx context.Context, userID int64, wordID int64) (deleted bool, err error)
	// SearchByUser is like ListByUser but narrows results to words whose
	// word or translation contains query (case-insensitive), used by the
	// web dashboard's search box. An empty query behaves like ListByUser
	// with no filter.
	SearchByUser(ctx context.Context, userID int64, query string) ([]*Word, error)
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
	// GetCompletedStats returns how many quiz sessions the user has
	// completed and when the most recent one finished, used by /status.
	// lastCompletedAt is nil if the user has never completed a quiz.
	GetCompletedStats(ctx context.Context, userID int64) (count int, lastCompletedAt *time.Time, err error)
}

// DashboardTokenRepository abstracts magic-link token issuance and
// validation for the web dashboard: "/dashboard" on WhatsApp creates a
// token, and the dashboard's /d/{token} endpoint consumes it.
type DashboardTokenRepository interface {
	// CreateToken generates and persists a new random token for userID,
	// valid for ttl.
	CreateToken(ctx context.Context, userID int64, ttl time.Duration) (token string, err error)
	// ConsumeToken validates token (exists, unexpired, unused), marks it
	// used so it can never be replayed, and returns the associated user
	// ID. Returns ErrInvalidToken if the token is missing, expired, or
	// already used.
	ConsumeToken(ctx context.Context, token string) (userID int64, err error)
}
