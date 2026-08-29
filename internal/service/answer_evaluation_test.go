package service

import "testing"

func TestEvaluateAnswer(t *testing.T) {
	tests := []struct {
		name          string
		questionType  string
		userAnswer    string
		correctAnswer string
		want          bool
	}{
		{"exact match", "complete_sentence", "resilient", "resilient", true},
		{"case and whitespace insensitive", "complete_sentence", "  Resilient  ", "resilient", true},
		{"single typo accepted on complete_sentence", "complete_sentence", "resiliant", "resilient", true},
		{"single typo accepted on reverse", "reverse", "resiliant", "resilient", true},
		{"two typos rejected", "complete_sentence", "raliablu", "reliable", false},
		{"completely different word rejected", "complete_sentence", "stubborn", "resilient", false},
		{"multiple_choice rejects a typo", "multiple_choice", "resiliant", "resilient", false},
		{"multiple_choice accepts exact match", "multiple_choice", "resilient", "resilient", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvaluateAnswer(tt.questionType, tt.userAnswer, tt.correctAnswer)
			if got != tt.want {
				t.Errorf("EvaluateAnswer(%q, %q, %q) = %v, want %v", tt.questionType, tt.userAnswer, tt.correctAnswer, got, tt.want)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"resilient", "resilient", 0},
		{"resilient", "resiliant", 1},
		{"kitten", "sitting", 3},
		{"café", "cafe", 1},
	}

	for _, tt := range tests {
		got := levenshteinDistance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
