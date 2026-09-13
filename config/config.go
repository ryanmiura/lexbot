package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath     string
	GroqAPIKey string

	// DashboardBaseURL is the public URL of the web dashboard (cmd/api),
	// e.g. "https://dashboard.example.com". Used by the bot's /dashboard
	// command to build the magic link it sends on WhatsApp.
	DashboardBaseURL string
	// DashboardSecret signs the dashboard's session cookies. Used only by
	// cmd/api; must be identical across every instance of that binary, or
	// existing sessions become invalid.
	DashboardSecret string
	// Port is the address cmd/api listens on, e.g. ":8081".
	Port string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		slog.Info("no .env file found, relying on environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "session.db"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8081"
	}

	return &Config{
		DBPath:           dbPath,
		GroqAPIKey:       os.Getenv("GROQ_API_KEY"),
		DashboardBaseURL: os.Getenv("DASHBOARD_BASE_URL"),
		DashboardSecret:  os.Getenv("DASHBOARD_SECRET"),
		Port:             port,
	}
}
