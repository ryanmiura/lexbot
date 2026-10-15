package bot

import (
	"strings"
	"unicode"
)

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
	msg = strings.TrimSpace(msg)
	if !strings.HasPrefix(msg, "/") {
		return MessageTypeWord, nil
	}

	name := strings.TrimPrefix(msg, "/")
	cmd := &Command{}
	if end := strings.IndexFunc(name, unicode.IsSpace); end >= 0 {
		cmd.Args = strings.TrimSpace(name[end:])
		name = name[:end]
	}
	cmd.Name = normalizeCommandName(name)
	return MessageTypeCommand, cmd
}

var commandAccents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e",
	"í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
)

func normalizeCommandName(name string) string {
	return commandAccents.Replace(strings.ToLower(strings.TrimSpace(name)))
}
