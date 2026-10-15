package service

// IncomingMessageKind describes user content independently of the chat provider.
type IncomingMessageKind int

const (
	IncomingMessageIgnored IncomingMessageKind = iota
	IncomingMessageText
	IncomingMessageAudio
	IncomingMessageImage
	IncomingMessageUnsupported
)

// IncomingMessage carries text or a media classification. Media captions are
// intentionally not vocabulary or quiz answers; media is currently unsupported.
type IncomingMessage struct {
	Kind IncomingMessageKind
	Text string
}
