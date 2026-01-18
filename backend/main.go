package main

import (
	"log"

	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/geniusdynamics/updater/backend/internal/git"
)

func main() {
	cfg := config.NewConfig()

	githubClient := git.NewGitHubClient(cfg)

	if err := githubClient.GetRepositories(); err != nil {
		log.Fatal(err)
	}
}
