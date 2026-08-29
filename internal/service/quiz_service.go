package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// masteredPenalty reduces the priority score of words already mastered by
// the user, so they still show up occasionally for long-term reinforcement
// without competing with words that actually need review.
const masteredPenalty = 0.2

type QuizService struct {
	words WordRepository
	quiz  QuizRepository
}

func NewQuizService(words WordRepository, quiz QuizRepository) *QuizService {
	return &QuizService{words: words, quiz: quiz}
}

// SelectWordsForQuiz returns up to `count` of the user's words, ordered by
// review priority (highest first): words with a higher error rate and words
// not reviewed in a while are prioritized over well-known, recently
// reviewed ones.
func (s *QuizService) SelectWordsForQuiz(ctx context.Context, userID int64, count int) ([]*Word, error) {
	words, err := s.words.ListByUser(ctx, userID, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list words: %w", err)
	}
	if len(words) == 0 {
		return nil, nil
	}

	now := time.Now()
	sort.SliceStable(words, func(i, j int) bool {
		return priorityScore(words[i], now) > priorityScore(words[j], now)
	})

	if count > 0 && count < len(words) {
		words = words[:count]
	}

	return words, nil
}

// RecordAnswer evaluates the user's answer against the correct one, persists
// it, and updates the word's review stats and difficulty tier accordingly.
// It returns whether the answer was accepted as correct.
func (s *QuizService) RecordAnswer(ctx context.Context, sessionID, wordID int64, questionType, userAnswer, correctAnswer string) (bool, error) {
	correct := EvaluateAnswer(questionType, userAnswer, correctAnswer)

	if err := s.quiz.SaveAnswer(ctx, &QuizAnswer{
		SessionID:     sessionID,
		WordID:        wordID,
		QuestionType:  questionType,
		UserAnswer:    userAnswer,
		CorrectAnswer: correctAnswer,
		IsCorrect:     correct,
	}); err != nil {
		return false, fmt.Errorf("failed to save answer: %w", err)
	}

	if err := s.words.UpdateAfterQuiz(ctx, wordID, correct); err != nil {
		return false, fmt.Errorf("failed to update word after quiz: %w", err)
	}

	return correct, nil
}

// priorityScore implements the algorithm described in the project docs:
//
//	priority_score = (error_weight × 0.6) + (recency_weight × 0.4)
//
//	error_weight   = 1 - (times_correct / times_reviewed)
//	                 1.0 for words never reviewed or with a 100% error rate
//	                 0.0 for words with a 100% success rate
//
//	recency_weight = min(days_since_last_review / 30, 1.0)
//	                 1.0 for words not seen in 30+ days, or never reviewed
//
// Mastered words have their score reduced by masteredPenalty.
func priorityScore(w *Word, now time.Time) float64 {
	errorWeight := 1.0
	if w.TimesReviewed > 0 {
		errorWeight = 1 - float64(w.TimesCorrect)/float64(w.TimesReviewed)
	}

	recencyWeight := 1.0
	if w.LastReviewedAt != nil {
		daysSinceReview := now.Sub(*w.LastReviewedAt).Hours() / 24
		recencyWeight = math.Min(math.Max(daysSinceReview, 0)/30.0, 1.0)
	}

	score := errorWeight*0.6 + recencyWeight*0.4
	if w.Difficulty == "mastered" {
		score *= masteredPenalty
	}

	return score
}
