package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GitHubToken string
	DatabaseURL string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	gitHubToken := os.Getenv("GITHUB_TOKEN")
	if gitHubToken == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN must be set")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL must be set")
	}

	return &Config{
		GitHubToken: gitHubToken,
		DatabaseURL: databaseURL,
	}, nil
}
