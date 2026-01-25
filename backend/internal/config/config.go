package config

import (
	"fmt"
	"net/http"
	"os"
)

type Config struct {
	GithubAPIKey    string
	GitHubClient    *http.Client
	UserName        string
	Organization    *string
	TemporaryFolder string
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
	tempFolder := getEnv("TEMPORARY_FOLDER", "/tmp/ns8-updater/")
	_ = checkTempDirExists(tempFolder)
	return &Config{
		GithubAPIKey:    token,
		GitHubClient:    NewHttpClient(token),
		UserName:        getEnv("GITHUB_USERNAME", ""),
		Organization:    &org,
		TemporaryFolder: tempFolder,
	}
}

func checkTempDirExists(dir string) error {
	info, err := os.Stat(dir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s exists but is not a directory", dir)
		}
		return nil
	}

	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		return nil
	}

	return fmt.Errorf("cannot stat %s: %w", dir, err)
}
