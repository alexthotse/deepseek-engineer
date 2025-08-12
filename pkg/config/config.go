package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	DeepSeekAPIKey string
}

// LoadConfig loads the configuration from environment variables or a .env file.
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load()

	apiKey := os.Getenv("DEEPSEEK_API_KEY")

	return &Config{
		DeepSeekAPIKey: apiKey,
	}, nil
}
