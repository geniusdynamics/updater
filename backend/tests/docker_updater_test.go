package tests

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nethserver/ns8-updater/internal/updater"
)

func TestDockerUpdaterName(t *testing.T) {
	dockerUpdater := updater.NewDockerUpdater()
	if dockerUpdater.Name() != "docker" {
		t.Errorf("Expected updater name 'docker', got '%s'", dockerUpdater.Name())
	}
}

func TestDockerUpdaterScanBuildImages(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a test build-images.sh file
	buildImagesContent := `#!/bin/bash
set -e

# Test NS8 app build script
demo_version="1.0.0"
postgres_version="15.0"

# Create container
container=$(buildah from scratch)

# Add images
buildah config --entrypoint=/ \
	--label="org.nethserver.images=docker.io/postgres:15 docker.io/redis:7 docker.io/nginx:1.25" \
	"${container}"

buildah commit "${container}" "${repobase}/${reponame}"
`

	buildImagesPath := filepath.Join(tempDir, "build-images.sh")
	err := ioutil.WriteFile(buildImagesPath, []byte(buildImagesContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test build-images.sh: %v", err)
	}

	// Test scanning
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	// Check that we found some dependencies
	if len(dependencies) == 0 {
		t.Error("Expected to find dependencies but got none")
	}

	// Check for specific dependencies
	foundDemoVersion := false
	foundPostgresVersion := false
	foundPostgresImage := false
	foundRedisImage := false
	foundNginxImage := false

	for _, dep := range dependencies {
		switch dep.Name {
		case "demo_version":
			foundDemoVersion = true
			if dep.CurrentVersion != "1.0.0" {
				t.Errorf("Expected demo_version to be '1.0.0', got '%s'", dep.CurrentVersion)
			}
		case "postgres_version":
			foundPostgresVersion = true
			if dep.CurrentVersion != "15.0" {
				t.Errorf("Expected postgres_version to be '15.0', got '%s'", dep.CurrentVersion)
			}
		case "postgres":
			foundPostgresImage = true
			if dep.CurrentVersion != "15" {
				t.Errorf("Expected postgres image version to be '15', got '%s'", dep.CurrentVersion)
			}
		case "redis":
			foundRedisImage = true
			if dep.CurrentVersion != "7" {
				t.Errorf("Expected redis image version to be '7', got '%s'", dep.CurrentVersion)
			}
		case "nginx":
			foundNginxImage = true
			if dep.CurrentVersion != "1.25" {
				t.Errorf("Expected nginx image version to be '1.25', got '%s'", dep.CurrentVersion)
			}
		}
	}

	if !foundDemoVersion {
		t.Error("Expected to find demo_version dependency")
	}
	if !foundPostgresVersion {
		t.Error("Expected to find postgres_version dependency")
	}
	if !foundPostgresImage {
		t.Error("Expected to find postgres image dependency")
	}
	if !foundRedisImage {
		t.Error("Expected to find redis image dependency")
	}
	if !foundNginxImage {
		t.Error("Expected to find nginx image dependency")
	}
}

func TestDockerUpdaterScanMultipleFiles(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	
	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "ns8-app1")
	subDir2 := filepath.Join(tempDir, "ns8-app2")
	subDir3 := filepath.Join(tempDir, "not-ns8-app")
	
	err := os.MkdirAll(subDir1, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir1: %v", err)
	}
	err = os.MkdirAll(subDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir2: %v", err)
	}
	err = os.MkdirAll(subDir3, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir3: %v", err)
	}

	// Create build-images.sh files
	buildImages1 := `#!/bin/bash
app1_version="1.0.0"
buildah config --label="org.nethserver.images=docker.io/postgres:14" "${container}"
`
	buildImages2 := `#!/bin/bash
app2_version="2.0.0"
buildah config --label="org.nethserver.images=docker.io/mysql:8.0" "${container}"
`
	buildImages3 := `#!/bin/bash
app3_version="3.0.0"
buildah config --label="org.nethserver.images=docker.io/redis:6" "${container}"
`

	// Write files
	err = ioutil.WriteFile(filepath.Join(subDir1, "build-images.sh"), []byte(buildImages1), 0644)
	if err != nil {
		t.Fatalf("Failed to write build-images.sh for app1: %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(subDir2, "build-images.sh"), []byte(buildImages2), 0644)
	if err != nil {
		t.Fatalf("Failed to write build-images.sh for app2: %v", err)
	}
	err = ioutil.WriteFile(filepath.Join(subDir3, "build-images.sh"), []byte(buildImages3), 0644)
	if err != nil {
		t.Fatalf("Failed to write build-images.sh for app3: %v", err)
	}

	// Test scanning
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	// Should find dependencies from all three files
	if len(dependencies) < 6 { // At least 3 version variables + 3 images
		t.Errorf("Expected at least 6 dependencies, got %d", len(dependencies))
	}

	// Check specific dependencies
	foundApp1 := false
	foundApp2 := false
	foundApp3 := false

	for _, dep := range dependencies {
		switch dep.Name {
		case "app1_version":
			foundApp1 = true
		case "app2_version":
			foundApp2 = true
		case "app3_version":
			foundApp3 = true
		}
	}

	if !foundApp1 {
		t.Error("Expected to find app1_version dependency")
	}
	if !foundApp2 {
		t.Error("Expected to find app2_version dependency")
	}
	if !foundApp3 {
		t.Error("Expected to find app3_version dependency")
	}
}

func TestDockerUpdaterSkipVariableReferences(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a test build-images.sh file with variable references
	buildImagesContent := `#!/bin/bash
set -e

penpot_version="2.8.0"

# This should be skipped because it uses variables
buildah config --entrypoint=/ \
	--label="org.nethserver.images=docker.io/postgres:15 docker.io/redis:7 docker.io/penpotapp/frontend:$penpot_version docker.io/penpotapp/backend:$penpot_version" \
	"${container}"
`

	buildImagesPath := filepath.Join(tempDir, "build-images.sh")
	err := ioutil.WriteFile(buildImagesPath, []byte(buildImagesContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test build-images.sh: %v", err)
	}

	// Test scanning
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	// Should find the version variable and concrete image versions, but not variable references
	foundVariableRef := false
	for _, dep := range dependencies {
		if dep.CurrentVersion == "$penpot_version" {
			foundVariableRef = true
		}
	}

	if foundVariableRef {
		t.Error("Expected to skip variable references like $penpot_version")
	}
}

func TestDockerUpdaterApplyUpdate(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "build-images.sh")
	
	originalContent := `#!/bin/bash
set -e

penpot_version="2.8.0"
postgres_version="15.0"

buildah config --label="org.nethserver.images=docker.io/postgres:15 docker.io/redis:7" "${container}"
`

	err := ioutil.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create dependency update
	dep := updater.Dependency{
		Name:           "penpot_version",
		CurrentVersion: "2.8.0",
		LatestVersion:  "2.9.0",
		File:           testFile,
		UpdaterName:    "docker",
	}

	// Apply update
	dockerUpdater := updater.NewDockerUpdater()
	err = dockerUpdater.ApplyUpdate(dep)
	if err != nil {
		t.Fatalf("Failed to apply update: %v", err)
	}

	// Read updated content
	updatedContent, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check that the version was updated
	expectedContent := `#!/bin/bash
set -e

penpot_version="2.9.0"
postgres_version="15.0"

buildah config --label="org.nethserver.images=docker.io/postgres:15 docker.io/redis:7" "${container}"
`

	if string(updatedContent) != expectedContent {
		t.Errorf("Expected updated content:\n%s\n\nGot:\n%s", expectedContent, string(updatedContent))
	}
}

func TestDockerUpdaterApplyImageUpdate(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "build-images.sh")
	
	originalContent := `#!/bin/bash
set -e

buildah config --label="org.nethserver.images=docker.io/postgres:15 docker.io/redis:7" "${container}"
`

	err := ioutil.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create dependency update for image
	dep := updater.Dependency{
		Name:           "postgres",
		CurrentVersion: "15",
		LatestVersion:  "16",
		File:           testFile,
		UpdaterName:    "docker",
	}

	// Apply update
	dockerUpdater := updater.NewDockerUpdater()
	err = dockerUpdater.ApplyUpdate(dep)
	if err != nil {
		t.Fatalf("Failed to apply update: %v", err)
	}

	// Read updated content
	updatedContent, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	// Check that the image version was updated
	expectedContent := `#!/bin/bash
set -e

buildah config --label="org.nethserver.images=docker.io/postgres:16 docker.io/redis:7" "${container}"
`

	if string(updatedContent) != expectedContent {
		t.Errorf("Expected updated content:\n%s\n\nGot:\n%s", expectedContent, string(updatedContent))
	}
}

func TestDockerUpdaterScanEmptyFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create an empty build-images.sh file
	buildImagesPath := filepath.Join(tempDir, "build-images.sh")
	err := ioutil.WriteFile(buildImagesPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create empty build-images.sh: %v", err)
	}

	// Test scanning
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	// Should find no dependencies
	if len(dependencies) != 0 {
		t.Errorf("Expected no dependencies from empty file, got %d", len(dependencies))
	}
}

func TestDockerUpdaterScanCommentsAndEmptyLines(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a build-images.sh file with comments and empty lines
	buildImagesContent := `#!/bin/bash

# This is a comment
set -e

# Another comment
# demo_version="commented_out"

demo_version="1.0.0"

# More comments
buildah config --label="org.nethserver.images=docker.io/postgres:15" "${container}"
`

	buildImagesPath := filepath.Join(tempDir, "build-images.sh")
	err := ioutil.WriteFile(buildImagesPath, []byte(buildImagesContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test build-images.sh: %v", err)
	}

	// Test scanning
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan dependencies: %v", err)
	}

	// Should find dependencies but ignore comments
	foundCommentedVersion := false
	foundActualVersion := false

	for _, dep := range dependencies {
		if dep.Name == "demo_version" {
			if dep.CurrentVersion == "commented_out" {
				foundCommentedVersion = true
			} else if dep.CurrentVersion == "1.0.0" {
				foundActualVersion = true
			}
		}
	}

	if foundCommentedVersion {
		t.Error("Expected to ignore commented version variables")
	}
	if !foundActualVersion {
		t.Error("Expected to find actual version variable")
	}
}

func TestDockerUpdaterScanNonExistentDirectory(t *testing.T) {
	// Test scanning non-existent directory
	dockerUpdater := updater.NewDockerUpdater()
	dependencies, err := dockerUpdater.Scan("/non/existent/path")
	
	// Should handle error gracefully
	if err == nil {
		t.Error("Expected error when scanning non-existent directory")
	}
	
	if len(dependencies) != 0 {
		t.Errorf("Expected no dependencies from non-existent directory, got %d", len(dependencies))
	}
}

func TestDockerUpdaterApplyUpdateNonExistentFile(t *testing.T) {
	// Create dependency update for non-existent file
	dep := updater.Dependency{
		Name:           "test_version",
		CurrentVersion: "1.0.0",
		LatestVersion:  "2.0.0",
		File:           "/non/existent/file.sh",
		UpdaterName:    "docker",
	}

	// Apply update
	dockerUpdater := updater.NewDockerUpdater()
	err := dockerUpdater.ApplyUpdate(dep)
	
	// Should return error
	if err == nil {
		t.Error("Expected error when updating non-existent file")
	}
}
