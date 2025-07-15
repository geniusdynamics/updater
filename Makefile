# NS8 Updater Makefile

.PHONY: all build clean test help install

# Default target
all: build

# Build the CLI tool
build:
	@echo "Building NS8 updater..."
	cd backend && go build -o ../ns8-updater ./cmd/cli
	@echo "Built successfully: ns8-updater"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f ns8-updater
	cd backend && go clean
	@echo "Cleaned successfully"

# Run tests
test:
	@echo "Running tests..."
	cd backend && go test ./...
	@echo "Tests completed"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	cd backend && go mod tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	cd backend && go fmt ./...
	@echo "Code formatted"

# Run linter
lint:
	@echo "Running linter..."
	cd backend && golangci-lint run
	@echo "Linting completed"

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	cd backend && GOOS=linux GOARCH=amd64 go build -o ../ns8-updater-linux ./cmd/cli
	@echo "Built for Linux: ns8-updater-linux"

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	cd backend && GOOS=windows GOARCH=amd64 go build -o ../ns8-updater.exe ./cmd/cli
	@echo "Built for Windows: ns8-updater.exe"

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	cd backend && GOOS=darwin GOARCH=amd64 go build -o ../ns8-updater-darwin ./cmd/cli
	@echo "Built for macOS: ns8-updater-darwin"

# Build for all platforms
build-all: build-linux build-windows build-darwin

# Install the binary to system PATH
install: build
	@echo "Installing ns8-updater to /usr/local/bin..."
	sudo cp ns8-updater /usr/local/bin/
	sudo chmod +x /usr/local/bin/ns8-updater
	@echo "Installed successfully"

# Uninstall the binary
uninstall:
	@echo "Uninstalling ns8-updater..."
	sudo rm -f /usr/local/bin/ns8-updater
	@echo "Uninstalled successfully"

# Run the scanner on test apps
demo:
	@echo "Running demo scan..."
	./ns8-updater --base-dir ./test-apps scan
	@echo "Demo completed"

# Show help
help:
	@echo "NS8 Updater Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  build         Build the CLI tool"
	@echo "  clean         Clean build artifacts"
	@echo "  test          Run tests"
	@echo "  deps          Install dependencies"
	@echo "  fmt           Format code"
	@echo "  lint          Run linter"
	@echo "  build-linux   Build for Linux"
	@echo "  build-windows Build for Windows"
	@echo "  build-darwin  Build for macOS"
	@echo "  build-all     Build for all platforms"
	@echo "  install       Install to system PATH"
	@echo "  uninstall     Uninstall from system"
	@echo "  demo          Run demo scan"
	@echo "  help          Show this help"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make demo"
	@echo "  make install"
