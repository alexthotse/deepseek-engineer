package main

import (
	"fmt"
	"log"

	"deepseek-eng-go/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.DeepSeekAPIKey == "" {
		log.Fatalf("DEEPSEEK_API_KEY is not set. Please set it in a .env file or as an environment variable.")
	}

	fmt.Println("Configuration loaded successfully.")
	fmt.Println("DeepSeek API Key found.")
	// The rest of the application logic will go here.
}
