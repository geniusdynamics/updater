package config

import (
	"net/http"
	"os"
)

type Config struct {
	GithubAPIKey string
	GitHubClient *http.Client
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func NewConfig() *Config {
	token := getEnv("GITHUB_TOKEN", "")
	return &Config{
		GithubAPIKey: token,
		GitHubClient: NewGitHubClient(token),
	}
}
