package main

import (
	"context"
	"fmt"
	"lexbot/config"

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
	fmt.Printf("Processing word: %s\n", word)

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
	fmt.Println("--- RAW AI RESPONSE ---")
	fmt.Println(rawJSON)
	fmt.Println("-----------------------")
}
