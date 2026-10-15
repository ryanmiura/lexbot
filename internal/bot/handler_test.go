package bot

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/service"
)

const testPhone = "5511999999999"

type recordingMessenger struct {
	messages []string
}

func (m *recordingMessenger) Send(_ string, message string) error {
	m.messages = append(m.messages, message)
	return nil
}

func (m *recordingMessenger) last() string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[len(m.messages)-1]
}

type unexpectedAI struct{ t *testing.T }

func (a unexpectedAI) ProcessWord(_ context.Context, word string) (*service.Word, error) {
	a.t.Fatalf("unexpected AI request for %q", word)
	return nil, nil
}

func newTestHandler(t *testing.T) (*Handler, *recordingMessenger, *sqlite.Adapter, *service.User) {
	t.Helper()
	repo, err := sqlite.NewAdapter(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	user, err := repo.Upsert(context.Background(), testPhone)
	if err != nil {
		t.Fatal(err)
	}
	messenger := &recordingMessenger{}
	handler := NewHandler(messenger, repo, service.NewWordService(unexpectedAI{t}, repo), repo, repo, service.NewQuizService(repo, repo), repo, "https://lexbot.example")
	return handler, messenger, repo, user
}

func TestHelpListsEveryCommand(t *testing.T) {
	handler, messenger, _, _ := newTestHandler(t)
	handler.HandleMessage(context.Background(), "chat", testPhone, "/ajuda")
	message := messenger.last()
	for _, command := range handler.commands() {
		for _, name := range command.names {
			if !strings.Contains(message, "/"+name) {
				t.Errorf("help is missing command /%s", name)
			}
		}
	}
	for _, guidance := range []string{"uma palavra ou expressão", "uma por linha", "até 10 por mensagem", "Durante um quiz"} {
		if !strings.Contains(message, guidance) {
			t.Errorf("help is missing registration or quiz guidance %q", guidance)
		}
	}
}

func TestCommandGuidance(t *testing.T) {
	handler, messenger, _, _ := newTestHandler(t)
	for _, test := range []struct{ input, want string }{
		{"/ajuda lista", "/lista novas"},
		{"/ajuda /quiz", "Iniciar ou retomar"},
		{"/ajuda inexistente", "Digite /ajuda"},
		{"/inexistente", "Não reconheci esse comando"},
	} {
		t.Run(test.input, func(t *testing.T) {
			handler.HandleMessage(context.Background(), "chat", testPhone, test.input)
			if !strings.Contains(messenger.last(), test.want) {
				t.Fatalf("got %q, want guidance containing %q", messenger.last(), test.want)
			}
		})
	}
}
