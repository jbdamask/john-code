package config

import (
	"fmt"
	"os"
)

type Config struct {
    APIKey string
}

func Load() (*Config, error) {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable is not set")
    }

	return &Config{
        APIKey: apiKey,
    }, nil
}
