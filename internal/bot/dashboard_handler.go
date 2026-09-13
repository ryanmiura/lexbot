package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"lexbot/internal/service"
)

// dashboardTokenTTL is how long a magic link stays valid before the user
// has to ask the bot for a new one. Short-lived by design — the link is
// only meant to bootstrap a longer-lived session cookie once opened, not to
// be reused as a standing credential.
const dashboardTokenTTL = 15 * time.Minute

// DashboardHandler handles the "/dashboard" command.
type DashboardHandler struct {
	messenger service.Messenger
	tokens    service.DashboardTokenRepository
	baseURL   string
}

func NewDashboardHandler(messenger service.Messenger, tokens service.DashboardTokenRepository, baseURL string) *DashboardHandler {
	return &DashboardHandler{messenger: messenger, tokens: tokens, baseURL: baseURL}
}

// Handle generates a one-time magic-link token and sends the user a URL to
// access the web dashboard. The token itself is never logged — it's a bearer
// credential, and this bot's logs are viewable by more people (Dozzle,
// Portainer) than should ever see a valid one.
func (h *DashboardHandler) Handle(ctx context.Context, chat string, userID int64) {
	if h.baseURL == "" {
		slog.Error("dashboard requested but DASHBOARD_BASE_URL is not configured", "user_id", userID)
		h.messenger.Send(chat, "❌ O dashboard ainda não está disponível. Tente novamente mais tarde.")
		return
	}

	token, err := h.tokens.CreateToken(ctx, userID, dashboardTokenTTL)
	if err != nil {
		slog.Error("failed to create dashboard token", "user_id", userID, "error", err)
		h.messenger.Send(chat, "❌ Ocorreu um erro ao gerar seu link. Tente novamente.")
		return
	}

	link := fmt.Sprintf("%s/dashboard/%s", h.baseURL, token)
	msg := fmt.Sprintf(
		"🔗 Acesse seu dashboard (link válido por %d minutos, uso único):\n%s",
		int(dashboardTokenTTL.Minutes()), link,
	)
	if err := h.messenger.Send(chat, msg); err != nil {
		slog.Error("failed to send dashboard link", "user_id", userID, "error", err)
		return
	}

	slog.Info("dashboard link issued", "user_id", userID)
}
