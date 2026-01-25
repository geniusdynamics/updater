package main

import (
	"log"

	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/geniusdynamics/updater/backend/internal/files"
	"github.com/geniusdynamics/updater/backend/internal/git"
)

func main() {
	cfg := config.NewConfig()
	err := files.LoadEnv(".env")
	if err != nil {
		log.Println(err)
	}

	githubClient := git.NewGitHubClient(cfg)
	if _, err := githubClient.GetRepositories(); err != nil {
		log.Fatal(err)
	}
	_, err = githubClient.SearchRepositories("ns8-")
	if err != nil {
	}
}
