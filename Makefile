# Variables
BINARY_NAME=login-server
GO=go
GOFLAGS=-v
BUILD_DIR=./bin
SRC_DIR=.
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build flags with version information
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(SRC_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(SRC_DIR)
	@echo "Linux build complete"

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(SRC_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(SRC_DIR)
	@echo "macOS build complete"

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(SRC_DIR)
	@echo "Windows build complete"

# Run the auth server
.PHONY: run-auth
run-auth: build
	@echo "Starting auth server..."
	$(BUILD_DIR)/$(BINARY_NAME) --role=auth

# Run the data server
.PHONY: run-data
run-data: build
	@echo "Starting data server..."
	$(BUILD_DIR)/$(BINARY_NAME) --role=data

# Run the view server
.PHONY: run-view
run-view: build
	@echo "Starting view server..."
	$(BUILD_DIR)/$(BINARY_NAME) --role=view

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
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) $(SRC_DIR)
	@echo "Installation complete"

# Uninstall from GOPATH/bin
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstall complete"

# Development setup - run all three servers in separate terminals
.PHONY: dev
dev:
	@echo "To run in development mode, open 3 terminals and run:"
	@echo "  Terminal 1: make run-auth"
	@echo "  Terminal 2: make run-data"
	@echo "  Terminal 3: make run-view"
	@echo ""
	@echo "Or use environment variables to configure:"
	@echo "  GOFFXI_AUTH_PORT=54230 make run-auth"
	@echo "  GOFFXI_DATA_PORT=54231 make run-data"
	@echo "  GOFFXI_VIEW_PORT=54001 make run-view"

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
	@echo "  make run-auth       - Build and run the auth server"
	@echo "  make run-data       - Build and run the data server"
	@echo "  make run-view       - Build and run the view server"
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
