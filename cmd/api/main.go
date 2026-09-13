package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"lexbot/config"
	"lexbot/internal/adapter/sqlite"
	"lexbot/internal/web"
)

// cmd/api serves the web dashboard: a separate process/container from
// cmd/bot, sharing the same SQLite file (SQLite's WAL mode is built for
// exactly this — one writer at a time, concurrent readers, across
// processes). A bug or panic in a dashboard HTTP handler this way can never
// take down the WhatsApp connection, and one binary can be redeployed
// without touching the other.
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "lexbot-api"))

	cfg := config.Load()

	if cfg.DashboardSecret == "" {
		panic("DASHBOARD_SECRET must be set — it signs every session cookie")
	}

	dbAdapter, err := sqlite.NewAdapter(cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("failed to initialize database: %v", err))
	}

	sessions := web.NewSessionManager(cfg.DashboardSecret)
	server := web.NewServer(sessions, dbAdapter, dbAdapter, dbAdapter, dbAdapter)

	slog.Info("dashboard listening", "port", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, server); err != nil {
		panic(fmt.Errorf("server error: %v", err))
	}
}
