package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath     string
	GroqAPIKey string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error reading it. Relying on environment variables.")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "session.db"
	}

	groqAPIKey := os.Getenv("GROQ_API_KEY")

	return &Config{
		DBPath:     dbPath,
		GroqAPIKey: groqAPIKey,
	}
}
