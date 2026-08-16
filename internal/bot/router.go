package bot

import "strings"

// MessageType classifies an incoming text message.
type MessageType int

const (
	MessageTypeWord MessageType = iota
	MessageTypeCommand
)

// Command is a parsed "/name args" message.
type Command struct {
	Name string
	Args string
}

// Classify splits an already-trimmed message into a command (messages
// starting with "/") or plain text, which is treated as one or more words.
func Classify(msg string) (MessageType, *Command) {
	if !strings.HasPrefix(msg, "/") {
		return MessageTypeWord, nil
	}

	parts := strings.SplitN(strings.TrimPrefix(msg, "/"), " ", 2)
	cmd := &Command{Name: strings.ToLower(strings.TrimSpace(parts[0]))}
	if len(parts) > 1 {
		cmd.Args = strings.TrimSpace(parts[1])
	}
	return MessageTypeCommand, cmd
}
