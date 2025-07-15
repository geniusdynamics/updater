package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config holds the configuration for the NS8 updater
type Config struct {
	// Repository patterns to match (e.g., "ns8-*")
	RepoPatterns []string `json:"repo_patterns"`
	// Repository names to exclude from updates
	ExcludeRepos []string `json:"exclude_repos"`
	// File patterns to scan for dependencies
	ScanPatterns []string `json:"scan_patterns"`
	// Docker Hub settings
	DockerHub DockerHubConfig `json:"docker_hub"`
	// Git settings
	Git GitConfig `json:"git"`
	// Update settings
	Update UpdateConfig `json:"update"`
}

// DockerHubConfig holds Docker Hub specific configuration
type DockerHubConfig struct {
	// Username for Docker Hub authentication (optional)
	Username string `json:"username"`
	// Token for Docker Hub authentication (optional)
	Token string `json:"token"`
	// Rate limit settings
	RateLimit RateLimitConfig `json:"rate_limit"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Requests per hour
	RequestsPerHour int `json:"requests_per_hour"`
	// Enable rate limiting
	Enabled bool `json:"enabled"`
}

// GitConfig holds Git specific configuration
type GitConfig struct {
	// Default branch name for updates
	DefaultBranch string `json:"default_branch"`
	// Commit message template
	CommitTemplate string `json:"commit_template"`
	// Author name
	AuthorName string `json:"author_name"`
	// Author email
	AuthorEmail string `json:"author_email"`
}

// UpdateConfig holds update behavior configuration
type UpdateConfig struct {
	// Whether to create pull requests automatically
	CreatePullRequests bool `json:"create_pull_requests"`
	// Whether to push branches automatically
	PushBranches bool `json:"push_branches"`
	// Whether to update all dependencies or only selected ones
	UpdateAll bool `json:"update_all"`
	// Batch size for bulk updates
	BatchSize int `json:"batch_size"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		RepoPatterns: []string{"ns8-*"},
		ExcludeRepos: []string{},
		ScanPatterns: []string{"build-images.sh"},
		DockerHub: DockerHubConfig{
			RateLimit: RateLimitConfig{
				RequestsPerHour: 100,
				Enabled:         true,
			},
		},
		Git: GitConfig{
			DefaultBranch:  "updater",
			CommitTemplate: "Update Docker dependencies\n\n{{.Updates}}",
			AuthorName:     "NS8 Updater",
			AuthorEmail:    "ns8-updater@example.com",
		},
		Update: UpdateConfig{
			CreatePullRequests: false,
			PushBranches:       true,
			UpdateAll:          true,
			BatchSize:          10,
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Merge with default config for missing fields
	defaultConfig := DefaultConfig()
	if len(config.RepoPatterns) == 0 {
		config.RepoPatterns = defaultConfig.RepoPatterns
	}
	if len(config.ScanPatterns) == 0 {
		config.ScanPatterns = defaultConfig.ScanPatterns
	}
	if config.Git.DefaultBranch == "" {
		config.Git.DefaultBranch = defaultConfig.Git.DefaultBranch
	}
	if config.Git.CommitTemplate == "" {
		config.Git.CommitTemplate = defaultConfig.Git.CommitTemplate
	}
	if config.Update.BatchSize == 0 {
		config.Update.BatchSize = defaultConfig.Update.BatchSize
	}

	return &config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// MatchesRepoPattern checks if a repository name matches any of the configured patterns
func (c *Config) MatchesRepoPattern(repoName string) bool {
	for _, pattern := range c.RepoPatterns {
		if matched, _ := filepath.Match(pattern, repoName); matched {
			return true
		}
	}
	return false
}

// IsExcluded checks if a repository is in the exclusion list
func (c *Config) IsExcluded(repoName string) bool {
	for _, excluded := range c.ExcludeRepos {
		if excluded == repoName {
			return true
		}
	}
	return false
}

// ShouldUpdateRepo checks if a repository should be updated based on patterns and exclusions
func (c *Config) ShouldUpdateRepo(repoName string) bool {
	// Check if it matches any pattern
	if !c.MatchesRepoPattern(repoName) {
		return false
	}

	// Check if it's excluded
	if c.IsExcluded(repoName) {
		return false
	}

	return true
}

// MatchesScanPattern checks if a filename matches any of the scan patterns
func (c *Config) MatchesScanPattern(filename string) bool {
	for _, pattern := range c.ScanPatterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}
	return false
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if len(c.RepoPatterns) == 0 {
		return fmt.Errorf("at least one repository pattern must be specified")
	}

	if len(c.ScanPatterns) == 0 {
		return fmt.Errorf("at least one scan pattern must be specified")
	}

	// Validate patterns are valid glob patterns
	for _, pattern := range c.RepoPatterns {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return fmt.Errorf("invalid repository pattern %s: %v", pattern, err)
		}
	}

	for _, pattern := range c.ScanPatterns {
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return fmt.Errorf("invalid scan pattern %s: %v", pattern, err)
		}
	}

	// Validate Git settings
	if c.Git.DefaultBranch == "" {
		return fmt.Errorf("default branch name cannot be empty")
	}

	if c.Git.CommitTemplate == "" {
		return fmt.Errorf("commit template cannot be empty")
	}

	// Validate email format
	if c.Git.AuthorEmail != "" && !isValidEmail(c.Git.AuthorEmail) {
		return fmt.Errorf("invalid author email format: %s", c.Git.AuthorEmail)
	}

	// Validate update settings
	if c.Update.BatchSize <= 0 {
		return fmt.Errorf("batch size must be greater than 0")
	}

	return nil
}

// isValidEmail checks if an email address is valid
func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "ns8-updater", "config.json")
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	configPath := GetConfigPath()
	config := DefaultConfig()
	return SaveConfig(config, configPath)
}

// PrintConfig prints the current configuration in a human-readable format
func PrintConfig(config *Config) {
	fmt.Println("Current Configuration:")
	fmt.Printf("  Repository Patterns: %s\n", strings.Join(config.RepoPatterns, ", "))
	fmt.Printf("  Excluded Repositories: %s\n", strings.Join(config.ExcludeRepos, ", "))
	fmt.Printf("  Scan Patterns: %s\n", strings.Join(config.ScanPatterns, ", "))
	fmt.Printf("  Default Branch: %s\n", config.Git.DefaultBranch)
	fmt.Printf("  Author: %s <%s>\n", config.Git.AuthorName, config.Git.AuthorEmail)
	fmt.Printf("  Push Branches: %v\n", config.Update.PushBranches)
	fmt.Printf("  Create Pull Requests: %v\n", config.Update.CreatePullRequests)
	fmt.Printf("  Batch Size: %d\n", config.Update.BatchSize)
}
