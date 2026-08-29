package service

import "strings"

// levenshteinTolerance is the maximum edit distance still accepted as a
// correct answer for free-text question types, to forgive small spelling
// mistakes without accepting a genuinely different word.
const levenshteinTolerance = 1

// EvaluateAnswer checks a user's answer against the correct one, following
// the rules described in the project docs: multiple_choice requires an
// exact match, while complete_sentence and reverse accept spelling
// variations within a Levenshtein distance of levenshteinTolerance.
func EvaluateAnswer(questionType, userAnswer, correctAnswer string) bool {
	normalizedUser := normalizeAnswer(userAnswer)
	normalizedCorrect := normalizeAnswer(correctAnswer)

	if normalizedUser == normalizedCorrect {
		return true
	}
	if questionType == "multiple_choice" {
		return false
	}

	return levenshteinDistance(normalizedUser, normalizedCorrect) <= levenshteinTolerance
}

func normalizeAnswer(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// levenshteinDistance computes the edit distance between two strings using
// the classic dynamic-programming algorithm, operating on runes so accented
// Portuguese characters count as a single edit.
func levenshteinDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	la, lb := len(ar), len(br)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}
