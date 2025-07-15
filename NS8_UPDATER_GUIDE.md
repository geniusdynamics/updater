# NS8 Updater Guide

A Go-based CLI tool to automatically update Docker dependencies in NS8 applications and create update branches.

## Features

- 🔍 **Automatic Dependency Scanning**: Scans NS8 app repositories for Docker dependencies
- 📦 **Docker Hub Integration**: Checks for latest Docker image versions
- 🌿 **Git Integration**: Creates update branches and commits changes
- 🔄 **Batch Updates**: Update single repositories or all repositories at once
- 📊 **JSON Output**: Machine-readable output for automation
- 🚀 **CLI Interface**: Easy-to-use command-line interface

## Installation

### Prerequisites

- Go 1.21 or higher
- Git
- GitHub token for repository access (optional for public repos)

### Building from Source

```bash
cd backend
go build -o ../ns8-updater ./cmd/cli
```

## Configuration

The tool can be configured using environment variables or command-line flags:

### Environment Variables

```bash
export NS8_BASE_DIR="/path/to/ns8-apps"      # Directory containing NS8 repositories
export GITHUB_TOKEN="your-github-token"      # GitHub token for authentication
export GIT_EMAIL="your-email@example.com"    # Git email for commits
export GIT_NAME="Your Name"                   # Git name for commits
```

### Command-line Flags

```bash
--base-dir, -d    Base directory containing NS8 repositories
--token, -t       GitHub token for authentication
--email, -e       Git email for commits
--name, -n        Git name for commits
```

## Usage

### 1. List Available NS8 Repositories

```bash
./ns8-updater list
```

Example output:
```
Found 2 NS8 repositories:

📁 ns8-penpot
   Path: /home/user/ns8-apps/ns8-penpot
   URL: https://github.com/nethserver/ns8-penpot.git

📁 ns8-nextcloud
   Path: /home/user/ns8-apps/ns8-nextcloud
   URL: https://github.com/nethserver/ns8-nextcloud.git
```

### 2. Scan for Dependency Updates

```bash
./ns8-updater scan
```

Example output:
```
Repository: ns8-penpot
  Status: Found 4 dependencies
  📦 penpot_version: 2.8.0 -> 2.9.0 (UPDATE AVAILABLE)
  ✅ postgres: 15 (up to date)
  ✅ redis: 7 (up to date)
  📦 nginx: 1.25 -> 1.26 (UPDATE AVAILABLE)
```

### 3. Update Dependencies

#### Update All Repositories
```bash
./ns8-updater update
```

#### Update Specific Repository
```bash
./ns8-updater update ns8-penpot
```

#### Use Custom Branch Name
```bash
./ns8-updater update ns8-penpot --branch feature/docker-updates
```

### 4. Clone NS8 Repositories

```bash
./ns8-updater clone --org https://github.com/nethserver
```

### 5. JSON Output for Automation

```bash
./ns8-updater json
```

Example output:
```json
[
  {
    "repository": "ns8-penpot",
    "dependencies": [
      {
        "name": "penpot_version",
        "current_version": "2.8.0",
        "latest_version": "2.9.0",
        "file": "/path/to/ns8-penpot/build-images.sh",
        "updater_name": "docker"
      }
    ],
    "success": true,
    "message": "Found 4 dependencies"
  }
]
```

## How It Works

### 1. Repository Discovery
The tool scans the base directory for directories starting with `ns8-` that contain:
- A `.git` directory (indicating a Git repository)
- A `build-images.sh` file (NS8 app build script)

### 2. Dependency Parsing
The tool parses `build-images.sh` files to find:
- **Version Variables**: `app_version="1.0.0"`
- **Docker Images**: `docker.io/postgres:15`

### 3. Version Checking
For each dependency, the tool:
- Queries Docker Hub API for the latest version
- Compares with the current version
- Identifies available updates

### 4. Update Process
When updating dependencies:
1. Creates a new branch (e.g., `updater-20240715-123456`)
2. Updates version strings in `build-images.sh`
3. Commits changes with descriptive message
4. Pushes branch to remote repository

## Supported Dependency Types

### Docker Images
- `docker.io/postgres:15`
- `docker.io/redis:7`
- `docker.io/nginx:1.25`

### Version Variables
- `penpot_version="2.8.0"`
- `nextcloud_version="28.0.0"`

## Example NS8 App Structure

```
ns8-penpot/
├── build-images.sh          # Main build script (scanned)
├── imageroot/
│   ├── actions/
│   └── systemd/
├── ui/
├── README.md
└── .git/
```

## Error Handling

The tool handles various error conditions gracefully:
- **Repository not found**: Warns and continues with other repositories
- **Docker Hub API errors**: Uses current version as fallback
- **Network issues**: Retries with timeout
- **Git errors**: Provides detailed error messages

## Contributing

### Adding New Dependency Types

To add support for new dependency types:

1. Create a new updater implementing the `Updater` interface:
```go
type Updater interface {
    Name() string
    Scan(path string) ([]Dependency, error)
    ApplyUpdate(dep Dependency) error
}
```

2. Register the updater in the service:
```go
func NewUpdaterService(baseDir, gitToken, gitEmail, gitName string) *UpdaterService {
    return &UpdaterService{
        dockerUpdater: updater.NewDockerUpdater(),
        yourUpdater:   updater.NewYourUpdater(),
        // ...
    }
}
```

### Docker Hub API Limitations

The tool uses the public Docker Hub API which has rate limits:
- 100 requests per 6 hours for anonymous users
- 200 requests per 6 hours for authenticated users

For high-volume usage, consider:
- Using Docker Hub authentication
- Implementing caching
- Using alternative registries

## Troubleshooting

### Common Issues

1. **"No NS8 repositories found"**
   - Verify the base directory path
   - Ensure directories start with `ns8-`
   - Check that directories contain `.git` and `build-images.sh`

2. **"Docker Hub API returned status 404"**
   - Image name may be incorrect
   - Image may not exist on Docker Hub
   - Check network connectivity

3. **"Failed to push branch"**
   - Verify GitHub token permissions
   - Check repository write access
   - Ensure Git credentials are configured

### Debug Mode

For detailed logging, use:
```bash
./ns8-updater --debug scan
```

### Manual Testing

To test the updater with a specific repository:
```bash
./ns8-updater --base-dir /path/to/single/repo scan
```

## Automation Examples

### CI/CD Pipeline

```yaml
name: Update NS8 Dependencies
on:
  schedule:
    - cron: '0 2 * * 1'  # Weekly on Monday at 2 AM

jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Update dependencies
        run: |
          ./ns8-updater update --token ${{ secrets.GITHUB_TOKEN }}
```

### Cron Job

```bash
# Add to crontab for daily updates
0 2 * * * /path/to/ns8-updater update --base-dir /home/user/ns8-apps
```

## License

This tool is released under the GPL-3.0 license.

## Support

For issues and feature requests, please create an issue in the repository.
