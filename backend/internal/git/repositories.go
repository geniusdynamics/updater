package git

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geniusdynamics/updater/backend/internal/config"
)

type Repository struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type GitRepository struct {
	client *config.Config
}

func NewGithubRepository(cfg *config.Config) *GitRepository {
	return &GitRepository{
		client: cfg,
	}
}

func (repo *GitRepository) GetRepositories() {
	resp, err := repo.client.GitHubClient.Get("https://api.github.com/user/repos")
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}
	defer resp.Body.Close()

	fmt.Println("Status: ", resp.Status)
	if resp.StatusCode != http.StatusOK {
		fmt.Println("API ERROR: %d \n", resp.Status)
		return
	}
	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		fmt.Printf("Failed to parse JSON: %v \n", err)
		return
	}
	for i, r := range repos {
		fmt.Printf("NO: %d, Repo Name: %s \n", i, r.Name)
	}
	fmt.Println(resp.Body)
}
