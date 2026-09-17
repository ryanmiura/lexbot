package whatsmeow

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"

	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// waVersionRefreshInterval controls how often a long-running process
// re-fetches the WhatsApp Web client version it announces. WhatsApp
// periodically raises the minimum accepted version server-side (roughly
// every couple of months, per community reports); once that happens, every
// connection presenting an older version is rejected outright with
// "Client outdated (405)" and whatsmeow does not retry that specific
// failure on its own. Refreshing periodically means a long-lived process
// self-heals from this without needing a restart — though it's a
// complement to keeping the whatsmeow dependency itself up to date, not a
// substitute: SetWAVersion only changes the announced number, not any
// actual protocol/schema changes a real library update might carry.
const waVersionRefreshInterval = 24 * time.Hour

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
	refreshWAVersion()
	go periodicallyRefreshWAVersion()

	if a.client.Store.ID == nil {
		// Not logged in, generate QR Code
		qrChan, _ := a.client.GetQRChannel(context.Background())
		err := a.client.Connect()
		if err != nil {
			return err
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Print QR Code in terminal (raw stdout, not a log line — it must
				// stay human-scannable ASCII art, not JSON).
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				slog.Info("QR code generated, waiting for scan")
			} else {
				slog.Info("QR code event", "event", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err := a.client.Connect()
		if err != nil {
			return err
		}
		slog.Info("already logged in, connected to WhatsApp")
	}
	return nil
}

// refreshWAVersion fetches the WhatsApp Web client version currently
// accepted by WhatsApp's servers and, if successful, makes whatsmeow
// announce it on future connections (store.SetWAVersion is process-global,
// not per-client). A fetch failure (e.g. no network) is logged and
// otherwise ignored — Connect still proceeds with whatever version is
// already set, same as before this existed.
func refreshWAVersion() {
	latestVer, err := whatsmeow.GetLatestVersion(context.Background(), nil)
	if err != nil {
		slog.Warn("failed to fetch latest WhatsApp Web client version, keeping the current one", "error", err)
		return
	}
	store.SetWAVersion(*latestVer)
	slog.Info("WhatsApp Web client version set", "version", latestVer.String())
}

// periodicallyRefreshWAVersion re-runs refreshWAVersion on a fixed
// interval for as long as the process lives, so a long-running bot
// self-heals from WhatsApp's periodic version bumps without a restart.
func periodicallyRefreshWAVersion() {
	ticker := time.NewTicker(waVersionRefreshInterval)
	defer ticker.Stop()
	for range ticker.C {
		refreshWAVersion()
	}
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

// ResolveSenderPhone returns the real phone number of whoever sent a message,
// even when WhatsApp addressed it by LID (Linked ID) — a privacy-preserving
// synthetic identifier that replaces the phone number in Info.Sender for some
// messages. For direct messages, whatsmeow already resolves the
// phone-number counterpart into Info.SenderAlt whenever Sender is a LID (see
// parseMessageSource in the whatsmeow library); this just picks the right
// field, so callers never persist or log a LID as if it were the phone
// number.
func ResolveSenderPhone(info types.MessageInfo) string {
	if info.Sender.Server != types.HiddenUserServer {
		return info.Sender.User
	}
	if !info.SenderAlt.IsEmpty() {
		return info.SenderAlt.User
	}
	slog.Warn("message sender is a LID with no phone-number counterpart available, falling back to LID",
		"lid", info.Sender.User,
	)
	return info.Sender.User
}
