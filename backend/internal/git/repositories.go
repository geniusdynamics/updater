package git

import (
	"context"
	"fmt"

	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/google/go-github/v81/github"
)

type Repository struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type GitHubClient struct {
	client       *github.Client
	UserName     string
	Organization *string
}

func NewGitHubClient(cfg *config.Config) *GitHubClient {
	client := github.NewClient(cfg.GitHubClient)
	return &GitHubClient{
		client:       client,
		UserName:     cfg.UserName,
		Organization: cfg.Organization,
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
