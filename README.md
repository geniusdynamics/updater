# NS8-Updater

A tool to automatically update NethServer 8 module dependencies, including Docker images and other resources.

## Architecture

This project is composed of two main parts:

1.  **Backend (Go):** A Go application that exposes a REST API. It's responsible for all the core logic:
    *   Scanning for NethServer module repositories.
    *   Parsing `build-images.sh` files to find current Docker image versions.
    *   Querying container registries (like Docker Hub) for the latest image tags.
    *   Updating the script files with new versions.
    *   Handling git operations (committing and pushing changes).

2.  **Frontend (Vue.js):** A single-page web application that provides a user-friendly interface for the backend.
    *   Displays a list of all discovered modules.
    *   Shows the current version of each dependency and whether an update is available.
    *   Allows users to trigger updates for selected modules.
    *   Provides real-time feedback and logs from the update process.

## Development Workflow

### Backend (Go)

1.  **Navigate to the backend directory:**
    ```bash
    cd ns8-updater/backend
    ```

2.  **Initialize the Go module:**
    ```bash
    go mod init ns8-updater/backend
    ```

3.  **Install dependencies:** We'll use `gin` for the web server and `go-git` for git operations.
    ```bash
    go get -u github.com/gin-gonic/gin
    go get -u github.com/go-git/go-git/v5
    ```

4.  **Run the backend server:**
    ```bash
    go run main.go
    ```

### Frontend (Vue.js)

1.  **Navigate to the frontend directory:**
    ```bash
    cd ns8-updater/frontend
    ```

2.  **Create the Vue.js project:**
    ```bash
    npm create vue@latest .
    ```
    *Follow the prompts to set up your Vue project (e.g., with TypeScript, Pinia, etc.).*

3.  **Install dependencies:**
    ```bash
    npm install
    ```

4.  **Run the frontend development server:**
    ```bash
    npm run dev
    ```

## Scalable Dependency Updates

To make this system scalable and capable of handling any type of dependency, we can use a plugin-based architecture.

1.  **Define an `Updater` interface in Go:**
    ```go
    package main

    type Updater interface {
        // Name returns the name of the updater (e.g., "docker", "github-release").
        Name() string
        // CheckForUpdates scans a file or directory and returns a list of available updates.
        CheckForUpdates(filePath string) ([]Update, error)
        // ApplyUpdate applies a specific update to a file.
        ApplyUpdate(update Update) error
    }

    type Update struct {
        // The file to update.
        File string
        // The old version string.
        OldVersion string
        // The new version string.
        NewVersion string
        // The updater to use.
        UpdaterName string
    }
    ```

2.  **Implement different updaters:**
    *   **DockerUpdater:** Parses `build-images.sh` files, queries the Docker Hub API, and updates image tags.
    *   **GitHubReleaseUpdater:** Checks for new releases of a GitHub repository (useful for updating forks or other dependencies).
    *   **GitSubmoduleUpdater:** Checks for new commits in git submodules.

3.  **The backend API will:**
    *   Scan a directory for projects.
    *   For each project, it will run all registered `Updater` plugins.
    *   It will then aggregate all the available updates and present them to the frontend.

This approach allows you to easily add new update logic for different kinds of dependencies in the future without changing the core application.
