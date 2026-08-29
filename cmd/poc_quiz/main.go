package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/service"

	_ "github.com/mattn/go-sqlite3"
)

// seedWord inserts a word via the repository and then back-fills its review
// stats with raw SQL, so we can simulate words at different stages of the
// review cycle without having to actually run a quiz first.
func seedWord(ctx context.Context, repo *sqlite.Adapter, raw *sql.DB, userID int64, word string, timesReviewed, timesCorrect int, daysSinceReview int, difficulty string) *service.Word {
	w := &service.Word{
		UserID: userID, Word: word, Translation: "tradução-" + word, GrammarClass: "noun",
		ExampleEN: "example", ExamplePT: "exemplo",
		Quiz: service.Quiz{
			MultipleChoice:   service.QuizQuestion{Question: "q1", Correct: word},
			CompleteSentence: service.QuizQuestion{Question: "q2", Correct: word},
			Reverse:          service.QuizQuestion{Question: "q3", Correct: word},
		},
	}
	if err := repo.Save(ctx, w); err != nil {
		panic(err)
	}

	var lastReviewedAt any
	if daysSinceReview >= 0 {
		lastReviewedAt = time.Now().Add(-time.Duration(daysSinceReview) * 24 * time.Hour)
	}
	if _, err := raw.ExecContext(ctx,
		`UPDATE words SET times_reviewed = ?, times_correct = ?, last_reviewed_at = ?, difficulty = ? WHERE id = ?`,
		timesReviewed, timesCorrect, lastReviewedAt, difficulty, w.ID,
	); err != nil {
		panic(err)
	}

	return w
}

func main() {
	ctx := context.Background()
	dbPath := filepath.Join(os.TempDir(), fmt.Sprintf("lexbot_poc_quiz_%d.db", time.Now().UnixNano()))
	defer os.Remove(dbPath)
	fmt.Println("Using scratch database:", dbPath)

	repo, err := sqlite.NewAdapter(dbPath)
	if err != nil {
		panic(err)
	}
	raw, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(err)
	}
	defer raw.Close()

	user, err := repo.Upsert(ctx, "5511999999999")
	if err != nil {
		panic(err)
	}

	seedWord(ctx, repo, raw, user.ID, "never_reviewed", 0, 0, -1, "new")
	seedWord(ctx, repo, raw, user.ID, "reviewed_long_ago_mixed", 4, 2, 40, "learning")
	seedWord(ctx, repo, raw, user.ID, "reviewed_recently_good", 5, 5, 2, "familiar")
	seedWord(ctx, repo, raw, user.ID, "mastered_recent", 10, 9, 0, "mastered")

	quizService := service.NewQuizService(repo, repo)

	selected, err := quizService.SelectWordsForQuiz(ctx, user.ID, 10)
	if err != nil {
		panic(err)
	}

	fmt.Println("\nSelected words, in priority order (highest priority first):")
	for i, w := range selected {
		lastReviewed := "never"
		if w.LastReviewedAt != nil {
			lastReviewed = fmt.Sprintf("%.0f days ago", time.Since(*w.LastReviewedAt).Hours()/24)
		}
		fmt.Printf("%d. %-28s difficulty=%-10s reviewed=%d correct=%d last_reviewed=%s\n",
			i+1, w.Word, w.Difficulty, w.TimesReviewed, w.TimesCorrect, lastReviewed)
	}
}
