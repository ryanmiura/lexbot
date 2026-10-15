package service

import (
	"fmt"
	"slices"
)

// Thresholds are shared by persistence and the user-facing level explanations.
const (
	FamiliarMinimumAccuracy = 0.60
	MasteredMinimumAccuracy = 0.85
	MasteredMinimumReviews  = 5
)

type WordLevel struct {
	Code        string
	Label       string
	Description string
}

const WordLevelNote = "Cada resposta no quiz conta como uma revisão da palavra. A taxa de acerto considera todo o histórico dessa palavra. O nível é recalculado a cada resposta e pode subir ou descer. Um primeiro acerto já pode deixar a palavra como Familiar."

var wordLevels = []WordLevel{
	{Code: "new", Label: "Nova", Description: "Ainda não foi respondida em um quiz."},
	{Code: "learning", Label: "Aprendendo", Description: fmt.Sprintf("Já foi revisada e tem menos de %.0f%% de acertos.", FamiliarMinimumAccuracy*100)},
	{Code: "familiar", Label: "Familiar", Description: fmt.Sprintf("Tem pelo menos %.0f%% de acertos e ainda não atende ao critério de Dominada.", FamiliarMinimumAccuracy*100)},
	{Code: "mastered", Label: "Dominada", Description: fmt.Sprintf("Tem pelo menos %d revisões e uma taxa de acerto de %.0f%% ou mais.", MasteredMinimumReviews, MasteredMinimumAccuracy*100)},
}

// WordLevels returns a copy of the ordered level guide for chat and web views.
func WordLevels() []WordLevel {
	return slices.Clone(wordLevels)
}

func WordLevelLabel(code string) string {
	for _, level := range wordLevels {
		if level.Code == code {
			return level.Label
		}
	}
	return code
}
