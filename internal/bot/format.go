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
