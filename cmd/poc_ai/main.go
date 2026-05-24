package main

import (
	"fmt"
	"lexbot/config"
)

func main() {
	cfg := config.Load()
	fmt.Println("Groq AI PoC initialized. API Key present:", cfg.GroqAPIKey != "")
}
