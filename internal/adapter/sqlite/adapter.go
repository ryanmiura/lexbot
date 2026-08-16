package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"lexbot/internal/service"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_schema.sql
var schemaSQL string

// Adapter implements the service.UserRepository and service.WordRepository
// interfaces using SQLite. It shares the same database file used by the
// whatsmeow session store.
type Adapter struct {
	db *sql.DB
}

// NewAdapter opens the SQLite database and applies the initial schema.
func NewAdapter(dbPath string) (*Adapter, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on&_journal_mode=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Adapter{db: db}, nil
}

// Upsert implements service.UserRepository. It creates the user on first
// contact and returns the existing row on subsequent calls.
func (a *Adapter) Upsert(ctx context.Context, phone string) (*service.User, error) {
	if _, err := a.db.ExecContext(ctx,
		`INSERT INTO users (phone) VALUES (?) ON CONFLICT(phone) DO NOTHING`, phone,
	); err != nil {
		return nil, fmt.Errorf("failed to upsert user: %w", err)
	}

	var u service.User
	var hintsEnabled int
	err := a.db.QueryRowContext(ctx,
		`SELECT id, phone, quiz_hints_enabled, created_at FROM users WHERE phone = ?`, phone,
	).Scan(&u.ID, &u.Phone, &hintsEnabled, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to load upserted user: %w", err)
	}
	u.QuizHintsEnabled = hintsEnabled != 0

	return &u, nil
}

// Save implements service.WordRepository. It persists the word and its three
// quiz questions atomically, and sets word.ID on success.
func (a *Adapter) Save(ctx context.Context, word *service.Word) error {
	synonymsJSON, err := json.Marshal(word.Synonyms)
	if err != nil {
		return fmt.Errorf("failed to marshal synonyms: %w", err)
	}
	connectedSpeechJSON, err := json.Marshal(word.ConnectedSpeech)
	if err != nil {
		return fmt.Errorf("failed to marshal connected_speech: %w", err)
	}

	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO words (
			user_id, word, translation, grammar_class, phonetic, definition_en,
			example_en, example_pt, synonyms, quiz_tip, quiz_error_explain, connected_speech
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		word.UserID, word.Word, word.Translation, word.GrammarClass, word.Phonetic, word.DefinitionEN,
		word.ExampleEN, word.ExamplePT, string(synonymsJSON), word.QuizTip, word.QuizErrorExplain, string(connectedSpeechJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to insert word: %w", err)
	}

	wordID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to read inserted word id: %w", err)
	}

	questions := []struct {
		questionType string
		question     service.QuizQuestion
	}{
		{"multiple_choice", word.Quiz.MultipleChoice},
		{"complete_sentence", word.Quiz.CompleteSentence},
		{"reverse", word.Quiz.Reverse},
	}
	for _, q := range questions {
		distractorsJSON, err := json.Marshal(q.question.Distractors)
		if err != nil {
			return fmt.Errorf("failed to marshal distractors: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO quiz_questions (word_id, question_type, question_text, correct_answer, distractors)
			VALUES (?, ?, ?, ?, ?)`,
			wordID, q.questionType, q.question.Question, q.question.Correct, string(distractorsJSON),
		); err != nil {
			return fmt.Errorf("failed to insert quiz question %q: %w", q.questionType, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	word.ID = wordID
	return nil
}

// FindByUserAndWord implements service.WordRepository. It returns nil, nil
// when no matching word exists for the user.
func (a *Adapter) FindByUserAndWord(ctx context.Context, userID int64, word string) (*service.Word, error) {
	var w service.Word
	var synonymsJSON, connectedSpeechJSON string

	err := a.db.QueryRowContext(ctx, `
		SELECT id, user_id, word, translation, grammar_class, phonetic, definition_en,
		       example_en, example_pt, synonyms, quiz_tip, quiz_error_explain, connected_speech,
		       difficulty, times_reviewed, times_correct, last_reviewed_at, created_at
		FROM words WHERE user_id = ? AND word = ?`,
		userID, word,
	).Scan(
		&w.ID, &w.UserID, &w.Word, &w.Translation, &w.GrammarClass, &w.Phonetic, &w.DefinitionEN,
		&w.ExampleEN, &w.ExamplePT, &synonymsJSON, &w.QuizTip, &w.QuizErrorExplain, &connectedSpeechJSON,
		&w.Difficulty, &w.TimesReviewed, &w.TimesCorrect, &w.LastReviewedAt, &w.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find word: %w", err)
	}

	if err := json.Unmarshal([]byte(synonymsJSON), &w.Synonyms); err != nil {
		return nil, fmt.Errorf("failed to unmarshal synonyms: %w", err)
	}
	if err := json.Unmarshal([]byte(connectedSpeechJSON), &w.ConnectedSpeech); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connected_speech: %w", err)
	}

	if err := a.loadQuizQuestions(ctx, &w); err != nil {
		return nil, err
	}

	return &w, nil
}

// loadQuizQuestions fills w.Quiz from the quiz_questions rows belonging to w.
func (a *Adapter) loadQuizQuestions(ctx context.Context, w *service.Word) error {
	rows, err := a.db.QueryContext(ctx,
		`SELECT question_type, question_text, correct_answer, distractors FROM quiz_questions WHERE word_id = ?`,
		w.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to load quiz questions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var questionType, distractorsJSON string
		var q service.QuizQuestion
		if err := rows.Scan(&questionType, &q.Question, &q.Correct, &distractorsJSON); err != nil {
			return fmt.Errorf("failed to scan quiz question: %w", err)
		}
		if distractorsJSON != "" {
			if err := json.Unmarshal([]byte(distractorsJSON), &q.Distractors); err != nil {
				return fmt.Errorf("failed to unmarshal distractors: %w", err)
			}
		}

		switch questionType {
		case "multiple_choice":
			w.Quiz.MultipleChoice = q
		case "complete_sentence":
			w.Quiz.CompleteSentence = q
		case "reverse":
			w.Quiz.Reverse = q
		}
	}

	return rows.Err()
}
