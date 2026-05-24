package service

// Messenger abstracts sending messages to the end user,
// allowing integration with WhatsApp, Telegram, etc.
type Messenger interface {
	Send(to string, message string) error
	// SendDocument(to string, filename string, data []byte) error // future
}
