package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nethserver/ns8-updater/internal/config"
	"github.com/nethserver/ns8-updater/internal/git"
	"github.com/nethserver/ns8-updater/internal/updater"
)

// UpdaterService orchestrates the updating of NS8 apps
type UpdaterService struct {
	gitManager     *git.Manager
	dockerUpdater  *updater.DockerUpdater
	configuration  *config.Config
	repositories   []*git.Repository
	updateBranch   string
}

// NewUpdaterService creates a new UpdaterService
func NewUpdaterService(baseDir, gitToken, gitEmail, gitName string, conf *config.Config) *UpdaterService {
	return &UpdaterService{
		configuration: conf,
		gitManager:    git.NewManager(baseDir, gitToken, gitEmail, gitName),
		dockerUpdater: updater.NewDockerUpdater(),
		updateBranch:  conf.Git.DefaultBranch,
	}
}
// UpdateResult represents the result of an update operation
type UpdateResult struct {
	Repository   string                `json:"repository"`
	Dependencies []updater.Dependency  `json:"dependencies"`
	Success      bool                  `json:"success"`
	Message      string                `json:"message"`
	Branch       string                `json:"branch,omitempty"`
	CommitHash   string                `json:"commit_hash,omitempty"`
}

// ScanAll scans all NS8 repositories for dependency updates
func (s *UpdaterService) ScanAll() ([]UpdateResult, error) {
	// Ensure configuration is valid
	if err := s.configuration.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}
	var results []UpdateResult

	// Discover NS8 repositories
	repos, err := s.gitManager.ListNS8Repos()
	if err != nil {
		return nil, fmt.Errorf("failed to list NS8 repositories: %v", err)
	}

	s.repositories = repos

	for _, repo := range repos {
		result := UpdateResult{
			Repository: repo.Name,
			Success:    false,
		}

		// Check if repository should be updated
		if !s.configuration.ShouldUpdateRepo(repo.Name) {
			result.Message = "Repository excluded by configuration"
			results = append(results, result)
			continue
		}

		// Scan for dependencies
		dependencies, err := s.dockerUpdater.Scan(repo.Path)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to scan dependencies: %v", err)
			results = append(results, result)
			continue
		}

		result.Dependencies = dependencies
		result.Success = true
		result.Message = fmt.Sprintf("Found %d dependencies", len(dependencies))

		results = append(results, result)
	}

	return results, nil
}

// UpdateRepository updates a specific repository with the latest Docker versions
func (s *UpdaterService) UpdateRepository(repoName string, selectedDeps []string) (*UpdateResult, error) {
	// Find the repository
	var targetRepo *git.Repository
	for _, repo := range s.repositories {
		if repo.Name == repoName {
			targetRepo = repo
			break
		}
	}

	if targetRepo == nil {
		return nil, fmt.Errorf("repository %s not found", repoName)
	}

	result := &UpdateResult{
		Repository: repoName,
		Success:    false,
	}

	// Get current branch to ensure we're in a valid state
	_, err := targetRepo.GetCurrentBranch()
	if err != nil {
		result.Message = fmt.Sprintf("Failed to get current branch: %v", err)
		return result, nil
	}

	// Scan for dependencies
	dependencies, err := s.dockerUpdater.Scan(targetRepo.Path)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to scan dependencies: %v", err)
		return result, nil
	}

	// Filter dependencies based on selection
	var depsToUpdate []updater.Dependency
	if len(selectedDeps) == 0 {
		// If no specific dependencies are selected, update all that have newer versions
		for _, dep := range dependencies {
			if dep.CurrentVersion != dep.LatestVersion {
				depsToUpdate = append(depsToUpdate, dep)
			}
		}
	} else {
		// Update only selected dependencies
		for _, dep := range dependencies {
			for _, selectedDep := range selectedDeps {
				if dep.Name == selectedDep && dep.CurrentVersion != dep.LatestVersion {
					depsToUpdate = append(depsToUpdate, dep)
					break
				}
			}
		}
	}

	if len(depsToUpdate) == 0 {
		result.Success = true
		result.Message = "No dependencies need updating"
		result.Dependencies = dependencies
		return result, nil
	}

	// Create update branch
	branchName := fmt.Sprintf("%s-%s", s.updateBranch, time.Now().Format("20060102-150405"))
	err = targetRepo.CreateUpdateBranch(branchName)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create update branch: %v", err)
		return result, nil
	}

	// Apply updates
	var updatedFiles []string
	var updateMessages []string

	for _, dep := range depsToUpdate {
		err = s.dockerUpdater.ApplyUpdate(dep)
		if err != nil {
			result.Message = fmt.Sprintf("Failed to apply update for %s: %v", dep.Name, err)
			return result, nil
		}

		updatedFiles = append(updatedFiles, dep.File)
		updateMessages = append(updateMessages, fmt.Sprintf("%s: %s -> %s", dep.Name, dep.CurrentVersion, dep.LatestVersion))
	}

	// Remove duplicates from updated files
	updatedFiles = removeDuplicates(updatedFiles)

	// Commit changes
	commitMessage := fmt.Sprintf("Update Docker dependencies\n\n%s", strings.Join(updateMessages, "\n"))
	err = targetRepo.CommitChanges(commitMessage, updatedFiles)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to commit changes: %v", err)
		return result, nil
	}

	// Push branch
	err = targetRepo.PushBranch(branchName)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to push branch: %v", err)
		return result, nil
	}

	// Get commit hash
	head, err := targetRepo.GetCurrentBranch()
	if err != nil {
		head = "unknown"
	}

	result.Success = true
	result.Message = fmt.Sprintf("Updated %d dependencies and pushed to branch %s", len(depsToUpdate), branchName)
	result.Dependencies = depsToUpdate
	result.Branch = branchName
	result.CommitHash = head

	return result, nil
}

// UpdateAll updates all repositories with available dependency updates
func (s *UpdaterService) UpdateAll() ([]UpdateResult, error) {
	var results []UpdateResult

	// First scan all repositories
	scanResults, err := s.ScanAll()
	if err != nil {
		return nil, fmt.Errorf("failed to scan repositories: %v", err)
	}

	// Update each repository that has updates available
	for _, scanResult := range scanResults {
		if !scanResult.Success {
			results = append(results, scanResult)
			continue
		}

		// Check if there are any updates available
		hasUpdates := false
		for _, dep := range scanResult.Dependencies {
			if dep.CurrentVersion != dep.LatestVersion {
				hasUpdates = true
				break
			}
		}

		if !hasUpdates {
			result := UpdateResult{
				Repository:   scanResult.Repository,
				Dependencies: scanResult.Dependencies,
				Success:      true,
				Message:      "No updates available",
			}
			results = append(results, result)
			continue
		}

		// Update the repository
		updateResult, err := s.UpdateRepository(scanResult.Repository, nil)
		if err != nil {
			result := UpdateResult{
				Repository: scanResult.Repository,
				Success:    false,
				Message:    fmt.Sprintf("Failed to update: %v", err),
			}
			results = append(results, result)
			continue
		}

		results = append(results, *updateResult)
	}

	return results, nil
}

// CloneNS8Repos clones all NS8 repositories from a GitHub organization
func (s *UpdaterService) CloneNS8Repos(orgURL string) error {
	return s.gitManager.CloneNS8Repos(orgURL)
}

// GetRepositoryStatus returns the status of a specific repository
func (s *UpdaterService) GetRepositoryStatus(repoName string) (*git.Repository, error) {
	for _, repo := range s.repositories {
		if repo.Name == repoName {
			return repo, nil
		}
	}
	return nil, fmt.Errorf("repository %s not found", repoName)
}

// SetUpdateBranch sets the branch name to use for updates
func (s *UpdaterService) SetUpdateBranch(branchName string) {
	s.updateBranch = branchName
}

// RefreshRepositories re-scans the base directory for NS8 repositories
func (s *UpdaterService) RefreshRepositories() error {
	repos, err := s.gitManager.ListNS8Repos()
	if err != nil {
		return fmt.Errorf("failed to refresh repositories: %v", err)
	}
	s.repositories = repos
	return nil
}

// GetRepositories returns the list of discovered repositories
func (s *UpdaterService) GetRepositories() []*git.Repository {
	return s.repositories
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// ValidateRepository checks if a repository exists and is valid
func (s *UpdaterService) ValidateRepository(repoPath string) error {
	// Check if directory exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return fmt.Errorf("repository directory does not exist: %s", repoPath)
	}

	// Check if it's a git repository
	gitPath := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitPath); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Check if it has a build-images.sh file
	buildImagesPath := filepath.Join(repoPath, "build-images.sh")
	if _, err := os.Stat(buildImagesPath); os.IsNotExist(err) {
		return fmt.Errorf("no build-images.sh file found: %s", repoPath)
	}

	return nil
}
