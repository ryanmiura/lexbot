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

// optionEmoji renders 1-based option numbers as the keycap emoji digits used
// in the docs' multiple-choice template (1️⃣, 2️⃣, ...), falling back to a
// plain number for positions beyond what the emoji set covers.
var optionEmoji = []string{"1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣"}

func optionLabel(i int) string {
	if i >= 1 && i <= len(optionEmoji) {
		return optionEmoji[i-1]
	}
	return fmt.Sprintf("%d.", i)
}

// formatQuizQuestion renders a single quiz round, following the per-type
// templates described in the docs (section 7.4).
func formatQuizQuestion(index, total int, questionType string, q service.QuizQuestion, options []string, showHint bool, hint string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "❓ Pergunta %d/%d\n\n", index, total)

	switch questionType {
	case "multiple_choice":
		fmt.Fprintf(&sb, "\"%s\"\n\nQual palavra completa a frase?\n\n", q.Question)
		for i, opt := range options {
			fmt.Fprintf(&sb, "%s %s\n", optionLabel(i+1), opt)
		}
		fmt.Fprintf(&sb, "\nResponda com o número (1-%d):", len(options))
	case "reverse":
		fmt.Fprintf(&sb, "Qual palavra em inglês descreve:\n\n\"%s\"\n", q.Question)
		if showHint && hint != "" {
			fmt.Fprintf(&sb, "\n💡 Dica: %s\n", hint)
		}
		sb.WriteString("\nDigite a palavra em inglês:")
	default: // complete_sentence
		fmt.Fprintf(&sb, "Complete a frase com a palavra correta:\n\n\"%s\"\n", q.Question)
		if showHint && hint != "" {
			fmt.Fprintf(&sb, "\n💡 Dica: %s\n", hint)
		}
		sb.WriteString("\nDigite a palavra:")
	}

	return sb.String()
}

// formatAnswerFeedback renders the correct/incorrect feedback for one
// answer, following the docs' template (section 7.5). Feedback is built
// entirely from data saved at word-registration time — no AI call happens
// here.
func formatAnswerFeedback(correct, hasNext bool, userAnswer string, q *activeQuestion) string {
	var sb strings.Builder

	if correct {
		sb.WriteString("✅ Correto!\n")
	} else {
		sb.WriteString("❌ Quase lá!\n\n")
		fmt.Fprintf(&sb, "Você respondeu: \"%s\"\n", userAnswer)
		fmt.Fprintf(&sb, "Resposta correta: %s\n", q.correctAnswer)
		if q.quizErrorExplain != "" {
			fmt.Fprintf(&sb, "\n💡 %s\n", q.quizErrorExplain)
		}
	}

	if q.exampleEN != "" {
		fmt.Fprintf(&sb, "\n💬 \"%s\"\n    \"%s\"\n", q.exampleEN, q.examplePT)
	}

	if hasNext {
		sb.WriteString("\nPróxima pergunta...")
	}

	return sb.String()
}

// formatFinalResult renders the end-of-quiz scoreboard (section 7.6). The
// cross-session comparison and pattern-detection suggestions described in
// the docs are out of scope here — they'd need historical data and AI
// analysis beyond what this phase covers.
func formatFinalResult(session *service.QuizSession, correctWords, incorrectWords []string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 Quiz finalizado!\n\nResultado: %d/%d\n", session.CorrectCount, session.TotalQuestions)

	if len(correctWords) > 0 {
		fmt.Fprintf(&sb, "\n✅ Você domina bem:\n   %s\n", strings.Join(correctWords, ", "))
	}
	if len(incorrectWords) > 0 {
		fmt.Fprintf(&sb, "\n⚠️ Ainda precisa de atenção:\n   %s\n", strings.Join(incorrectWords, ", "))
	}

	sb.WriteString("\nDigite /quiz para praticar novamente.")

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
