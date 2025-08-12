package main

import (
	"log"

	"deepseek-eng-go/internal/api"
	"deepseek-eng-go/internal/cli"
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

	apiClient := api.NewClient(cfg.DeepSeekAPIKey)

	cli.Run(apiClient)
}
