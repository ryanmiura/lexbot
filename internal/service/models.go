package service

import "time"

// User represents a bot user, identified by their WhatsApp phone number.
type User struct {
	ID               int64
	Phone            string
	QuizHintsEnabled bool
	CreatedAt        time.Time
}

type Word struct {
	// Persistence fields, populated by WordRepository, never set from the AI response.
	ID             int64      `json:"-"`
	UserID         int64      `json:"-"`
	Difficulty     string     `json:"-"`
	TimesReviewed  int        `json:"-"`
	TimesCorrect   int        `json:"-"`
	LastReviewedAt *time.Time `json:"-"`
	CreatedAt      time.Time  `json:"-"`

	Word             string          `json:"word"`
	Translation      string          `json:"translation"`
	GrammarClass     string          `json:"grammar_class"`
	Phonetic         string          `json:"phonetic"`
	DefinitionEN     string          `json:"definition_en"`
	ExampleEN        string          `json:"example_en"`
	ExamplePT        string          `json:"example_pt"`
	Synonyms         []string        `json:"synonyms"`
	QuizTip          string          `json:"quiz_tip"`
	QuizErrorExplain string          `json:"quiz_error_explain"`
	ConnectedSpeech  ConnectedSpeech `json:"connected_speech"`
	Quiz             Quiz            `json:"quiz"`
}

type ConnectedSpeech struct {
	ExamplePhrase    string `json:"example_phrase"`
	ExamplePhrasePT  string `json:"example_phrase_pt"`
	LinkingWords     string `json:"linking_words"`
	ConnectedSound   string `json:"connected_sound"`
	ConnectedSoundPT string `json:"connected_sound_pt"`
	TipPT            string `json:"tip_pt"`
}

type Quiz struct {
	MultipleChoice   QuizQuestion `json:"multiple_choice"`
	CompleteSentence QuizQuestion `json:"complete_sentence"`
	Reverse          QuizQuestion `json:"reverse"`
}

type QuizQuestion struct {
	// Persistence fields, populated by WordRepository/QuizRepository, never set from the AI response.
	ID           int64  `json:"-"`
	WordID       int64  `json:"-"`
	QuestionType string `json:"-"` // multiple_choice | complete_sentence | reverse

	Question    string   `json:"question"`
	Correct     string   `json:"correct"`
	Distractors []string `json:"distractors,omitempty"`
}

// QuizSession tracks an in-progress or finished quiz run for a user.
type QuizSession struct {
	ID             int64
	UserID         int64
	Status         string // active | completed | abandoned
	WordIDs        []int64
	CurrentIndex   int
	CorrectCount   int
	TotalQuestions int
	StartedAt      time.Time
	CompletedAt    *time.Time
}

// QuizAnswer records a single answer given during a quiz session.
type QuizAnswer struct {
	ID            int64
	SessionID     int64
	WordID        int64
	QuestionType  string
	UserAnswer    string
	CorrectAnswer string
	IsCorrect     bool
	AnsweredAt    time.Time
}
