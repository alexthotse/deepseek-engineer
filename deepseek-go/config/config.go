package config

import (
	"os"
)

var APIKey string

func LoadConfig() {
	APIKey = os.Getenv("DEEPSEEK_API_KEY")
}
