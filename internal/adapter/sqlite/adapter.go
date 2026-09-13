package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"lexbot/internal/service"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/001_schema.sql
var schemaSQL string

//go:embed migrations/002_quiz.sql
var quizSchemaSQL string

// Adapter implements the service.UserRepository, service.WordRepository and
// service.QuizRepository interfaces using SQLite. It shares the same
// database file used by the whatsmeow session store.
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
	if _, err := db.Exec(quizSchemaSQL); err != nil {
		return nil, fmt.Errorf("failed to run quiz migrations: %w", err)
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

	if err := a.fillQuizQuestions(ctx, &w); err != nil {
		return nil, err
	}

	return &w, nil
}

// FindByID implements service.WordRepository. It returns nil, nil when no
// word with the given id exists.
func (a *Adapter) FindByID(ctx context.Context, id int64) (*service.Word, error) {
	var w service.Word
	var synonymsJSON, connectedSpeechJSON string

	err := a.db.QueryRowContext(ctx, `
		SELECT id, user_id, word, translation, grammar_class, phonetic, definition_en,
		       example_en, example_pt, synonyms, quiz_tip, quiz_error_explain, connected_speech,
		       difficulty, times_reviewed, times_correct, last_reviewed_at, created_at
		FROM words WHERE id = ?`,
		id,
	).Scan(
		&w.ID, &w.UserID, &w.Word, &w.Translation, &w.GrammarClass, &w.Phonetic, &w.DefinitionEN,
		&w.ExampleEN, &w.ExamplePT, &synonymsJSON, &w.QuizTip, &w.QuizErrorExplain, &connectedSpeechJSON,
		&w.Difficulty, &w.TimesReviewed, &w.TimesCorrect, &w.LastReviewedAt, &w.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find word by id: %w", err)
	}

	if err := json.Unmarshal([]byte(synonymsJSON), &w.Synonyms); err != nil {
		return nil, fmt.Errorf("failed to unmarshal synonyms: %w", err)
	}
	if err := json.Unmarshal([]byte(connectedSpeechJSON), &w.ConnectedSpeech); err != nil {
		return nil, fmt.Errorf("failed to unmarshal connected_speech: %w", err)
	}

	if err := a.fillQuizQuestions(ctx, &w); err != nil {
		return nil, err
	}

	return &w, nil
}

// ListByUser implements service.WordRepository. Quiz questions are not
// loaded here since the list view doesn't need them.
func (a *Adapter) ListByUser(ctx context.Context, userID int64, filter string) ([]*service.Word, error) {
	query := `
		SELECT id, user_id, word, translation, grammar_class, phonetic, definition_en,
		       example_en, example_pt, synonyms, quiz_tip, quiz_error_explain, connected_speech,
		       difficulty, times_reviewed, times_correct, last_reviewed_at, created_at
		FROM words WHERE user_id = ?`

	switch normalizeListFilter(filter) {
	case "novas":
		query += " AND difficulty = 'new'"
	case "dificeis":
		query += " AND difficulty = 'learning'"
	}
	query += " ORDER BY created_at DESC"

	rows, err := a.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list words: %w", err)
	}
	defer rows.Close()

	var words []*service.Word
	for rows.Next() {
		var w service.Word
		var synonymsJSON, connectedSpeechJSON string
		if err := rows.Scan(
			&w.ID, &w.UserID, &w.Word, &w.Translation, &w.GrammarClass, &w.Phonetic, &w.DefinitionEN,
			&w.ExampleEN, &w.ExamplePT, &synonymsJSON, &w.QuizTip, &w.QuizErrorExplain, &connectedSpeechJSON,
			&w.Difficulty, &w.TimesReviewed, &w.TimesCorrect, &w.LastReviewedAt, &w.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		if err := json.Unmarshal([]byte(synonymsJSON), &w.Synonyms); err != nil {
			return nil, fmt.Errorf("failed to unmarshal synonyms: %w", err)
		}
		if err := json.Unmarshal([]byte(connectedSpeechJSON), &w.ConnectedSpeech); err != nil {
			return nil, fmt.Errorf("failed to unmarshal connected_speech: %w", err)
		}
		words = append(words, &w)
	}

	return words, rows.Err()
}

// UpdateAfterQuiz implements service.WordRepository. It bumps the word's
// review counters and recomputes its difficulty tier from the resulting
// success rate, following the thresholds described in the project docs:
// >=85% (with >=5 reviews) is "mastered", >=60% is "familiar", any review
// at all is at least "learning".
func (a *Adapter) UpdateAfterQuiz(ctx context.Context, wordID int64, correct bool) error {
	correctIncrement := 0
	if correct {
		correctIncrement = 1
	}

	_, err := a.db.ExecContext(ctx, `
		UPDATE words SET
			times_reviewed   = times_reviewed + 1,
			times_correct    = times_correct + ?,
			last_reviewed_at = CURRENT_TIMESTAMP,
			difficulty       = CASE
				WHEN (CAST(times_correct + ? AS FLOAT) / (times_reviewed + 1)) >= 0.85
				     AND (times_reviewed + 1) >= 5 THEN 'mastered'
				WHEN (CAST(times_correct + ? AS FLOAT) / (times_reviewed + 1)) >= 0.60
				     THEN 'familiar'
				ELSE 'learning'
			END
		WHERE id = ?`,
		correctIncrement, correctIncrement, correctIncrement, wordID,
	)
	if err != nil {
		return fmt.Errorf("failed to update word after quiz: %w", err)
	}

	return nil
}

// GetStats implements service.WordRepository. It aggregates the user's word
// counts by difficulty and review totals for the /status command.
func (a *Adapter) GetStats(ctx context.Context, userID int64) (*service.WordStats, error) {
	var s service.WordStats
	var lastAddedAt sql.NullString
	err := a.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN difficulty = 'new' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN difficulty = 'learning' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN difficulty = 'familiar' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN difficulty = 'mastered' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(times_reviewed), 0),
			COALESCE(SUM(times_correct), 0),
			MAX(created_at)
		FROM words WHERE user_id = ?`,
		userID,
	).Scan(
		&s.Total, &s.New, &s.Learning, &s.Familiar, &s.Mastered,
		&s.TimesReviewed, &s.TimesCorrect, &lastAddedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get word stats: %w", err)
	}

	s.LastAddedAt, err = parseNullableSQLiteTime(lastAddedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse last added at: %w", err)
	}
	return &s, nil
}

// sqliteTimeLayouts covers the datetime string formats this codebase can
// produce: CURRENT_TIMESTAMP's default ("2006-01-02 15:04:05", used for
// created_at) and the driver's own format for a bound time.Time value
// ("...999999999-07:00", used for completed_at, set from Go).
var sqliteTimeLayouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05",
}

// parseNullableSQLiteTime parses a datetime value returned by an aggregate
// SQL function (e.g. MAX(created_at)). The sqlite3 driver only
// auto-converts bare DATETIME columns to time.Time based on their declared
// column type — an aggregate's result loses that type info and comes back
// as a plain string, so it needs parsing by hand here.
func parseNullableSQLiteTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	for _, layout := range sqliteTimeLayouts {
		if t, err := time.Parse(layout, s.String); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("unrecognized datetime format: %q", s.String)
}

// normalizeListFilter maps the raw "/lista <filter>" argument (with or
// without accents) to a canonical filter key.
func normalizeListFilter(filter string) string {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "novas", "nova", "new":
		return "novas"
	case "dificeis", "difíceis", "dificil", "difícil":
		return "dificeis"
	default:
		return ""
	}
}

// fillQuizQuestions populates w.Quiz from the quiz_questions rows belonging to w.
func (a *Adapter) fillQuizQuestions(ctx context.Context, w *service.Word) error {
	questions, err := a.GetQuestionsByWordID(ctx, w.ID)
	if err != nil {
		return err
	}

	for _, q := range questions {
		switch q.QuestionType {
		case "multiple_choice":
			w.Quiz.MultipleChoice = *q
		case "complete_sentence":
			w.Quiz.CompleteSentence = *q
		case "reverse":
			w.Quiz.Reverse = *q
		}
	}

	return nil
}

// GetQuestionsByWordID implements service.QuizRepository.
func (a *Adapter) GetQuestionsByWordID(ctx context.Context, wordID int64) ([]*service.QuizQuestion, error) {
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, word_id, question_type, question_text, correct_answer, distractors
		 FROM quiz_questions WHERE word_id = ?`,
		wordID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load quiz questions: %w", err)
	}
	defer rows.Close()

	var questions []*service.QuizQuestion
	for rows.Next() {
		var distractorsJSON string
		q := &service.QuizQuestion{}
		if err := rows.Scan(&q.ID, &q.WordID, &q.QuestionType, &q.Question, &q.Correct, &distractorsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan quiz question: %w", err)
		}
		if distractorsJSON != "" {
			if err := json.Unmarshal([]byte(distractorsJSON), &q.Distractors); err != nil {
				return nil, fmt.Errorf("failed to unmarshal distractors: %w", err)
			}
		}
		questions = append(questions, q)
	}

	return questions, rows.Err()
}

// SaveSession implements service.QuizRepository. It persists a new quiz
// session and sets session.ID.
func (a *Adapter) SaveSession(ctx context.Context, session *service.QuizSession) error {
	wordIDsJSON, err := json.Marshal(session.WordIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal word_ids: %w", err)
	}
	if session.Status == "" {
		session.Status = "active"
	}

	res, err := a.db.ExecContext(ctx, `
		INSERT INTO quiz_sessions (user_id, status, word_ids, current_index, correct_count, total_questions)
		VALUES (?, ?, ?, ?, ?, ?)`,
		session.UserID, session.Status, string(wordIDsJSON), session.CurrentIndex, session.CorrectCount, session.TotalQuestions,
	)
	if err != nil {
		return fmt.Errorf("failed to insert quiz session: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to read inserted quiz session id: %w", err)
	}
	session.ID = id

	return nil
}

// GetActiveSession implements service.QuizRepository. It returns nil, nil
// when the user has no active session.
func (a *Adapter) GetActiveSession(ctx context.Context, userID int64) (*service.QuizSession, error) {
	var s service.QuizSession
	var wordIDsJSON string

	err := a.db.QueryRowContext(ctx, `
		SELECT id, user_id, status, word_ids, current_index, correct_count, total_questions, started_at, completed_at
		FROM quiz_sessions WHERE user_id = ? AND status = 'active'
		ORDER BY started_at DESC LIMIT 1`,
		userID,
	).Scan(&s.ID, &s.UserID, &s.Status, &wordIDsJSON, &s.CurrentIndex, &s.CorrectCount, &s.TotalQuestions, &s.StartedAt, &s.CompletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active quiz session: %w", err)
	}

	if err := json.Unmarshal([]byte(wordIDsJSON), &s.WordIDs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal word_ids: %w", err)
	}

	return &s, nil
}

// UpdateSession implements service.QuizRepository.
func (a *Adapter) UpdateSession(ctx context.Context, session *service.QuizSession) error {
	_, err := a.db.ExecContext(ctx, `
		UPDATE quiz_sessions
		SET status = ?, current_index = ?, correct_count = ?, completed_at = ?
		WHERE id = ?`,
		session.Status, session.CurrentIndex, session.CorrectCount, session.CompletedAt, session.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update quiz session: %w", err)
	}
	return nil
}

// SaveAnswer implements service.QuizRepository. It persists an answer and
// sets answer.ID.
func (a *Adapter) SaveAnswer(ctx context.Context, answer *service.QuizAnswer) error {
	res, err := a.db.ExecContext(ctx, `
		INSERT INTO quiz_answers (session_id, word_id, question_type, user_answer, correct_answer, is_correct)
		VALUES (?, ?, ?, ?, ?, ?)`,
		answer.SessionID, answer.WordID, answer.QuestionType, answer.UserAnswer, answer.CorrectAnswer, answer.IsCorrect,
	)
	if err != nil {
		return fmt.Errorf("failed to insert quiz answer: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to read inserted quiz answer id: %w", err)
	}
	answer.ID = id

	return nil
}

// GetCompletedStats implements service.QuizRepository. It reports how many
// quiz sessions the user has completed and when the most recent one
// finished, used by the /status command.
func (a *Adapter) GetCompletedStats(ctx context.Context, userID int64) (count int, lastCompletedAt *time.Time, err error) {
	var lastCompletedAtStr sql.NullString
	err = a.db.QueryRowContext(ctx, `
		SELECT COUNT(*), MAX(completed_at)
		FROM quiz_sessions WHERE user_id = ? AND status = 'completed'`,
		userID,
	).Scan(&count, &lastCompletedAtStr)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get quiz stats: %w", err)
	}

	lastCompletedAt, err = parseNullableSQLiteTime(lastCompletedAtStr)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to parse last completed at: %w", err)
	}
	return count, lastCompletedAt, nil
}
