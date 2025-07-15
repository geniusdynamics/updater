package updater

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// DockerUpdater handles updating Docker image versions in NS8 apps
type DockerUpdater struct {
	client *http.Client
}

// NewDockerUpdater creates a new DockerUpdater instance
func NewDockerUpdater() *DockerUpdater {
	return &DockerUpdater{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the name of this updater
func (d *DockerUpdater) Name() string {
	return "docker"
}

// Scan scans for Docker dependencies in NS8 apps
func (d *DockerUpdater) Scan(path string) ([]Dependency, error) {
	var dependencies []Dependency

	// Walk through the directory to find build-images.sh files
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process build-images.sh files
		if info.Name() == "build-images.sh" {
			deps, err := d.scanBuildImagesFile(filePath)
			if err != nil {
				fmt.Printf("Error scanning %s: %v\n", filePath, err)
				return nil // Continue processing other files
			}
			dependencies = append(dependencies, deps...)
		}

		return nil
	})

	return dependencies, err
}

// scanBuildImagesFile scans a build-images.sh file for Docker dependencies
func (d *DockerUpdater) scanBuildImagesFile(filePath string) ([]Dependency, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var dependencies []Dependency
	scanner := bufio.NewScanner(file)

	// Regular expressions to match Docker image references
	versionPattern := regexp.MustCompile(`(\w+)_version="([^"]+)"`)
	imagePattern := regexp.MustCompile(`docker\.io/([^:\s"]+):([^\s"\$]+)`)

	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Check for version variables like penpot_version="2.8.0"
		if match := versionPattern.FindStringSubmatch(line); match != nil {
			appName := match[1]
			currentVersion := match[2]
			
			// Try to get the latest version for this app
			latestVersion, err := d.getLatestVersionForApp(appName, currentVersion)
			if err != nil {
				fmt.Printf("Warning: Could not check latest version for %s: %v\n", appName, err)
				latestVersion = currentVersion
			}

			dependencies = append(dependencies, Dependency{
				Name:           fmt.Sprintf("%s_version", appName),
				CurrentVersion: currentVersion,
				LatestVersion:  latestVersion,
				File:           filePath,
				UpdaterName:    d.Name(),
			})
		}

		// Check for Docker image references like docker.io/postgres:15
		if matches := imagePattern.FindAllStringSubmatch(line, -1); matches != nil {
			for _, match := range matches {
				if len(match) >= 3 {
					imageName := match[1]
					currentVersion := match[2]
					
					// Skip variable references like $penpot_version
					if strings.HasPrefix(currentVersion, "$") {
						continue
					}
					
					// Get the latest version from Docker Hub
					latestVersion, err := d.getLatestDockerTag(imageName)
					if err != nil {
						fmt.Printf("Warning: Could not check latest version for %s: %v\n", imageName, err)
						latestVersion = currentVersion
					}

					dependencies = append(dependencies, Dependency{
						Name:           imageName,
						CurrentVersion: currentVersion,
						LatestVersion:  latestVersion,
						File:           filePath,
						UpdaterName:    d.Name(),
					})
				}
			}
		}
	}

	return dependencies, scanner.Err()
}

// getLatestVersionForApp gets the latest version for a specific app
func (d *DockerUpdater) getLatestVersionForApp(appName, currentVersion string) (string, error) {
	// This is a simplified version - in reality, you'd need to implement
	// specific logic for each app type (penpot, nextcloud, etc.)
	// For now, we'll try to get it from Docker Hub if it's a known pattern
	
	knownApps := map[string]string{
		"penpot":    "penpotapp/frontend",
		"nextcloud": "nextcloud",
		"postgres":  "postgres",
		"redis":     "redis",
		"mariadb":   "mariadb",
	}

	if dockerImage, exists := knownApps[appName]; exists {
		return d.getLatestDockerTag(dockerImage)
	}

	// If we can't determine the Docker image, return the current version
	return currentVersion, nil
}

// getLatestDockerTag gets the latest tag for a Docker image from Docker Hub
func (d *DockerUpdater) getLatestDockerTag(imageName string) (string, error) {
	// Docker Hub API endpoint
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags/?page_size=1&ordering=-last_updated", imageName)
	
	resp, err := d.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags for %s: %v", imageName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Docker Hub API returned status %d for %s", resp.StatusCode, imageName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	var response struct {
		Results []struct {
			Name        string `json:"name"`
			LastUpdated string `json:"last_updated"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse Docker Hub response: %v", err)
	}

	if len(response.Results) == 0 {
		return "", fmt.Errorf("no tags found for image %s", imageName)
	}

	// Return the most recently updated tag
	return response.Results[0].Name, nil
}

// ApplyUpdate applies a Docker dependency update
func (d *DockerUpdater) ApplyUpdate(dep Dependency) error {
	// Read the file
	content, err := os.ReadFile(dep.File)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %v", dep.File, err)
	}

	// Replace the old version with the new version
	oldPattern := fmt.Sprintf(`%s="%s"`, dep.Name, dep.CurrentVersion)
	newPattern := fmt.Sprintf(`%s="%s"`, dep.Name, dep.LatestVersion)
	
	// Also handle Docker image references
	oldImagePattern := fmt.Sprintf(`%s:%s`, dep.Name, dep.CurrentVersion)
	newImagePattern := fmt.Sprintf(`%s:%s`, dep.Name, dep.LatestVersion)

	updatedContent := strings.ReplaceAll(string(content), oldPattern, newPattern)
	updatedContent = strings.ReplaceAll(updatedContent, oldImagePattern, newImagePattern)

	// Write the updated content back to the file
	if err := os.WriteFile(dep.File, []byte(updatedContent), 0644); err != nil {
		return fmt.Errorf("failed to write updated file %s: %v", dep.File, err)
	}

	return nil
}
