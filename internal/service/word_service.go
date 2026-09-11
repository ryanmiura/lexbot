package service

import (
	"context"
	"fmt"
	"strings"
)

type WordService struct {
	aiProvider AIProvider
	wordRepo   WordRepository
}

func NewWordService(aiProvider AIProvider, wordRepo WordRepository) *WordService {
	return &WordService{
		aiProvider: aiProvider,
		wordRepo:   wordRepo,
	}
}

// ProcessNewWord checks whether the user already has the word saved; if so, it
// returns the existing record with alreadyExists=true. Otherwise it processes
// the word via AI and persists it (word + quiz questions) in a transaction.
func (s *WordService) ProcessNewWord(ctx context.Context, userID int64, rawWord string) (word *Word, alreadyExists bool, err error) {
	normalized := strings.ToLower(strings.TrimSpace(rawWord))
	if normalized == "" {
		return nil, false, fmt.Errorf("empty word")
	}

	existing, err := s.wordRepo.FindByUserAndWord(ctx, userID, normalized)
	if err != nil {
		return nil, false, fmt.Errorf("failed to check existing word: %w", err)
	}
	if existing != nil {
		return existing, true, nil
	}

	aiWord, err := s.aiProvider.ProcessWord(ctx, normalized)
	if err != nil {
		return nil, false, err
	}
	aiWord.Word = normalized
	aiWord.UserID = userID

	if err := validateComplete(aiWord); err != nil {
		return nil, false, err
	}

	if err := s.wordRepo.Save(ctx, aiWord); err != nil {
		return nil, false, fmt.Errorf("failed to save word: %w", err)
	}

	return aiWord, false, nil
}

// validateComplete enforces the rule that a word must never be persisted
// with incomplete data. It checks the fields required for the word card and
// the three quiz questions to be usable, regardless of what the AI provider
// returned.
func validateComplete(w *Word) error {
	var missing []string
	require := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}

	require("translation", w.Translation)
	require("grammar_class", w.GrammarClass)
	require("example_en", w.ExampleEN)
	require("example_pt", w.ExamplePT)
	require("connected_speech.tip_pt", w.ConnectedSpeech.TipPT)
	require("quiz.multiple_choice.question", w.Quiz.MultipleChoice.Question)
	require("quiz.multiple_choice.correct", w.Quiz.MultipleChoice.Correct)
	require("quiz.complete_sentence.question", w.Quiz.CompleteSentence.Question)
	require("quiz.complete_sentence.correct", w.Quiz.CompleteSentence.Correct)
	require("quiz.reverse.question", w.Quiz.Reverse.Question)
	require("quiz.reverse.correct", w.Quiz.Reverse.Correct)

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrIncompleteWordData, strings.Join(missing, ", "))
	}
	return nil
}
