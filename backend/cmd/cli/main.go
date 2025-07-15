package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/nethserver/ns8-updater/internal/config"
	"github.com/nethserver/ns8-updater/internal/service"
)

var (
	baseDir   string
	gitToken  string
	gitEmail  string
	gitName   string
	orgURL    string
	branchName string
)

var rootCmd = &cobra.Command{
	Use:   "ns8-updater",
	Short: "A general-purpose dependency updater for NS8 apps",
	Long:  `A CLI tool to scan for and update Docker dependencies in NS8 applications.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no subcommand is given
		fmt.Println("Use 'ns8-updater help' for more information.")
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan NS8 repositories for dependency updates",
	Long:  `Scan all NS8 repositories in the base directory for available Docker dependency updates.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		updaterService := service.NewUpdaterService(baseDir, gitToken, gitEmail, gitName, conf)
		
		results, err := updaterService.ScanAll()
		if err != nil {
			fmt.Printf("Error scanning repositories: %v\n", err)
			os.Exit(1)
		}
		
		for _, result := range results {
			fmt.Printf("Repository: %s\n", result.Repository)
			if result.Success {
				fmt.Printf("  Status: %s\n", result.Message)
				for _, dep := range result.Dependencies {
					if dep.CurrentVersion != dep.LatestVersion {
						fmt.Printf("  📦 %s: %s -> %s (UPDATE AVAILABLE)\n", dep.Name, dep.CurrentVersion, dep.LatestVersion)
					} else {
						fmt.Printf("  ✅ %s: %s (up to date)\n", dep.Name, dep.CurrentVersion)
					}
				}
			} else {
				fmt.Printf("  ❌ Error: %s\n", result.Message)
			}
			fmt.Println()
		}
	},
}

var updateCmd = &cobra.Command{
	Use:   "update [repository-name]",
	Short: "Update dependencies in NS8 repositories",
	Long:  `Update Docker dependencies in all NS8 repositories or a specific repository.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		updaterService := service.NewUpdaterService(baseDir, gitToken, gitEmail, gitName, conf)
		
		if branchName != "" {
			updaterService.SetUpdateBranch(branchName)
		}
		
		var results []service.UpdateResult
		
		if len(args) == 0 {
			// Update all repositories
			fmt.Println("Updating all NS8 repositories...")
			results, err = updaterService.UpdateAll()
		} else {
			// Update specific repository
			repoName := args[0]
			fmt.Printf("Updating repository: %s...\n", repoName)
			
			// First refresh repositories to ensure we have the latest list
			err = updaterService.RefreshRepositories()
			if err != nil {
				fmt.Printf("Error refreshing repositories: %v\n", err)
				os.Exit(1)
			}
			
			result, err := updaterService.UpdateRepository(repoName, nil)
			if err != nil {
				fmt.Printf("Error updating repository: %v\n", err)
				os.Exit(1)
			}
			results = []service.UpdateResult{*result}
		}
		
		if err != nil {
			fmt.Printf("Error updating repositories: %v\n", err)
			os.Exit(1)
		}
		
		for _, result := range results {
			fmt.Printf("Repository: %s\n", result.Repository)
			if result.Success {
				fmt.Printf("  ✅ %s\n", result.Message)
				if result.Branch != "" {
					fmt.Printf("  🌿 Branch: %s\n", result.Branch)
				}
				if len(result.Dependencies) > 0 {
					fmt.Printf("  📦 Updated dependencies:\n")
					for _, dep := range result.Dependencies {
						fmt.Printf("    - %s: %s -> %s\n", dep.Name, dep.CurrentVersion, dep.LatestVersion)
					}
				}
			} else {
				fmt.Printf("  ❌ Error: %s\n", result.Message)
			}
			fmt.Println()
		}
	},
}

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone NS8 repositories from GitHub organization",
	Long:  `Clone all NS8 repositories from a GitHub organization.`,
	Run: func(cmd *cobra.Command, args []string) {
		if orgURL == "" {
			fmt.Println("Error: Organization URL is required (use --org flag)")
			os.Exit(1)
		}
		
		// Load configuration
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		updaterService := service.NewUpdaterService(baseDir, gitToken, gitEmail, gitName, conf)
		
		fmt.Printf("Cloning NS8 repositories from: %s\n", orgURL)
		err = updaterService.CloneNS8Repos(orgURL)
		if err != nil {
			fmt.Printf("Error cloning repositories: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println("Successfully cloned NS8 repositories!")
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered NS8 repositories",
	Long:  `List all NS8 repositories found in the base directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		updaterService := service.NewUpdaterService(baseDir, gitToken, gitEmail, gitName, conf)
		
		err = updaterService.RefreshRepositories()
		if err != nil {
			fmt.Printf("Error refreshing repositories: %v\n", err)
			os.Exit(1)
		}
		
		repos := updaterService.GetRepositories()
		if len(repos) == 0 {
			fmt.Println("No NS8 repositories found in the base directory.")
			fmt.Printf("Base directory: %s\n", baseDir)
			fmt.Println("Use 'ns8-updater clone' to clone repositories first.")
			return
		}
		
		fmt.Printf("Found %d NS8 repositories:\n\n", len(repos))
		for _, repo := range repos {
			fmt.Printf("📁 %s\n", repo.Name)
			fmt.Printf("   Path: %s\n", repo.Path)
			fmt.Printf("   URL: %s\n", repo.URL)
			fmt.Println()
		}
	},
}

var jsonCmd = &cobra.Command{
	Use:   "json",
	Short: "Output scan results in JSON format",
	Long:  `Scan NS8 repositories and output results in JSON format for integration with other tools.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		updaterService := service.NewUpdaterService(baseDir, gitToken, gitEmail, gitName, conf)
		
		results, err := updaterService.ScanAll()
		if err != nil {
			fmt.Printf("Error scanning repositories: %v\n", err)
			os.Exit(1)
		}
		
		jsonOutput, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Println(string(jsonOutput))
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage NS8 updater configuration including repository patterns and exclusions.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		conf, err := config.LoadConfig(config.GetConfigPath())
		if err != nil {
			fmt.Printf("Error loading configuration: %v\n", err)
			os.Exit(1)
		}
		
		config.PrintConfig(conf)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration",
	Run: func(cmd *cobra.Command, args []string) {
		err := config.CreateDefaultConfig()
		if err != nil {
			fmt.Printf("Error creating default configuration: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Default configuration created at: %s\n", config.GetConfigPath())
	},
}

func init() {
	// Get default values from environment or use defaults
	defaultBaseDir := os.Getenv("NS8_BASE_DIR")
	if defaultBaseDir == "" {
		homeDir, _ := os.UserHomeDir()
		defaultBaseDir = filepath.Join(homeDir, "ns8-apps")
	}
	
	defaultGitToken := os.Getenv("GITHUB_TOKEN")
	defaultGitEmail := os.Getenv("GIT_EMAIL")
	if defaultGitEmail == "" {
		defaultGitEmail = "ns8-updater@example.com"
	}
	defaultGitName := os.Getenv("GIT_NAME")
	if defaultGitName == "" {
		defaultGitName = "NS8 Updater"
	}
	
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&baseDir, "base-dir", "d", defaultBaseDir, "Base directory containing NS8 repositories")
	rootCmd.PersistentFlags().StringVarP(&gitToken, "token", "t", defaultGitToken, "GitHub token for authentication")
	rootCmd.PersistentFlags().StringVarP(&gitEmail, "email", "e", defaultGitEmail, "Git email for commits")
	rootCmd.PersistentFlags().StringVarP(&gitName, "name", "n", defaultGitName, "Git name for commits")
	
	// Command-specific flags
	updateCmd.Flags().StringVarP(&branchName, "branch", "b", "updater", "Branch name for updates")
	cloneCmd.Flags().StringVarP(&orgURL, "org", "o", "", "GitHub organization URL (required)")
	
	// Add config subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	
	// Add subcommands
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(cloneCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(jsonCmd)
	rootCmd.AddCommand(configCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
