package config

import (
	"net/http"
	"os"
)

type Config struct {
	GithubAPIKey string
	GitHubClient *http.Client
	UserName     string
	Organization *string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func NewConfig() *Config {
	token := getEnv("GITHUB_TOKEN", "")
	org := getEnv("GITHUB_ORGANIZATION", "")
	return &Config{
		GithubAPIKey: token,
		GitHubClient: NewHttpClient(token),
		UserName:     getEnv("GITHUB_USERNAME", ""),
		Organization: &org,
	}
}
