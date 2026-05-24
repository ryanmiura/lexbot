package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"lexbot/config"
	"lexbot/internal/adapter/whatsmeow"

	"go.mau.fi/whatsmeow/types/events"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize whatsmeow adapter
	waAdapter, err := whatsmeow.NewAdapter(cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to initialize whatsapp adapter: %v", err))
	}

	// Define event handler
	eventHandler := func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			// Ignore if it doesn't have an extended text message or conversation
			msg := v.Message.GetExtendedTextMessage().GetText()
			if msg == "" {
				msg = v.Message.GetConversation()
			}

			// Basic console log
			fmt.Printf("Message received from %s: %s\n", v.Info.Sender.User, msg)

			// Respond with "pong" if the message is "ping"
			if strings.ToLower(strings.TrimSpace(msg)) == "ping" {
				err := waAdapter.Send(v.Info.Chat.String(), "pong")
				if err != nil {
					fmt.Printf("Error sending message: %v\n", err)
				} else {
					fmt.Println("Replied with 'pong'")
				}
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
