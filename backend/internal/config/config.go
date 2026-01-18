package config

import (
	"net/http"
	"os"
)

type Config struct {
	GithubAPIKEY string
	GitHubClient *http.Client
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func NewConfig() *Config {
	return &Config{
		GithubAPIKEY: getEnv("GITHUB_TOKEN", ""),
		GitHubClient: NewGitHubClient(getEnv("GITHUB_TOKEN", "")),
	}
}
