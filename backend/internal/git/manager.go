package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// Manager handles Git operations for NS8 app repositories
type Manager struct {
	baseDir string
	token   string
	email   string
	name    string
}

// NewManager creates a new Git manager
func NewManager(baseDir, token, email, name string) *Manager {
	return &Manager{
		baseDir: baseDir,
		token:   token,
		email:   email,
		name:    name,
	}
}

// Repository represents a Git repository
type Repository struct {
	Name     string
	Path     string
	URL      string
	repo     *git.Repository
	manager  *Manager
}

// CloneOrUpdateRepo clones a repository or updates it if it already exists
func (m *Manager) CloneOrUpdateRepo(repoURL, repoName string) (*Repository, error) {
	repoPath := filepath.Join(m.baseDir, repoName)
	
	var repo *git.Repository
	var err error

	// Check if the repository already exists
	if _, statErr := os.Stat(repoPath); os.IsNotExist(statErr) {
		// Clone the repository
		repo, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL: repoURL,
			Auth: &http.BasicAuth{
				Username: "token",
				Password: m.token,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone repository %s: %v", repoURL, err)
		}
	} else {
		// Open existing repository
		repo, err = git.PlainOpen(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open repository %s: %v", repoPath, err)
		}

		// Pull latest changes
		workTree, err := repo.Worktree()
		if err != nil {
			return nil, fmt.Errorf("failed to get worktree: %v", err)
		}

		err = workTree.Pull(&git.PullOptions{
			Auth: &http.BasicAuth{
				Username: "token",
				Password: m.token,
			},
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return nil, fmt.Errorf("failed to pull repository: %v", err)
		}
	}

	return &Repository{
		Name:    repoName,
		Path:    repoPath,
		URL:     repoURL,
		repo:    repo,
		manager: m,
	}, nil
}

// CreateUpdateBranch creates a new branch for updates
func (r *Repository) CreateUpdateBranch(branchName string) error {
	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	// Check if branch already exists
	branchRef := plumbing.NewBranchReferenceName(branchName)
	_, err = r.repo.Reference(branchRef, true)
	if err == nil {
		// Branch exists, check it out
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
		})
		if err != nil {
			return fmt.Errorf("failed to checkout existing branch %s: %v", branchName, err)
		}
	} else {
		// Branch doesn't exist, create it
		err = workTree.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
			Create: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create and checkout branch %s: %v", branchName, err)
		}
	}

	return nil
}

// CommitChanges commits changes to the current branch
func (r *Repository) CommitChanges(message string, files []string) error {
	workTree, err := r.repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %v", err)
	}

	// Add files to staging
	for _, file := range files {
		// Make file path relative to repository root
		relPath, err := filepath.Rel(r.Path, file)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %v", file, err)
		}

		_, err = workTree.Add(relPath)
		if err != nil {
			return fmt.Errorf("failed to add file %s: %v", relPath, err)
		}
	}

	// Commit changes
	commit, err := workTree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  r.manager.name,
			Email: r.manager.email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit changes: %v", err)
	}

	fmt.Printf("Committed changes: %s\n", commit.String())
	return nil
}

// PushBranch pushes the current branch to the remote repository
func (r *Repository) PushBranch(branchName string) error {
	err := r.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("refs/heads/%s:refs/heads/%s", branchName, branchName)),
		},
		Auth: &http.BasicAuth{
			Username: "token",
			Password: r.manager.token,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to push branch %s: %v", branchName, err)
	}

	return nil
}

// GetStatus returns the status of the repository
func (r *Repository) GetStatus() (*git.Status, error) {
	workTree, err := r.repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %v", err)
	}

	status, err := workTree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %v", err)
	}

	return &status, nil
}

// HasChanges checks if there are any uncommitted changes
func (r *Repository) HasChanges() (bool, error) {
	status, err := r.GetStatus()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

// GetCurrentBranch returns the current branch name
func (r *Repository) GetCurrentBranch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %v", err)
	}

	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}

	return "", fmt.Errorf("not on a branch")
}

// ListNS8Repos discovers NS8 repositories in the base directory
func (m *Manager) ListNS8Repos() ([]*Repository, error) {
	var repos []*Repository

	// Walk through the base directory to find NS8 repositories
	err := filepath.Walk(m.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for directories that start with "ns8-"
		if info.IsDir() && strings.HasPrefix(info.Name(), "ns8-") {
			// Check if it's a git repository
			gitPath := filepath.Join(path, ".git")
			if _, err := os.Stat(gitPath); err == nil {
				// Open the repository
				repo, err := git.PlainOpen(path)
				if err != nil {
					fmt.Printf("Warning: Failed to open repository %s: %v\n", path, err)
					return nil
				}

				// Get the remote URL
				remotes, err := repo.Remotes()
				if err != nil || len(remotes) == 0 {
					fmt.Printf("Warning: No remotes found for repository %s\n", path)
					return nil
				}

				var remoteURL string
				if len(remotes[0].Config().URLs) > 0 {
					remoteURL = remotes[0].Config().URLs[0]
				}

				repos = append(repos, &Repository{
					Name:    info.Name(),
					Path:    path,
					URL:     remoteURL,
					repo:    repo,
					manager: m,
				})
			}
		}

		return nil
	})

	return repos, err
}

// CloneNS8Repos clones all NS8 repositories from a GitHub organization
func (m *Manager) CloneNS8Repos(orgURL string) error {
	// This is a simplified version - in a real implementation,
	// you would use the GitHub API to list all repositories
	// For now, we'll assume the repositories are known
	
	ns8Repos := []string{
		"ns8-penpot",
		"ns8-nextcloud",
		"ns8-dokuwiki",
		"ns8-wordpress",
		"ns8-roundcube",
		"ns8-webtop",
		"ns8-mattermost",
		"ns8-gitea",
		"ns8-jenkins",
		"ns8-grafana",
	}

	for _, repoName := range ns8Repos {
		repoURL := fmt.Sprintf("%s/%s", orgURL, repoName)
		_, err := m.CloneOrUpdateRepo(repoURL, repoName)
		if err != nil {
			fmt.Printf("Warning: Failed to clone/update %s: %v\n", repoName, err)
			continue
		}
		fmt.Printf("Successfully cloned/updated %s\n", repoName)
	}

	return nil
}
