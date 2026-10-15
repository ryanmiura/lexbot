package bot

import (
	"context"
	"strings"
	"testing"

	"lexbot/internal/service"
)

func TestIncomingMessageGuidance(t *testing.T) {
	handler, messenger, repo, user := newTestHandler(t)
	ctx := context.Background()
	for _, test := range []struct {
		kind service.IncomingMessageKind
		want string
	}{
		{service.IncomingMessageAudio, "Recebi seu áudio"},
		{service.IncomingMessageImage, "Recebi sua imagem"},
		{service.IncomingMessageUnsupported, "mídia que ainda não consigo processar"},
	} {
		handler.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{Kind: test.kind, Text: "resilient"})
		if !strings.Contains(messenger.last(), test.want) {
			t.Errorf("got %q, want guidance containing %q", messenger.last(), test.want)
		}
	}
	words, err := repo.ListByUser(ctx, user.ID, "")
	if err != nil || len(words) != 0 {
		t.Fatalf("media must not register words: %v, %v", words, err)
	}
	count := len(messenger.messages)
	handler.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{})
	if len(messenger.messages) != count {
		t.Fatal("ignored event must not receive a reply")
	}
	handler.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{Kind: service.IncomingMessageText, Text: "/help"})
	if !strings.Contains(messenger.last(), "Comandos do LexBot") {
		t.Fatal("text command was not dispatched")
	}
}

func TestMediaPreservesQuiz(t *testing.T) {
	handler, messenger, repo, user := newTestHandler(t)
	ctx := context.Background()
	word := saveTestWord(t, repo, user.ID, "resilient")
	handler.HandleMessage(ctx, "chat", testPhone, "/quiz")
	handler.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{Kind: service.IncomingMessageImage, Text: "1"})
	if session, err := repo.GetActiveSession(ctx, user.ID); err != nil || session != nil {
		t.Fatalf("media must not select a quiz length: %+v, %v", session, err)
	}
	handler.HandleMessage(ctx, "chat", testPhone, "1")
	session, err := repo.GetActiveSession(ctx, user.ID)
	if err != nil || session == nil || session.TotalQuestions != 1 {
		t.Fatalf("text must still configure the quiz: %+v, %v", session, err)
	}
	question := handler.quiz.runtime[user.ID].current
	for _, kind := range []service.IncomingMessageKind{service.IncomingMessageAudio, service.IncomingMessageImage, service.IncomingMessageUnsupported} {
		handler.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{Kind: kind, Text: "1"})
		current, err := repo.GetActiveSession(ctx, user.ID)
		if err != nil || current == nil || current.ID != session.ID || current.CurrentIndex != 0 || current.CorrectCount != 0 {
			t.Fatalf("media changed quiz progress: %+v, %v", current, err)
		}
		if handler.quiz.runtime[user.ID].current != question {
			t.Fatal("media changed the current question")
		}
	}
	// A fresh handler also leaves a persisted quiz untouched when media arrives.
	restarted := NewHandler(messenger, repo, service.NewWordService(unexpectedAI{t}, repo), repo, repo, service.NewQuizService(repo, repo), repo, "https://lexbot.example")
	restarted.HandleIncomingMessage(ctx, "chat", testPhone, service.IncomingMessage{Kind: service.IncomingMessageAudio})
	stored, err := repo.FindByID(ctx, word.ID)
	if err != nil || stored.TimesReviewed != 0 || stored.TimesCorrect != 0 {
		t.Fatalf("media recorded a review: %+v, %v", stored, err)
	}
	handler.HandleMessage(ctx, "chat", testPhone, "1")
	if !strings.Contains(messenger.last(), "Quiz finalizado") {
		t.Fatalf("quiz did not accept the next text answer: %s", messenger.last())
	}
	stored, err = repo.FindByID(ctx, word.ID)
	if err != nil || stored.TimesReviewed != 1 {
		t.Fatalf("expected exactly one text answer to be recorded: %+v, %v", stored, err)
	}
}
