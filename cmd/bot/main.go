package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

var client *whatsmeow.Client

func eventHandler(evt interface{}) {
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
			// Reply with "pong" using waE2E native structs
			msgResponse := &waE2E.Message{Conversation: proto.String("pong")}
			_, err := client.SendMessage(context.Background(), v.Info.Chat, msgResponse)
			
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
			} else {
				fmt.Println("Replied with 'pong'")
			}
		}
	}
}

func main() {
	// Loggers for sqlite and whatsmeow lib
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	clientLog := waLog.Stdout("Client", "INFO", true)

	// Create/Open sqlite3 database
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:session.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	// Get primary device session (if already logged in, no QR code needed)
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		panic(err)
	}

	// Instantiate whatsmeow client
	client = whatsmeow.NewClient(deviceStore, clientLog)
	
	// Add event handler
	client.AddEventHandler(eventHandler)

	// Check if already authenticated
	if client.Store.ID == nil {
		// Not logged in, generate QR Code
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Print QR Code in terminal
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("Please scan the QR Code above using WhatsApp.")
			} else {
				fmt.Println("QR Code event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		fmt.Println("Already logged in! Connected to WhatsApp.")
	}

	// Block main loop waiting for an interrupt signal (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Disconnect gracefully when closing
	client.Disconnect()
}
