package git

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/geniusdynamics/updater/backend/internal/config"
)

type Repository struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GitHubClient struct {
	client *http.Client
}

func NewGitHubClient(cfg *config.Config) *GitHubClient {
	return &GitHubClient{
		client: cfg.GitHubClient,
	}
}

func (c *GitHubClient) GetRepositories() error {
	resp, err := c.client.Get("https://api.github.com/user/repos")
	if err != nil {
		return fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error: %s, body: %s", resp.Status, string(body))
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	for i, r := range repos {
		fmt.Printf("NO: %d, Repo Name: %s\n", i, r.Name)
	}

	return nil
}
