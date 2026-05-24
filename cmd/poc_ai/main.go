package main

import (
	"context"
	"encoding/json"
	"fmt"
	"lexbot/config"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

const systemPrompt = `Você é um assistente especializado em ensino de inglês para brasileiros.
Retorne APENAS um objeto JSON válido, sem markdown, sem texto antes ou depois.

Palavra recebida: "{{WORD}}"

Formato exato:
{
  "word": "a palavra exatamente como recebida",
  "translation": "tradução principal em português",
  "grammar_class": "noun | verb | adjective | adverb | conjunction | preposition | other",
  "phonetic": "transcrição IPA",
  "definition_en": "definição curta em inglês, sem usar a própria palavra",
  "example_en": "frase natural em inglês nível B1-B2",
  "example_pt": "tradução da frase",
  "synonyms": ["sinonimo1", "sinonimo2", "sinonimo3"],
  "quiz_tip": "dica curta que ajuda a lembrar a palavra sem entregá-la diretamente (ex: classe gramatical + pista semântica ou morfológica)",
  "quiz_error_explain": "explicação curta em português do significado e uso correto da palavra, ideal para quem acabou de errar uma questão. Mencione armadilhas comuns ou confusões com outras palavras se houver.",
  "connected_speech": {
    "applicable": true,
    "phenomenon": "linking | elision | assimilation | weak_form | intrusion | relaxed",
    "phenomenon_label_pt": "nome do fenômeno em português (ex: Ligação, Supressão, Forma Fraca)",
    "phenomenon_description_pt": "explicação curta do fenômeno em português, 1-2 frases",
    "example_phrase": "frase curta em inglês onde o fenômeno ocorre com a palavra",
    "example_phrase_pt": "tradução da frase",
    "isolated_sound": "como a palavra soa isolada, em representação simples",
    "connected_sound": "como a palavra soa no fluxo da frase, em representação simples",
    "connected_sound_pt": "como o connected sound seria percebido pelo ouvido de um falante nativo de português brasileiro, usando letras e recursos do português (ex: ão, ô, ê, dj, tch) para aproximar a sonoridade real",
    "tip_pt": "dica prática para o brasileiro: o que ouvir e o que prestar atenção"
  },
  "quiz": {
    "multiple_choice": {
      "question": "frase com a palavra substituída por ___",
      "correct": "a palavra",
      "distractors": ["opção errada 1", "opção errada 2", "opção errada 3"]
    },
    "complete_sentence": {
      "question": "frase diferente com ___ no lugar da palavra",
      "correct": "a palavra"
    },
    "reverse": {
      "question": "definição ou descrição para o usuário adivinhar a palavra",
      "correct": "a palavra"
    }
  }
}

Regras:
- Distratores devem ser plausíveis: mesma classe gramatical, sentido próximo
- Nunca use a própria palavra na question de "reverse" nem no "quiz_tip"
- Frases devem ser naturais e diferentes entre si
- Frases de quiz distintas da example_en`

type AIQuizQuestion struct {
	Question    string   `json:"question"`
	Correct     string   `json:"correct"`
	Distractors []string `json:"distractors,omitempty"`
}

type AIConnectedSpeech struct {
	Applicable              bool   `json:"applicable"`
	Phenomenon              string `json:"phenomenon"` // linking | elision | assimilation | weak_form | intrusion | relaxed
	PhenomenonLabelPT       string `json:"phenomenon_label_pt"`
	PhenomenonDescriptionPT string `json:"phenomenon_description_pt"`
	ExamplePhrase           string `json:"example_phrase"`
	ExamplePhrasePT         string `json:"example_phrase_pt"`
	IsolatedSound           string `json:"isolated_sound"`
	ConnectedSound          string `json:"connected_sound"`
	ConnectedSoundPT        string `json:"connected_sound_pt"` // aproximação para o ouvido brasileiro
	TipPT                   string `json:"tip_pt"`
}

type AIWordResponse struct {
	Word             string            `json:"word"`
	Translation      string            `json:"translation"`
	GrammarClass     string            `json:"grammar_class"`
	Phonetic         string            `json:"phonetic"`
	DefinitionEN     string            `json:"definition_en"`
	ExampleEN        string            `json:"example_en"`
	ExamplePT        string            `json:"example_pt"`
	Synonyms         []string          `json:"synonyms"`
	QuizTip          string            `json:"quiz_tip"`           // dica exibida em complete_sentence e reverse
	QuizErrorExplain string            `json:"quiz_error_explain"` // explicação exibida no feedback de erro
	ConnectedSpeech  AIConnectedSpeech `json:"connected_speech"`
	Quiz             struct {
		MultipleChoice   AIQuizQuestion `json:"multiple_choice"`
		CompleteSentence AIQuizQuestion `json:"complete_sentence"`
		Reverse          AIQuizQuestion `json:"reverse"`
	} `json:"quiz"`
}

// cleanJSONMarkdown removes Markdown code block formatting to ensure valid JSON parsing
func cleanJSONMarkdown(raw string) string {
	raw = strings.TrimSpace(raw)
	// Match ```json ... ``` or just ``` ... ```
	re := regexp.MustCompile("(?s)^```(?:json)?\n?(.*?)\n?```$")
	match := re.FindStringSubmatch(raw)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return raw
}

func main() {
	cfg := config.Load()
	if cfg.GroqAPIKey == "" {
		panic("GROQ_API_KEY is not set in .env")
	}

	// Configure the client for Groq API
	clientConfig := openai.DefaultConfig(cfg.GroqAPIKey)
	clientConfig.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(clientConfig)

	word := "resilient"
	fmt.Printf("Processing word: %s\n\n", word)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "llama3-70b-8192", // Using Groq's LLaMA 3 70B
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: word,
				},
			},
			Temperature: 0.2, // Low temperature for more deterministic JSON
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return
	}

	rawJSON := resp.Choices[0].Message.Content
	cleanJSON := cleanJSONMarkdown(rawJSON)

	var aiResponse AIWordResponse
	err = json.Unmarshal([]byte(cleanJSON), &aiResponse)
	if err != nil {
		fmt.Printf("JSON Unmarshal error: %v\n", err)
		fmt.Println("--- RAW STRING ---")
		fmt.Println(rawJSON)
		return
	}

	fmt.Println("✅ Unmarshal successful! Extracted data:")
	fmt.Printf("- Word: %s (%s)\n", aiResponse.Word, aiResponse.Translation)
	fmt.Printf("- Phonetic: %s\n", aiResponse.Phonetic)
	fmt.Printf("- Example: %s -> %s\n", aiResponse.ExampleEN, aiResponse.ExamplePT)
	fmt.Printf("- Quiz (Multiple Choice Question): %s\n", aiResponse.Quiz.MultipleChoice.Question)
	
	if aiResponse.ConnectedSpeech.Applicable {
		fmt.Printf("- Connected Speech Phenomenon: %s\n", aiResponse.ConnectedSpeech.PhenomenonLabelPT)
	}
}
