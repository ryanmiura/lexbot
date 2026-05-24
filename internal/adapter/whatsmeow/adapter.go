package whatsmeow

import (
	"context"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// Adapter implements the service.Messenger interface using whatsmeow
type Adapter struct {
	client *whatsmeow.Client
}

// NewAdapter creates a new whatsmeow adapter
func NewAdapter(dbPath string) (*Adapter, error) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	clientLog := waLog.Stdout("Client", "INFO", true)

	container, err := sqlstore.New(context.Background(), "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
	if err != nil {
		return nil, err
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, err
	}

	client := whatsmeow.NewClient(deviceStore, clientLog)
	return &Adapter{client: client}, nil
}

// Connect starts the connection and handles QR code generation if needed
func (a *Adapter) Connect() error {
	if a.client.Store.ID == nil {
		// Not logged in, generate QR Code
		qrChan, _ := a.client.GetQRChannel(context.Background())
		err := a.client.Connect()
		if err != nil {
			return err
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
		err := a.client.Connect()
		if err != nil {
			return err
		}
		fmt.Println("Already logged in! Connected to WhatsApp.")
	}
	return nil
}

// Disconnect closes the connection
func (a *Adapter) Disconnect() {
	a.client.Disconnect()
}

// AddEventHandler registers an event handler
func (a *Adapter) AddEventHandler(handler whatsmeow.EventHandler) {
	a.client.AddEventHandler(handler)
}

// Send implements the service.Messenger interface
func (a *Adapter) Send(to string, message string) error {
	jid, err := types.ParseJID(to)
	if err != nil {
		return err
	}

	msgResponse := &waE2E.Message{Conversation: proto.String(message)}
	_, err = a.client.SendMessage(context.Background(), jid, msgResponse)
	return err
}
