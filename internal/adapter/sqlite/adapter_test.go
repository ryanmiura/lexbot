package sqlite_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

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

func TestGetStats(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	t.Run("no words yet", func(t *testing.T) {
		stats, err := adapter.GetStats(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}
		if stats.Total != 0 || stats.TimesReviewed != 0 || stats.TimesCorrect != 0 || stats.LastAddedAt != nil {
			t.Errorf("expected zeroed stats with nil LastAddedAt, got %+v", stats)
		}
	})

	w1 := newTestWord(t, adapter, user.ID, "one")
	w2 := newTestWord(t, adapter, user.ID, "two")
	_ = newTestWord(t, adapter, user.ID, "three")

	if err := adapter.UpdateAfterQuiz(ctx, w1.ID, true); err != nil {
		t.Fatalf("UpdateAfterQuiz failed: %v", err)
	}
	if err := adapter.UpdateAfterQuiz(ctx, w2.ID, false); err != nil {
		t.Fatalf("UpdateAfterQuiz failed: %v", err)
	}

	t.Run("mixed difficulties and review totals", func(t *testing.T) {
		stats, err := adapter.GetStats(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetStats failed: %v", err)
		}
		if stats.Total != 3 {
			t.Errorf("got Total=%d, want 3", stats.Total)
		}
		if stats.New != 1 {
			t.Errorf("got New=%d, want 1 (word 'three' never reviewed)", stats.New)
		}
		if stats.Familiar != 1 || stats.Learning != 1 {
			t.Errorf("got Familiar=%d Learning=%d, want Familiar=1 (w1, 100%%) Learning=1 (w2, 0%%)", stats.Familiar, stats.Learning)
		}
		if stats.TimesReviewed != 2 || stats.TimesCorrect != 1 {
			t.Errorf("got TimesReviewed=%d TimesCorrect=%d, want 2 and 1", stats.TimesReviewed, stats.TimesCorrect)
		}
		if stats.LastAddedAt == nil {
			t.Errorf("expected LastAddedAt to be set once words exist")
		}
	})
}

func TestGetCompletedStats(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	w := newTestWord(t, adapter, user.ID, "turn")

	t.Run("no completed quizzes yet", func(t *testing.T) {
		count, lastCompletedAt, err := adapter.GetCompletedStats(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetCompletedStats failed: %v", err)
		}
		if count != 0 || lastCompletedAt != nil {
			t.Errorf("got count=%d lastCompletedAt=%v, want 0 and nil", count, lastCompletedAt)
		}
	})

	session := &service.QuizSession{UserID: user.ID, WordIDs: []int64{w.ID}, TotalQuestions: 1}
	if err := adapter.SaveSession(ctx, session); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}
	session.Status = "completed"
	now := time.Now().UTC().Truncate(time.Second)
	session.CompletedAt = &now
	if err := adapter.UpdateSession(ctx, session); err != nil {
		t.Fatalf("UpdateSession failed: %v", err)
	}

	t.Run("one completed quiz", func(t *testing.T) {
		count, lastCompletedAt, err := adapter.GetCompletedStats(ctx, user.ID)
		if err != nil {
			t.Fatalf("GetCompletedStats failed: %v", err)
		}
		if count != 1 {
			t.Errorf("got count=%d, want 1", count)
		}
		if lastCompletedAt == nil {
			t.Fatalf("expected lastCompletedAt to be set")
		}
		if !lastCompletedAt.Equal(now) {
			t.Errorf("got lastCompletedAt=%v, want %v", lastCompletedAt, now)
		}
	})
}

func TestSearchByUser(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	other, err := adapter.Upsert(ctx, "5511888888888")
	if err != nil {
		t.Fatalf("failed to upsert other user: %v", err)
	}
	newTestWord(t, adapter, user.ID, "resilient")
	newTestWord(t, adapter, user.ID, "turn")
	newTestWord(t, adapter, other.ID, "resilient") // same word, different user — must not leak

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query returns everything", "", []string{"turn", "resilient"}},
		{"matches word", "resil", []string{"resilient"}},
		{"matches translation", "trad", []string{"turn", "resilient"}}, // newTestWord sets Translation="trad"
		{"no match", "xyz", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := adapter.SearchByUser(ctx, user.ID, tt.query)
			if err != nil {
				t.Fatalf("SearchByUser failed: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d words, want %d (%+v)", len(got), len(tt.want), got)
			}
			for i, w := range got {
				if w.Word != tt.want[i] {
					t.Errorf("word[%d] = %q, want %q", i, w.Word, tt.want[i])
				}
			}
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	other, err := adapter.Upsert(ctx, "5511888888888")
	if err != nil {
		t.Fatalf("failed to upsert other user: %v", err)
	}

	t.Run("deleting someone else's word is a no-op", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "turn")
		deleted, err := adapter.Delete(ctx, other.ID, w.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if deleted {
			t.Errorf("expected deleted=false when the word doesn't belong to the caller")
		}
		got, err := adapter.FindByID(ctx, w.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if got == nil {
			t.Fatalf("expected word to survive a delete attempt by a different user")
		}
	})

	t.Run("deletes a word with quiz history without violating foreign keys", func(t *testing.T) {
		w := newTestWord(t, adapter, user.ID, "resilient")
		// Give it quiz history: a session and a couple of answers, both of
		// which have a NOT NULL foreign key on word_id.
		session := &service.QuizSession{UserID: user.ID, WordIDs: []int64{w.ID}, TotalQuestions: 1}
		if err := adapter.SaveSession(ctx, session); err != nil {
			t.Fatalf("SaveSession failed: %v", err)
		}
		if err := adapter.SaveAnswer(ctx, &service.QuizAnswer{
			SessionID: session.ID, WordID: w.ID, QuestionType: "reverse",
			UserAnswer: "resilient", CorrectAnswer: "resilient", IsCorrect: true,
		}); err != nil {
			t.Fatalf("SaveAnswer failed: %v", err)
		}

		deleted, err := adapter.Delete(ctx, user.ID, w.ID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if !deleted {
			t.Errorf("expected deleted=true when the word belongs to the caller")
		}
		got, err := adapter.FindByID(ctx, w.ID)
		if err != nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if got != nil {
			t.Fatalf("expected word to be gone after Delete, got %+v", got)
		}
	})
}

func TestUpdatePreferences(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}
	if !user.QuizHintsEnabled {
		t.Fatalf("expected quiz hints enabled by default")
	}

	if err := adapter.UpdatePreferences(ctx, user.ID, false); err != nil {
		t.Fatalf("UpdatePreferences failed: %v", err)
	}

	// Upsert on an existing phone just re-reads the row, so it doubles as a
	// way to confirm the preference actually persisted.
	got, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to re-read user: %v", err)
	}
	if got.QuizHintsEnabled {
		t.Errorf("expected quiz hints disabled after UpdatePreferences")
	}
}

func TestDashboardTokenLifecycle(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	t.Run("valid token resolves to the right user, once", func(t *testing.T) {
		token, err := adapter.CreateToken(ctx, user.ID, 15*time.Minute)
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		if token == "" {
			t.Fatalf("expected a non-empty token")
		}

		gotUserID, err := adapter.ConsumeToken(ctx, token)
		if err != nil {
			t.Fatalf("ConsumeToken failed: %v", err)
		}
		if gotUserID != user.ID {
			t.Errorf("got userID=%d, want %d", gotUserID, user.ID)
		}

		if _, err := adapter.ConsumeToken(ctx, token); !errors.Is(err, service.ErrInvalidToken) {
			t.Errorf("expected ErrInvalidToken on replay, got %v", err)
		}
	})

	t.Run("unknown token", func(t *testing.T) {
		if _, err := adapter.ConsumeToken(ctx, "does-not-exist"); !errors.Is(err, service.ErrInvalidToken) {
			t.Errorf("expected ErrInvalidToken, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		token, err := adapter.CreateToken(ctx, user.ID, -1*time.Minute) // already expired
		if err != nil {
			t.Fatalf("CreateToken failed: %v", err)
		}
		if _, err := adapter.ConsumeToken(ctx, token); !errors.Is(err, service.ErrInvalidToken) {
			t.Errorf("expected ErrInvalidToken for an expired token, got %v", err)
		}
	})
}

func TestFindUserByID(t *testing.T) {
	ctx := context.Background()
	adapter := newTestAdapter(t)
	user, err := adapter.Upsert(ctx, "5511999999999")
	if err != nil {
		t.Fatalf("failed to upsert user: %v", err)
	}

	got, err := adapter.FindUserByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindUserByID failed: %v", err)
	}
	if got == nil || got.Phone != "5511999999999" {
		t.Fatalf("unexpected user: %+v", got)
	}

	missing, err := adapter.FindUserByID(ctx, user.ID+999)
	if err != nil {
		t.Fatalf("FindUserByID for missing id failed: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for missing id, got %+v", missing)
	}
}
