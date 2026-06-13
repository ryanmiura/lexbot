package service

import "context"

type WordService struct {
	aiProvider AIProvider
}

func NewWordService(aiProvider AIProvider) *WordService {
	return &WordService{
		aiProvider: aiProvider,
	}
}

// ProcessNewWord takes a raw string, sends it to the AI, and returns the structured data.
// In Phase 4, it does not persist the word to a database yet.
func (s *WordService) ProcessNewWord(ctx context.Context, word string) (*Word, error) {
	aiWord, err := s.aiProvider.ProcessWord(ctx, word)
	if err != nil {
		return nil, err
	}

	return aiWord, nil
}
