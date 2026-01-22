package main

import (
	"log"

	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/geniusdynamics/updater/backend/internal/git"
)

func main() {
	cfg := config.NewConfig()

	githubClient := git.NewGitHubClient(cfg)
	if _, err := githubClient.GetRepositories(); err != nil {
		log.Fatal(err)
	}
	_, err := githubClient.SearchRepositories("ns8-")
	if err != nil {
	}
}
