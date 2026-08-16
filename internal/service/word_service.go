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

	if err := s.wordRepo.Save(ctx, aiWord); err != nil {
		return nil, false, fmt.Errorf("failed to save word: %w", err)
	}

	return aiWord, false, nil
}
