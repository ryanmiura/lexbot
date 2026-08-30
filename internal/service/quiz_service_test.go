package service

import (
	"context"
	"testing"
	"time"
)

type fakeWordRepository struct {
	words           []*Word
	updateAfterQuiz []struct {
		wordID  int64
		correct bool
	}
}

func (f *fakeWordRepository) Save(ctx context.Context, word *Word) error { return nil }
func (f *fakeWordRepository) FindByUserAndWord(ctx context.Context, userID int64, word string) (*Word, error) {
	return nil, nil
}
func (f *fakeWordRepository) FindByID(ctx context.Context, id int64) (*Word, error) {
	for _, w := range f.words {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, nil
}
func (f *fakeWordRepository) ListByUser(ctx context.Context, userID int64, filter string) ([]*Word, error) {
	return f.words, nil
}
func (f *fakeWordRepository) UpdateAfterQuiz(ctx context.Context, wordID int64, correct bool) error {
	f.updateAfterQuiz = append(f.updateAfterQuiz, struct {
		wordID  int64
		correct bool
	}{wordID, correct})
	return nil
}

type fakeQuizRepository struct {
	savedAnswers []*QuizAnswer
}

func (f *fakeQuizRepository) SaveSession(ctx context.Context, session *QuizSession) error { return nil }
func (f *fakeQuizRepository) GetActiveSession(ctx context.Context, userID int64) (*QuizSession, error) {
	return nil, nil
}
func (f *fakeQuizRepository) UpdateSession(ctx context.Context, session *QuizSession) error {
	return nil
}
func (f *fakeQuizRepository) SaveAnswer(ctx context.Context, answer *QuizAnswer) error {
	f.savedAnswers = append(f.savedAnswers, answer)
	return nil
}
func (f *fakeQuizRepository) GetQuestionsByWordID(ctx context.Context, wordID int64) ([]*QuizQuestion, error) {
	return nil, nil
}

func daysAgo(d int) *time.Time {
	t := time.Now().Add(-time.Duration(d) * 24 * time.Hour)
	return &t
}

func TestSelectWordsForQuizPriorityOrder(t *testing.T) {
	words := []*Word{
		{ID: 1, Word: "reviewed_recently_good", Difficulty: "familiar", TimesReviewed: 5, TimesCorrect: 5, LastReviewedAt: daysAgo(2)},
		{ID: 2, Word: "mastered_recent", Difficulty: "mastered", TimesReviewed: 10, TimesCorrect: 9, LastReviewedAt: daysAgo(0)},
		{ID: 3, Word: "never_reviewed", Difficulty: "new", TimesReviewed: 0, TimesCorrect: 0, LastReviewedAt: nil},
		{ID: 4, Word: "reviewed_long_ago_mixed", Difficulty: "learning", TimesReviewed: 4, TimesCorrect: 2, LastReviewedAt: daysAgo(40)},
	}
	wordRepo := &fakeWordRepository{words: words}
	quizService := NewQuizService(wordRepo, &fakeQuizRepository{})

	selected, err := quizService.SelectWordsForQuiz(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("SelectWordsForQuiz returned error: %v", err)
	}

	wantOrder := []string{"never_reviewed", "reviewed_long_ago_mixed", "reviewed_recently_good", "mastered_recent"}
	if len(selected) != len(wantOrder) {
		t.Fatalf("got %d words, want %d", len(selected), len(wantOrder))
	}
	for i, w := range selected {
		if w.Word != wantOrder[i] {
			t.Errorf("position %d: got %q, want %q (full order: %v)", i, w.Word, wantOrder[i], wordNames(selected))
		}
	}
}

func TestSelectWordsForQuizRespectsCount(t *testing.T) {
	words := []*Word{
		{ID: 1, Word: "a", TimesReviewed: 0},
		{ID: 2, Word: "b", TimesReviewed: 0},
		{ID: 3, Word: "c", TimesReviewed: 0},
	}
	wordRepo := &fakeWordRepository{words: words}
	quizService := NewQuizService(wordRepo, &fakeQuizRepository{})

	selected, err := quizService.SelectWordsForQuiz(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("SelectWordsForQuiz returned error: %v", err)
	}
	if len(selected) != 2 {
		t.Fatalf("got %d words, want 2", len(selected))
	}
}

func TestSelectWordsForQuizEmptyList(t *testing.T) {
	wordRepo := &fakeWordRepository{words: nil}
	quizService := NewQuizService(wordRepo, &fakeQuizRepository{})

	selected, err := quizService.SelectWordsForQuiz(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("SelectWordsForQuiz returned error: %v", err)
	}
	if len(selected) != 0 {
		t.Fatalf("got %d words, want 0", len(selected))
	}
}

func TestRecordAnswerCorrect(t *testing.T) {
	wordRepo := &fakeWordRepository{}
	quizRepo := &fakeQuizRepository{}
	quizService := NewQuizService(wordRepo, quizRepo)

	correct, err := quizService.RecordAnswer(context.Background(), 42, 7, "complete_sentence", "resilient", "resilient")
	if err != nil {
		t.Fatalf("RecordAnswer returned error: %v", err)
	}
	if !correct {
		t.Errorf("expected correct=true")
	}

	if len(quizRepo.savedAnswers) != 1 {
		t.Fatalf("expected 1 saved answer, got %d", len(quizRepo.savedAnswers))
	}
	saved := quizRepo.savedAnswers[0]
	if saved.SessionID != 42 || saved.WordID != 7 || !saved.IsCorrect {
		t.Errorf("unexpected saved answer: %+v", saved)
	}

	if len(wordRepo.updateAfterQuiz) != 1 || wordRepo.updateAfterQuiz[0].wordID != 7 || !wordRepo.updateAfterQuiz[0].correct {
		t.Errorf("unexpected UpdateAfterQuiz calls: %+v", wordRepo.updateAfterQuiz)
	}
}

func TestRecordAnswerIncorrect(t *testing.T) {
	wordRepo := &fakeWordRepository{}
	quizRepo := &fakeQuizRepository{}
	quizService := NewQuizService(wordRepo, quizRepo)

	correct, err := quizService.RecordAnswer(context.Background(), 1, 2, "reverse", "wrong", "resilient")
	if err != nil {
		t.Fatalf("RecordAnswer returned error: %v", err)
	}
	if correct {
		t.Errorf("expected correct=false")
	}
	if len(wordRepo.updateAfterQuiz) != 1 || wordRepo.updateAfterQuiz[0].correct {
		t.Errorf("expected UpdateAfterQuiz called with correct=false, got %+v", wordRepo.updateAfterQuiz)
	}
}

func wordNames(words []*Word) []string {
	names := make([]string, len(words))
	for i, w := range words {
		names[i] = w.Word
	}
	return names
}
