package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/geniusdynamics/updater/backend/internal/config"
	git "github.com/go-git/go-git/v5"
	"github.com/google/go-github/v81/github"
)

type Repository struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GitHubClient struct {
	client          *github.Client
	UserName        string
	Organization    *string
	TemporaryFolder string
}

func NewGitHubClient(cfg *config.Config) *GitHubClient {
	client := github.NewClient(cfg.GitHubClient)
	return &GitHubClient{
		client:          client,
		UserName:        cfg.UserName,
		Organization:    cfg.Organization,
		TemporaryFolder: cfg.TemporaryFolder,
	}
}

func (c *GitHubClient) GetRepositories() ([]*github.Repository, error) {
	var repositories []*github.Repository
	var err error
	if c.Organization != nil && *c.Organization != "" {
		repositories, _, err = c.client.Repositories.ListByOrg(context.Background(), *c.Organization, &github.RepositoryListByOrgOptions{})
	} else {
		repositories, _, err = c.client.Repositories.ListByUser(context.Background(), c.UserName, &github.RepositoryListByUserOptions{})
	}
	if err != nil {
		return nil, fmt.Errorf("an error occurred: %w", err)
	}

	return repositories, nil
}

func (c *GitHubClient) SearchRepositories(search string) (*github.RepositoriesSearchResult, error) {
	var searchQuery string
	if c.Organization != nil && *c.Organization != "" {
		searchQuery = "org:" + *c.Organization + " " + search + " in:name"
	} else {
		searchQuery = "user:" + c.UserName + " " + search + " in:name"
	}
	repositories, _, err := c.client.Search.Repositories(context.Background(), searchQuery, &github.SearchOptions{})
	if err != nil {
		return nil, fmt.Errorf("error occurred when searching: %w", err)
	}
	for _, repo := range repositories.Repositories {
		fmt.Printf("Name: %s, Search: %s \n", *repo.Name, searchQuery)
	}
	return repositories, nil
}

func (c *GitHubClient) CloneRepositories(url string) (string, error) {
	lastUrl := strings.Split(url, "/")
	target := filepath.Join(c.TemporaryFolder, lastUrl[len(lastUrl)-1])
	_, err := git.PlainClone(target, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return "", fmt.Errorf("an error occurred while cloning repo: %s", err)
	}
	return target, nil
}

func (c *GitHubClient) RemoveClonedRepositories() error {
	if err := os.RemoveAll(c.TemporaryFolder); err != nil {
		return fmt.Errorf("failed to delete directory: %s", err)
	}
	if err := os.MkdirAll(c.TemporaryFolder, 0755); err != nil {
		return fmt.Errorf("unable to create dir: %s : %s", c.TemporaryFolder, err)
	}
	return nil
}
