# Variables
GO=go
GOFLAGS=-v
BUILD_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build flags with version information
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Default target
.PHONY: all
all: build

# Build the binaries
.PHONY: build
build:
	@echo "Building lobby-auth..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-auth ./cmd/lobby-auth
	@echo "Build complete: $(BUILD_DIR)/lobby-auth"
	@echo "Building lobby-data..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-data ./cmd/lobby-data
	@echo "Build complete: $(BUILD_DIR)/lobby-data"
	@echo "Building lobby-view..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-view ./cmd/lobby-view
	@echo "Build complete: $(BUILD_DIR)/lobby-view"

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-auth-linux-amd64 ./cmd/lobby-auth
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-data-linux-amd64 ./cmd/lobby-data
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-view-linux-amd64 ./cmd/lobby-view
	@echo "Linux build complete"

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-auth-darwin-amd64 ./cmd/lobby-auth
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-auth-darwin-arm64 ./cmd/lobby-auth
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-data-darwin-amd64 ./cmd/lobby-data
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-data-darwin-arm64 ./cmd/lobby-data
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-view-darwin-amd64 ./cmd/lobby-view
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-view-darwin-arm64 ./cmd/lobby-view
	@echo "macOS build complete"

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-auth-windows-amd64.exe ./cmd/lobby-auth
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-data-windows-amd64.exe ./cmd/lobby-data
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/lobby-view-windows-amd64.exe ./cmd/lobby-view
	@echo "Windows build complete"

# Run the lobby auth server
.PHONY: run-lobby-auth
run-lobby-auth: build
	@echo "Starting lobby auth server..."
	@SERVER_PORT=54231 NATS_CLIENT_PREFIX="dev-lobby-auth-" $(BUILD_DIR)/lobby-auth

# Run the lobby data server
.PHONY: run-lobby-data
run-lobby-data: build
	@echo "Starting lobby data server..."
	@SERVER_PORT=54230 NATS_CLIENT_PREFIX="dev-lobby-data-" $(BUILD_DIR)/lobby-data

# Run the lobby view server
.PHONY: run-lobby-view
run-lobby-view: build
	@echo "Starting lobby view server..."
	@SERVER_PORT=54001 NATS_CLIENT_PREFIX="dev-lobby-view-" $(BUILD_DIR)/lobby-view

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Run tests with coverage report
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin" && exit 1)
	golangci-lint run ./...

# Vet code
.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies updated"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "Installing lobby-auth..."
	$(GO) install $(LDFLAGS) ./cmd/lobby-auth
	@echo "Installing lobby-data..."
	$(GO) install $(LDFLAGS) ./cmd/lobby-data
	@echo "Installing lobby-view..."
	$(GO) install $(LDFLAGS) ./cmd/lobby-view
	@echo "Installation complete"

# Uninstall from GOPATH/bin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling lobby-auth..."
	@rm -f $(GOPATH)/bin/lobby-auth
	@echo "Uninstalling lobby-data..."
	@rm -f $(GOPATH)/bin/lobby-data
	@echo "Uninstalling lobby-view..."
	@rm -f $(GOPATH)/bin/lobby-view
	@echo "Uninstall complete"

# Development setup - run all three servers in separate terminals
.PHONY: dev
dev:
	@echo "To run in development mode, open 3 terminals and run:"
	@echo "  Terminal 1: make run-lobby-auth"
	@echo "  Terminal 2: make run-lobby-data"
	@echo "  Terminal 3: make run-lobby-view"

# Show version info
.PHONY: version
version:
	@echo "Version: ${VERSION}"
	@echo "Build Time: ${BUILD_TIME}"
	@echo "Git Commit: ${GIT_COMMIT}"

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary for current platform"
	@echo "  make build-all      - Build for Linux, macOS, and Windows"
	@echo "  make run-lobby-auth - Build and run the lobby auth server"
	@echo "  make run-lobby-data - Build and run the lobby data server"
	@echo "  make run-lobby-view - Build and run the lobby view server"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter (requires golangci-lint)"
	@echo "  make vet            - Run go vet"
	@echo "  make deps           - Download and tidy dependencies"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make install        - Install binary to GOPATH/bin"
	@echo "  make uninstall      - Remove binary from GOPATH/bin"
	@echo "  make dev            - Show development setup instructions"
	@echo "  make version        - Show version information"
	@echo "  make help           - Show this help message"
