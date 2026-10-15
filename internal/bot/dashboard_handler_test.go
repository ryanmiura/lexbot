package bot

import (
	"context"
	"strings"
	"testing"
)

func TestVocabularyCommandsIssueMagicLinks(t *testing.T) {
	handler, messenger, repo, user := newTestHandler(t)
	ctx := context.Background()
	for _, command := range []string{"/vocabulario", "/vocabulary", "/VOCABULÁRIO", "/dashboard"} {
		t.Run(command, func(t *testing.T) {
			handler.HandleMessage(ctx, "chat", testPhone, command)
			message := messenger.last()
			if !strings.Contains(message, "Acesse seu vocabulário") || !strings.Contains(message, "15 minutos, uso único") {
				t.Fatalf("expected vocabulary link guidance, got %q", message)
			}
			_, link, ok := strings.Cut(message, "\n")
			if !ok || !strings.HasPrefix(link, "https://lexbot.example/dashboard/") {
				t.Fatalf("expected a compatible magic-link URL, got %q", link)
			}
			token := strings.TrimPrefix(link, "https://lexbot.example/dashboard/")
			got, err := repo.ConsumeToken(ctx, token)
			if err != nil || got != user.ID {
				t.Fatalf("command did not issue a usable token for its user: %d, %v", got, err)
			}
		})
	}
}
