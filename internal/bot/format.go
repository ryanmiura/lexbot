package bot

import (
	"fmt"
	"strings"

	"lexbot/internal/service"
)

var grammarClassLabels = map[string]string{
	"noun": "Substantivo", "verb": "Verbo", "adjective": "Adjetivo",
	"adverb": "Advérbio", "conjunction": "Conjunção", "preposition": "Preposição",
}

func formatWordCard(w *service.Word) string {
	grammar := w.GrammarClass
	if translated, ok := grammarClassLabels[grammar]; ok {
		grammar = translated
	} else if len(grammar) > 0 {
		grammar = strings.ToUpper(grammar[:1]) + grammar[1:]
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "🔤 %s - 🔊 %s\n", w.Word, w.Phonetic)
	fmt.Fprintf(&sb, "📌 %s\n\n", grammar)
	fmt.Fprintf(&sb, "🇧🇷 %s\n\n", w.Translation)
	fmt.Fprintf(&sb, "📖 %s\n\n", w.DefinitionEN)
	fmt.Fprintf(&sb, "💬 \"%s\"\n", w.ExampleEN)
	fmt.Fprintf(&sb, "    \"%s\"\n", w.ExamplePT)

	if len(w.Synonyms) > 0 {
		fmt.Fprintf(&sb, "\n🔗 Sinônimos: %s\n", strings.Join(w.Synonyms, ", "))
	}

	if w.ConnectedSpeech.ExamplePhrase != "" {
		sb.WriteString("\n🔊 Connected Speech — Ligação\n")
		fmt.Fprintf(&sb, "Na fala natural: \"%s\"\n", w.ConnectedSpeech.ExamplePhrase)
		fmt.Fprintf(&sb, "🇺🇸 Soa como: %s\n", w.ConnectedSpeech.ConnectedSound)
		fmt.Fprintf(&sb, "🇧🇷 Ouvido brasileiro: %s\n", w.ConnectedSpeech.ConnectedSoundPT)
		fmt.Fprintf(&sb, "💡 %s\n", w.ConnectedSpeech.TipPT)
	}

	return sb.String()
}

// formatQuizQuestion renders a single quiz round. Formatting is intentionally
// plain for now; the polished, doc-matching templates land in a follow-up commit.
func formatQuizQuestion(index, total int, questionType string, q service.QuizQuestion, options []string, showHint bool, hint string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "❓ Pergunta %d/%d\n\n%s\n", index, total, q.Question)

	if questionType == "multiple_choice" {
		for i, opt := range options {
			fmt.Fprintf(&sb, "%d. %s\n", i+1, opt)
		}
	} else if showHint && hint != "" {
		fmt.Fprintf(&sb, "\n💡 Dica: %s\n", hint)
	}

	return sb.String()
}

// formatAnswerFeedback renders the correct/incorrect feedback for one answer.
func formatAnswerFeedback(correct bool, userAnswer string, q *activeQuestion) string {
	if correct {
		return "✅ Correto!"
	}
	return fmt.Sprintf("❌ Errado. Você respondeu \"%s\", a resposta certa é \"%s\".", userAnswer, q.correctAnswer)
}

// formatFinalResult renders the end-of-quiz scoreboard.
func formatFinalResult(session *service.QuizSession, correctWords, incorrectWords []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 Quiz finalizado!\n\nResultado: %d/%d\n", session.CorrectCount, session.TotalQuestions)
	if len(correctWords) > 0 {
		fmt.Fprintf(&sb, "\n✅ Acertou: %s\n", strings.Join(correctWords, ", "))
	}
	if len(incorrectWords) > 0 {
		fmt.Fprintf(&sb, "\n⚠️ Errou: %s\n", strings.Join(incorrectWords, ", "))
	}
	return sb.String()
}

var difficultyLabels = map[string]string{
	"new": "novo", "learning": "aprendendo", "familiar": "familiar", "mastered": "dominado",
}

func formatWordList(words []*service.Word) string {
	if len(words) == 0 {
		return "📋 Você ainda não tem nenhuma palavra salva. Envie uma palavra em inglês para começar!"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "📋 Suas palavras (%d):\n\n", len(words))
	for i, w := range words {
		label := difficultyLabels[w.Difficulty]
		if label == "" {
			label = w.Difficulty
		}
		fmt.Fprintf(&sb, "%d. %s — %s (%s)\n", i+1, w.Word, w.Translation, label)
	}

	return sb.String()
}
