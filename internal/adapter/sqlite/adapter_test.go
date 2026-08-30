package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"

	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/service"
)

func newTestAdapter(t *testing.T) *sqlite.Adapter {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	adapter, err := sqlite.NewAdapter(dbPath)
	if err != nil {
		t.Fatalf("failed to create test adapter: %v", err)
	}
	return adapter
}

func newTestWord(t *testing.T, adapter *sqlite.Adapter, userID int64, word string) *service.Word {
	t.Helper()
	w := &service.Word{
		UserID: userID, Word: word, Translation: "trad", GrammarClass: "noun",
		ExampleEN: "en", ExamplePT: "pt",
		Quiz: service.Quiz{
			MultipleChoice:   service.QuizQuestion{Question: "q1", Correct: word},
			CompleteSentence: service.QuizQuestion{Question: "q2", Correct: word},
			Reverse:          service.QuizQuestion{Question: "q3", Correct: word},
		},
	}
	if err := adapter.Save(context.Background(), w); err != nil {
		t.Fatalf("failed to save word: %v", err)
	}
	return w
}

func TestFindByID(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	w := newTestWord(t, adapter, user.ID, "turn")

	got, err := adapter.FindByID(ctx, w.ID)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if got == nil || got.Word != "turn" || got.Quiz.MultipleChoice.Correct != "turn" {
		t.Fatalf("unexpected word: %+v", got)
	}

	missing, err := adapter.FindByID(ctx, w.ID+999)
	if err != nil {
		t.Fatalf("FindByID for missing id failed: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing id, got %+v", missing)
	}
}

func TestUpdateAfterQuizDifficultyTransitions(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	t.Run("first review, correct -> familiar (100%% >= 60%% threshold, but not enough reviews to master)", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "one")
		if err := adapter.UpdateAfterQuiz(ctx, w.ID, true); err != nil {
			t.Fatalf("UpdateAfterQuiz failed: %v", err)
		}
		got, err := adapter.FindByUserAndWord(ctx, user.ID, "one")
		if err != nil {
			t.Fatalf("FindByUserAndWord failed: %v", err)
		}
		if got.TimesReviewed != 1 || got.TimesCorrect != 1 || got.Difficulty != "familiar" {
			t.Errorf("got reviewed=%d correct=%d difficulty=%s, want reviewed=1 correct=1 difficulty=familiar",
				got.TimesReviewed, got.TimesCorrect, got.Difficulty)
		}
	})

	t.Run("first review, incorrect -> learning", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "two")
		if err := adapter.UpdateAfterQuiz(ctx, w.ID, false); err != nil {
			t.Fatalf("UpdateAfterQuiz failed: %v", err)
		}
		got, err := adapter.FindByUserAndWord(ctx, user.ID, "two")
		if err != nil {
			t.Fatalf("FindByUserAndWord failed: %v", err)
		}
		if got.TimesReviewed != 1 || got.TimesCorrect != 0 || got.Difficulty != "learning" {
			t.Errorf("got reviewed=%d correct=%d difficulty=%s, want reviewed=1 correct=0 difficulty=learning",
				got.TimesReviewed, got.TimesCorrect, got.Difficulty)
		}
	})

	t.Run("60%% success rate -> familiar", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "three")
		// 3 correct, 2 incorrect => 60%
		outcomes := []bool{true, true, true, false, false}
		for _, correct := range outcomes {
			if err := adapter.UpdateAfterQuiz(ctx, w.ID, correct); err != nil {
				t.Fatalf("UpdateAfterQuiz failed: %v", err)
			}
		}
		got, err := adapter.FindByUserAndWord(ctx, user.ID, "three")
		if err != nil {
			t.Fatalf("FindByUserAndWord failed: %v", err)
		}
		if got.TimesReviewed != 5 || got.TimesCorrect != 3 || got.Difficulty != "familiar" {
			t.Errorf("got reviewed=%d correct=%d difficulty=%s, want reviewed=5 correct=3 difficulty=familiar",
				got.TimesReviewed, got.TimesCorrect, got.Difficulty)
		}
	})

	t.Run("85%% success rate with 5+ reviews -> mastered", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "four")
		outcomes := []bool{true, true, true, true, true}
		for _, correct := range outcomes {
			if err := adapter.UpdateAfterQuiz(ctx, w.ID, correct); err != nil {
				t.Fatalf("UpdateAfterQuiz failed: %v", err)
			}
		}
		got, err := adapter.FindByUserAndWord(ctx, user.ID, "four")
		if err != nil {
			t.Fatalf("FindByUserAndWord failed: %v", err)
		}
		if got.TimesReviewed != 5 || got.TimesCorrect != 5 || got.Difficulty != "mastered" {
			t.Errorf("got reviewed=%d correct=%d difficulty=%s, want reviewed=5 correct=5 difficulty=mastered",
				got.TimesReviewed, got.TimesCorrect, got.Difficulty)
		}
	})

	t.Run("100%% success rate but fewer than 5 reviews -> not mastered yet", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "five")
		outcomes := []bool{true, true, true}
		for _, correct := range outcomes {
			if err := adapter.UpdateAfterQuiz(ctx, w.ID, correct); err != nil {
				t.Fatalf("UpdateAfterQuiz failed: %v", err)
			}
		}
		got, err := adapter.FindByUserAndWord(ctx, user.ID, "five")
		if err != nil {
			t.Fatalf("FindByUserAndWord failed: %v", err)
		}
		if got.Difficulty != "familiar" {
			t.Errorf("got difficulty=%s, want familiar (100%% but only 3 reviews, needs >=5 for mastered)", got.Difficulty)
		}
	})
}

func TestQuizSessionLifecycle(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	w := newTestWord(t, adapter, user.ID, "turn")

	if active, err := adapter.GetActiveSession(ctx, user.ID); err != nil || active != nil {
		t.Fatalf("expected no active session initially, got %+v (err=%v)", active, err)
	}

	session := &service.QuizSession{UserID: user.ID, WordIDs: []int64{w.ID}, TotalQuestions: 1}
	if err := adapter.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
	if session.ID == 0 {
		t.Fatalf("expected session.ID to be set after SaveSession")
	}

	active, err := adapter.GetActiveSession(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil || active.ID != session.ID || len(active.WordIDs) != 1 || active.WordIDs[0] != w.ID {
		t.Fatalf("unexpected active session: %+v", active)
	}

	session.Status = "completed"
	session.CorrectCount = 1
	session.CurrentIndex = 1
	if err := adapter.UpdateSession(ctx, session); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	if active, err := adapter.GetActiveSession(ctx, user.ID); err != nil || active != nil {
		t.Fatalf("expected no active session after completion, got %+v (err=%v)", active, err)
	}
}
