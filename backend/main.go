package main

import (
	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/geniusdynamics/updater/backend/internal/git"
)

func main() {
	config := config.NewConfig()

	repo := git.NewGithubRepository(config)

	repo.GetRepositories()
}
