package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"lexbot/config"
	"lexbot/internal/adapter/openai"
	"lexbot/internal/adapter/whatsmeow"
	"lexbot/internal/service"

	"go.mau.fi/whatsmeow/types/events"
)

func formatWordCard(w *service.Word) string {
	grammarMap := map[string]string{
		"noun": "Substantivo", "verb": "Verbo", "adjective": "Adjetivo",
		"adverb": "Advérbio", "conjunction": "Conjunção", "preposition": "Preposição",
	}
	grammar := w.GrammarClass
	if translated, ok := grammarMap[grammar]; ok {
		grammar = translated
	} else if len(grammar) > 0 {
		grammar = strings.ToUpper(grammar[:1]) + grammar[1:]
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("🔤 %s  %s\n", w.Word, w.Phonetic))
	sb.WriteString(fmt.Sprintf("📌 %s\n\n", grammar))
	sb.WriteString(fmt.Sprintf("🇧🇷 %s\n\n", w.Translation))
	sb.WriteString(fmt.Sprintf("📖 %s\n\n", w.DefinitionEN))
	sb.WriteString(fmt.Sprintf("💬 \"%s\"\n", w.ExampleEN))
	sb.WriteString(fmt.Sprintf("    \"%s\"\n", w.ExamplePT))

	if len(w.Synonyms) > 0 {
		sb.WriteString(fmt.Sprintf("\n🔗 Sinônimos: %s\n", strings.Join(w.Synonyms, ", ")))
	}

	if w.ConnectedSpeech.ExamplePhrase != "" {
		sb.WriteString("\n🔊 Connected Speech — Ligação\n")
		sb.WriteString(fmt.Sprintf("Na fala natural: \"%s\"\n", w.ConnectedSpeech.ExamplePhrase))
		sb.WriteString(fmt.Sprintf("🇺🇸 Soa como: %s\n", w.ConnectedSpeech.ConnectedSound))
		sb.WriteString(fmt.Sprintf("🇧🇷 Ouvido brasileiro: %s\n", w.ConnectedSpeech.ConnectedSoundPT))
		sb.WriteString(fmt.Sprintf("💡 %s\n", w.ConnectedSpeech.TipPT))
	}

	return sb.String()
}

func main() {
	// Load configuration
	cfg := config.Load()

	if cfg.GroqAPIKey == "" {
		fmt.Println("Warning: GROQ_API_KEY is not set. AI features will not work.")
	}

	// Initialize whatsmeow adapter
	waAdapter, err := whatsmeow.NewAdapter(cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to initialize whatsapp adapter: %v", err))
	}

	// Initialize AI adapter and WordService
	aiAdapter := openai.NewAdapter(cfg.GroqAPIKey, "https://api.groq.com/openai/v1")
	wordService := service.NewWordService(aiAdapter)

	// Define event handler
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Ignore if it doesn't have an extended text message or conversation
			msg := v.Message.GetExtendedTextMessage().GetText()
			if msg == "" {
				msg = v.Message.GetConversation()
			}

			msg = strings.TrimSpace(msg)
			if msg == "" {
				return
			}

			// Basic console log
			fmt.Printf("Message received from %s: %s\n", v.Info.Sender.User, msg)

			// Handle commands vs words
			if strings.ToLower(msg) == "ping" {
				err := waAdapter.Send(v.Info.Chat.String(), "pong")
				if err != nil {
					fmt.Printf("Error sending message: %v\n", err)
				} else {
					fmt.Println("Replied with 'pong'")
				}
				return
			}

			// If it's a command like /quiz, /lista (to be implemented)
			if strings.HasPrefix(msg, "/") {
				// We just ignore them for now or reply that it's not implemented
				waAdapter.Send(v.Info.Chat.String(), "Comando não implementado ainda.")
				return
			}

			// Process word via AI
			fmt.Printf("Processing word: %s\n", msg)
			err = waAdapter.Send(v.Info.Chat.String(), "⏳ Processando...") // Feedback
			if err != nil {
				fmt.Printf("Error sending feedback message: %v\n", err)
			}

			// Pass background context. In real usage, you might want a timeout.
			aiWord, err := wordService.ProcessNewWord(context.Background(), msg)
			if err != nil {
				fmt.Printf("Error processing word: %v\n", err)
				waAdapter.Send(v.Info.Chat.String(), "❌ Ocorreu um erro ao processar a palavra. Tente novamente.")
				return
			}

			responseCard := formatWordCard(aiWord)
			err = waAdapter.Send(v.Info.Chat.String(), responseCard)
			if err != nil {
				fmt.Printf("Error sending card: %v\n", err)
			} else {
				fmt.Println("Successfully sent formatted word card.")
			}
		}
	}

	// Add event handler
	waAdapter.AddEventHandler(eventHandler)

	// Connect to WhatsApp
	err = waAdapter.Connect()
	if err != nil {
		panic(fmt.Errorf("failed to connect: %v", err))
	}

	// Block main loop waiting for an interrupt signal (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Disconnect gracefully when closing
	waAdapter.Disconnect()
}
