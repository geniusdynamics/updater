package main

import (
	"fmt"
	"log"

	"github.com/geniusdynamics/updater/backend/internal/config"
	"github.com/geniusdynamics/updater/backend/internal/files"
	"github.com/geniusdynamics/updater/backend/internal/git"
)

func main() {
	err := files.LoadEnv(".env")
	if err != nil {
		log.Println(err)
	}
	cfg := config.NewConfig()

	githubClient := git.NewGitHubClient(cfg)
	if _, err := githubClient.GetRepositories(); err != nil {
		log.Fatal(err)
	}
	repos, err := githubClient.SearchRepositories("ns8-")
	if err != nil {
	}
	fileNames := map[string]bool{
		"build-images.sh": true,
	}
	for i := range 4 {
		repo := repos.Repositories[i]

		dir, err := githubClient.CloneRepository(*repo.CloneURL)
		if err != nil {
			log.Fatalf("%s \n", err)
		}
		fmt.Printf("Github Repo: %s \n", dir)
		images, err := files.FindDockerImages(dir, fileNames)
		if err != nil {
			log.Fatalf("An error occurred: %s \n", err)
		}
		for _, image := range images {
			fmt.Printf("Image: %s, %s, %s \n", image.Registry, image.Repo, image.Tag)
		}
	}
}
