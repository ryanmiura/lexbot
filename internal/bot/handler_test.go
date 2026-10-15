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

func saveTestWord(t *testing.T, repo *sqlite.Adapter, userID int64, word string) *service.Word {
	t.Helper()
	w := &service.Word{
		UserID: userID, Word: word, Translation: "tradução", GrammarClass: "noun",
		ExampleEN: "Example in English", ExamplePT: "Exemplo em português",
		Quiz: service.Quiz{
			MultipleChoice:   service.QuizQuestion{Question: "Qual a tradução?", Correct: "tradução", Distractors: []string{"outra", "diferente"}},
			CompleteSentence: service.QuizQuestion{Question: "Complete a frase", Correct: word},
			Reverse:          service.QuizQuestion{Question: "Traduza para inglês", Correct: word},
		},
	}
	if err := repo.Save(context.Background(), w); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestBilingualCommandsAndFilters(t *testing.T) {
	handler, messenger, repo, user := newTestHandler(t)
	ctx := context.Background()
	saveTestWord(t, repo, user.ID, "resilient")
	difficult := saveTestWord(t, repo, user.ID, "awkward")
	if err := repo.UpdateAfterQuiz(ctx, difficult.ID, false); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ portuguese, english string }{
		{"/ajuda", "/HELP"},
		{"/ajuda lista", "/help list"},
		{"/lista", "/LIST"},
		{"/lista novas", "/list\tNEW"},
		{"/lista DIFÍCEIS", "/list\n difficult"},
		{"/status", "/STATUS"},
	} {
		t.Run(test.english, func(t *testing.T) {
			handler.HandleMessage(ctx, "chat", testPhone, test.portuguese)
			want := messenger.last()
			handler.HandleMessage(ctx, "chat", testPhone, test.english)
			if got := messenger.last(); got != want {
				t.Errorf("%s and %s differ:\n%s\n%s", test.portuguese, test.english, want, got)
			}
		})
	}
	for _, test := range []struct{ command, want, excluded string }{
		{"/list new", "resilient", "awkward"},
		{"/list difficult", "awkward", "resilient"},
		{"/lista dificeis", "awkward", "resilient"},
	} {
		handler.HandleMessage(ctx, "chat", testPhone, test.command)
		if got := messenger.last(); !strings.Contains(got, test.want) || strings.Contains(got, test.excluded) {
			t.Errorf("%s returned incorrect words: %s", test.command, got)
		}
	}
	handler.HandleMessage(ctx, "chat", testPhone, "/list typo")
	if got := messenger.last(); !strings.Contains(got, "Não reconheci esse filtro") || strings.Contains(got, "resilient") {
		t.Fatalf("unknown filter must give guidance, got %q", got)
	}
	if _, err := repo.ListByUser(ctx, user.ID, "typo"); err != service.ErrInvalidListFilter {
		t.Fatalf("repository must reject unknown filters, got %v", err)
	}
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
