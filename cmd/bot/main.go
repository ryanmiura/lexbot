package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"lexbot/config"
	"lexbot/internal/adapter/openai"
	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/adapter/whatsmeow"
	"lexbot/internal/bot"
	"lexbot/internal/service"

	"go.mau.fi/whatsmeow/types/events"
)

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

	// Initialize the SQLite adapter (users, words and quiz_questions)
	dbAdapter, err := sqlite.NewAdapter(cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to initialize database: %v", err))
	}

	// Initialize AI adapter and WordService
	aiAdapter := openai.NewAdapter(cfg.GroqAPIKey, "https://api.groq.com/openai/v1")
	wordService := service.NewWordService(aiAdapter, dbAdapter)

	// Bot handler: routes incoming messages to the right flow (word or command)
	handler := bot.NewHandler(waAdapter, dbAdapter, wordService)

	// Bridge whatsmeow events to the handler
	waAdapter.AddEventHandler(func(evt any) {
		v, ok := evt.(*events.Message)
		if !ok {
			return
		}

		msg := v.Message.GetExtendedTextMessage().GetText()
		if msg == "" {
			msg = v.Message.GetConversation()
		}

		handler.HandleMessage(context.Background(), v.Info.Chat.String(), v.Info.Sender.User, msg)
	})

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
