package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nethserver/ns8-updater/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// Test default values
	if len(cfg.RepoPatterns) != 1 || cfg.RepoPatterns[0] != "ns8-*" {
		t.Errorf("Expected default repo pattern 'ns8-*', got %v", cfg.RepoPatterns)
	}

	if len(cfg.ScanPatterns) != 1 || cfg.ScanPatterns[0] != "build-images.sh" {
		t.Errorf("Expected default scan pattern 'build-images.sh', got %v", cfg.ScanPatterns)
	}

	if cfg.Git.DefaultBranch != "updater" {
		t.Errorf("Expected default branch 'updater', got %s", cfg.Git.DefaultBranch)
	}

	if cfg.Update.BatchSize != 10 {
		t.Errorf("Expected default batch size 10, got %d", cfg.Update.BatchSize)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		shouldError bool
	}{
		{
			name:        "Valid default config",
			config:      config.DefaultConfig(),
			shouldError: false,
		},
		{
			name: "Empty repo patterns",
			config: &config.Config{
				RepoPatterns: []string{},
				ScanPatterns: []string{"build-images.sh"},
				Git: config.GitConfig{
					DefaultBranch:  "updater",
					CommitTemplate: "Update",
				},
				Update: config.UpdateConfig{
					BatchSize: 10,
				},
			},
			shouldError: true,
		},
		{
			name: "Empty scan patterns",
			config: &config.Config{
				RepoPatterns: []string{"ns8-*"},
				ScanPatterns: []string{},
				Git: config.GitConfig{
					DefaultBranch:  "updater",
					CommitTemplate: "Update",
				},
				Update: config.UpdateConfig{
					BatchSize: 10,
				},
			},
			shouldError: true,
		},
		{
			name: "Invalid email format",
			config: &config.Config{
				RepoPatterns: []string{"ns8-*"},
				ScanPatterns: []string{"build-images.sh"},
				Git: config.GitConfig{
					DefaultBranch:  "updater",
					CommitTemplate: "Update",
					AuthorEmail:    "invalid-email",
				},
				Update: config.UpdateConfig{
					BatchSize: 10,
				},
			},
			shouldError: true,
		},
		{
			name: "Invalid batch size",
			config: &config.Config{
				RepoPatterns: []string{"ns8-*"},
				ScanPatterns: []string{"build-images.sh"},
				Git: config.GitConfig{
					DefaultBranch:  "updater",
					CommitTemplate: "Update",
				},
				Update: config.UpdateConfig{
					BatchSize: 0,
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestRepoPatternMatching(t *testing.T) {
	cfg := &config.Config{
		RepoPatterns: []string{"ns8-*", "nethserver-*"},
		ExcludeRepos: []string{"ns8-test", "nethserver-deprecated"},
	}

	tests := []struct {
		repoName       string
		shouldMatch    bool
		shouldUpdate   bool
	}{
		{"ns8-penpot", true, true},
		{"ns8-nextcloud", true, true},
		{"ns8-test", true, false}, // matches pattern but excluded
		{"nethserver-mail", true, true},
		{"nethserver-deprecated", true, false}, // matches pattern but excluded
		{"random-repo", false, false},
		{"ns7-something", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.repoName, func(t *testing.T) {
			matches := cfg.MatchesRepoPattern(tt.repoName)
			if matches != tt.shouldMatch {
				t.Errorf("MatchesRepoPattern(%s) = %v, want %v", tt.repoName, matches, tt.shouldMatch)
			}

			shouldUpdate := cfg.ShouldUpdateRepo(tt.repoName)
			if shouldUpdate != tt.shouldUpdate {
				t.Errorf("ShouldUpdateRepo(%s) = %v, want %v", tt.repoName, shouldUpdate, tt.shouldUpdate)
			}
		})
	}
}

func TestScanPatternMatching(t *testing.T) {
	cfg := &config.Config{
		ScanPatterns: []string{"build-images.sh", "docker-compose.yml", "*.dockerfile"},
	}

	tests := []struct {
		filename    string
		shouldMatch bool
	}{
		{"build-images.sh", true},
		{"docker-compose.yml", true},
		{"app.dockerfile", true},
		{"test.dockerfile", true},
		{"random.txt", false},
		{"build-images.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			matches := cfg.MatchesScanPattern(tt.filename)
			if matches != tt.shouldMatch {
				t.Errorf("MatchesScanPattern(%s) = %v, want %v", tt.filename, matches, tt.shouldMatch)
			}
		})
	}
}

func TestConfigLoadSave(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	// Create a test config
	originalConfig := &config.Config{
		RepoPatterns: []string{"ns8-*", "test-*"},
		ExcludeRepos: []string{"ns8-deprecated"},
		ScanPatterns: []string{"build-images.sh", "Dockerfile"},
		Git: config.GitConfig{
			DefaultBranch:  "feature",
			CommitTemplate: "Test commit",
			AuthorName:     "Test Author",
			AuthorEmail:    "test@example.com",
		},
		Update: config.UpdateConfig{
			BatchSize: 5,
		},
	}

	// Save config
	err := config.SaveConfig(originalConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Compare configs
	if len(loadedConfig.RepoPatterns) != len(originalConfig.RepoPatterns) {
		t.Errorf("Repo patterns mismatch: got %v, want %v", loadedConfig.RepoPatterns, originalConfig.RepoPatterns)
	}

	if len(loadedConfig.ExcludeRepos) != len(originalConfig.ExcludeRepos) {
		t.Errorf("Exclude repos mismatch: got %v, want %v", loadedConfig.ExcludeRepos, originalConfig.ExcludeRepos)
	}

	if loadedConfig.Git.DefaultBranch != originalConfig.Git.DefaultBranch {
		t.Errorf("Default branch mismatch: got %s, want %s", loadedConfig.Git.DefaultBranch, originalConfig.Git.DefaultBranch)
	}

	if loadedConfig.Update.BatchSize != originalConfig.Update.BatchSize {
		t.Errorf("Batch size mismatch: got %d, want %d", loadedConfig.Update.BatchSize, originalConfig.Update.BatchSize)
	}
}

func TestConfigLoadNonExistent(t *testing.T) {
	// Try to load a non-existent config file
	configPath := "/tmp/non-existent-config.json"
	
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Expected no error when loading non-existent config, got: %v", err)
	}

	// Should return default config
	defaultConfig := config.DefaultConfig()
	if len(cfg.RepoPatterns) != len(defaultConfig.RepoPatterns) {
		t.Error("Expected default config when loading non-existent file")
	}
}

func TestConfigMerging(t *testing.T) {
	// Create a partial config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	partialConfig := map[string]interface{}{
		"repo_patterns": []string{"custom-*"},
		"git": map[string]interface{}{
			"default_branch": "custom-branch",
		},
	}

	data, _ := json.Marshal(partialConfig)
	err := os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write partial config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that custom values are preserved
	if len(cfg.RepoPatterns) != 1 || cfg.RepoPatterns[0] != "custom-*" {
		t.Errorf("Expected custom repo pattern, got %v", cfg.RepoPatterns)
	}

	if cfg.Git.DefaultBranch != "custom-branch" {
		t.Errorf("Expected custom branch, got %s", cfg.Git.DefaultBranch)
	}

	// Check that default values are filled in
	if len(cfg.ScanPatterns) != 1 || cfg.ScanPatterns[0] != "build-images.sh" {
		t.Errorf("Expected default scan pattern to be filled in, got %v", cfg.ScanPatterns)
	}

	if cfg.Update.BatchSize != 10 {
		t.Errorf("Expected default batch size to be filled in, got %d", cfg.Update.BatchSize)
	}
}

func TestEmailValidation(t *testing.T) {
	validEmails := []string{
		"test@example.com",
		"user.name@domain.org",
		"user+tag@example.co.uk",
		"123@example.com",
	}

	invalidEmails := []string{
		"invalid-email",
		"@example.com",
		"test@",
		"test.example.com",
		"test@.com",
	}

	for _, email := range validEmails {
		cfg := &config.Config{
			RepoPatterns: []string{"ns8-*"},
			ScanPatterns: []string{"build-images.sh"},
			Git: config.GitConfig{
				DefaultBranch:  "updater",
				CommitTemplate: "Update",
				AuthorEmail:    email,
			},
			Update: config.UpdateConfig{
				BatchSize: 10,
			},
		}

		if err := cfg.Validate(); err != nil {
			t.Errorf("Expected valid email %s to pass validation, got error: %v", email, err)
		}
	}

	for _, email := range invalidEmails {
		cfg := &config.Config{
			RepoPatterns: []string{"ns8-*"},
			ScanPatterns: []string{"build-images.sh"},
			Git: config.GitConfig{
				DefaultBranch:  "updater",
				CommitTemplate: "Update",
				AuthorEmail:    email,
			},
			Update: config.UpdateConfig{
				BatchSize: 10,
			},
		}

		if err := cfg.Validate(); err == nil {
			t.Errorf("Expected invalid email %s to fail validation", email)
		}
	}
}
