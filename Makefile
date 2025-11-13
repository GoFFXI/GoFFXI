# Variables
GO := go
GOFLAGS := -v
BUILD_DIR := ./bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")

# Build flags with version information
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# Binary names
MIGRATIONS_BIN := migrations
LOBBY_AUTH_BIN := lobby-auth
LOBBY_DATA_BIN := lobby-data
LOBBY_VIEW_BIN := lobby-view
MAP_ROUTER_BIN := map-router
MAP_INSTANCE_BIN := map-instance

# Default target
.PHONY: all
all: build-migrations build-lobby-auth build-lobby-data build-lobby-view build-map-router build-map-instance

# Ensure build directory exists
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Build the binaries
.PHONY: build-migrations
build-migrations: $(BUILD_DIR)
	@echo "Building migrations..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN) ./cmd/migrations
	@echo "Build complete: $(BUILD_DIR)/$(MIGRATIONS_BIN)"

.PHONY: build-lobby-auth
build-lobby-auth: $(BUILD_DIR)
	@echo "Building lobby-auth..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN) ./cmd/lobby-auth
	@echo "Build complete: $(BUILD_DIR)/$(LOBBY_AUTH_BIN)"

.PHONY: build-lobby-data
build-lobby-data: $(BUILD_DIR)
	@echo "Building lobby-data..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN) ./cmd/lobby-data
	@echo "Build complete: $(BUILD_DIR)/$(LOBBY_DATA_BIN)"

.PHONY: build-lobby-view
build-lobby-view: $(BUILD_DIR)
	@echo "Building lobby-view..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN) ./cmd/lobby-view
	@echo "Build complete: $(BUILD_DIR)/$(LOBBY_VIEW_BIN)"

.PHONY: build-map-router
build-map-router: $(BUILD_DIR)
	@echo "Building map-router..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN) ./cmd/map-router
	@echo "Build complete: $(BUILD_DIR)/$(MAP_ROUTER_BIN)"

.PHONY: build-map-instance
build-map-instance: $(BUILD_DIR)
	@echo "Building map-instance..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN) ./cmd/map-instance
	@echo "Build complete: $(BUILD_DIR)/$(MAP_INSTANCE_BIN)"

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux: $(BUILD_DIR)
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN)-linux-amd64 ./cmd/lobby-auth
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN)-linux-arm64 ./cmd/lobby-auth
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN)-linux-amd64 ./cmd/lobby-data
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN)-linux-arm64 ./cmd/lobby-data
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN)-linux-amd64 ./cmd/lobby-view
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN)-linux-arm64 ./cmd/lobby-view
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN)-linux-amd64 ./cmd/migrations
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN)-linux-arm64 ./cmd/migrations
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN)-linux-amd64 ./cmd/map-router
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN)-linux-arm64 ./cmd/map-router
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN)-linux-amd64 ./cmd/map-instance
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN)-linux-arm64 ./cmd/map-instance
	@echo "Linux build complete"

.PHONY: build-darwin
build-darwin: $(BUILD_DIR)
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN)-darwin-amd64 ./cmd/lobby-auth
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN)-darwin-arm64 ./cmd/lobby-auth
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN)-darwin-amd64 ./cmd/lobby-data
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN)-darwin-arm64 ./cmd/lobby-data
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN)-darwin-amd64 ./cmd/lobby-view
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN)-darwin-arm64 ./cmd/lobby-view
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN)-darwin-amd64 ./cmd/migrations
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN)-darwin-arm64 ./cmd/migrations
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN)-darwin-amd64 ./cmd/map-router
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN)-darwin-arm64 ./cmd/map-router
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN)-darwin-amd64 ./cmd/map-instance
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN)-darwin-arm64 ./cmd/map-instance
	@echo "macOS build complete"

.PHONY: build-windows
build-windows: $(BUILD_DIR)
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_AUTH_BIN)-windows-amd64.exe ./cmd/lobby-auth
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_DATA_BIN)-windows-amd64.exe ./cmd/lobby-data
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(LOBBY_VIEW_BIN)-windows-amd64.exe ./cmd/lobby-view
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MIGRATIONS_BIN)-windows-amd64.exe ./cmd/migrations
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_ROUTER_BIN)-windows-amd64.exe ./cmd/map-router
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(MAP_INSTANCE_BIN)-windows-amd64.exe ./cmd/map-instance
	@echo "Windows build complete"

# Run targets
.PHONY: run-migrations
run-migrations: build-migrations
	@echo "Running migrations..."
	$(BUILD_DIR)/$(MIGRATIONS_BIN)

.PHONY: run-lobby-auth
run-lobby-auth: build-lobby-auth
	@echo "Starting lobby auth server..."
	SERVER_PORT=54231 NATS_CLIENT_PREFIX="dev-lobby-auth-" $(BUILD_DIR)/$(LOBBY_AUTH_BIN)

.PHONY: run-lobby-data
run-lobby-data: build-lobby-data
	@echo "Starting lobby data server..."
	SERVER_PORT=54230 NATS_CLIENT_PREFIX="dev-lobby-data-" $(BUILD_DIR)/$(LOBBY_DATA_BIN)

.PHONY: run-lobby-view
run-lobby-view: build-lobby-view
	@echo "Starting lobby view server..."
	SERVER_PORT=54001 NATS_CLIENT_PREFIX="dev-lobby-view-" $(BUILD_DIR)/$(LOBBY_VIEW_BIN)

.PHONY: run-map-router
run-map-router: build-map-router
	@echo "Starting map router server..."
	SERVER_PORT=54230 NATS_CLIENT_PREFIX="dev-map-router-" $(BUILD_DIR)/$(MAP_ROUTER_BIN)

.PHONY: run-map-instance
run-map-instance: build-map-instance
	@echo "Starting map instance server..."
	NATS_CLIENT_PREFIX="dev-map-instance-" $(BUILD_DIR)/$(MAP_INSTANCE_BIN)

# Run all services (requires tmux or separate terminals)
.PHONY: run-all
run-all:
	@if command -v tmux >/dev/null 2>&1; then \
		echo "Starting all services in tmux..."; \
		tmux new-session -d -s lobby -n auth "make run-lobby-auth"; \
		tmux new-window -t lobby -n data "make run-lobby-data"; \
		tmux new-window -t lobby -n view "make run-lobby-view"; \
		tmux new-window -t lobby -n map-router "make run-map-router"; \
		tmux new-window -t lobby -n map-instance "make run-map-instance"; \
		tmux attach -t lobby; \
	else \
		echo "tmux not found. Please run the following commands in separate terminals:"; \
		echo "  Terminal 1: make run-lobby-auth"; \
		echo "  Terminal 2: make run-lobby-data"; \
		echo "  Terminal 3: make run-lobby-view"; \
		echo "  Terminal 4: make run-map-router"; \
		echo "  Terminal 5: make run-map-instance"; \
	fi

# Test targets
.PHONY: test
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: test-short
test-short:
	@echo "Running short tests..."
	$(GO) test -v -short ./...

# Code quality targets
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Code formatted"

.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed."; \
		echo "Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
		exit 1; \
	fi

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

.PHONY: check
check: fmt vet lint test
	@echo "All checks passed!"

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "Dependencies updated"

.PHONY: update
update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated to latest versions"

# Clean targets
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all generated files..."
	@$(GO) clean -cache
	@echo "Deep clean complete"

# Install/Uninstall targets
.PHONY: install
install: build-migrations build-lobby-auth build-lobby-data build-lobby-view build-map-router
	@echo "Installing binaries to GOPATH/bin..."
	@cp $(BUILD_DIR)/$(MIGRATIONS_BIN) $$(go env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(LOBBY_AUTH_BIN) $$(go env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(LOBBY_DATA_BIN) $$(go env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(LOBBY_VIEW_BIN) $$(go env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(MAP_ROUTER_BIN) $$(go env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(MAP_INSTANCE_BIN) $$(go env GOPATH)/bin/
	@echo "Installation complete"

.PHONY: uninstall
uninstall:
	@echo "Uninstalling binaries from GOPATH/bin..."
	@rm -f $$(go env GOPATH)/bin/$(MIGRATIONS_BIN)
	@rm -f $$(go env GOPATH)/bin/$(LOBBY_AUTH_BIN)
	@rm -f $$(go env GOPATH)/bin/$(LOBBY_DATA_BIN)
	@rm -f $$(go env GOPATH)/bin/$(LOBBY_VIEW_BIN)
	@rm -f $$(go env GOPATH)/bin/$(MAP_ROUTER_BIN)
	@rm -f $$(go env GOPATH)/bin/$(MAP_INSTANCE_BIN)
	@echo "Uninstall complete"

# Development helpers
.PHONY: dev
dev:
	@echo "Development Setup Instructions:"
	@echo "================================"
	@echo "Option 1: Run with tmux (recommended)"
	@echo "  make run-all"
	@echo ""
	@echo "Option 2: Run in separate terminals"
	@echo "  Terminal 1: make run-lobby-auth"
	@echo "  Terminal 2: make run-lobby-data"
	@echo "  Terminal 3: make run-lobby-view"
	@echo "  Terminal 4: make run-map-router"
	@echo "  Terminal 5: make run-map-instance"
	@echo ""
	@echo "Option 3: Run migrations first if needed"
	@echo "  make run-migrations"

.PHONY: watch
watch:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		exit 1; \
	fi

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	docker build -f docker/lobby-auth.Dockerfile -t lobby-auth:latest .
	docker build -f docker/lobby-data.Dockerfile -t lobby-data:latest .
	docker build -f docker/lobby-view.Dockerfile -t lobby-view:latest .
	docker build -f docker/map-router.Dockerfile -t map-router:latest .
	docker build -f docker/map-instance.Dockerfile -t map-instance:latest .
	@echo "Docker images built"

# Version and help
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $$(go version)"

.PHONY: help
help:
	@echo "Lobby Services Makefile"
	@echo "======================="
	@echo ""
	@echo "Build targets:"
	@echo "  make                    - Build all binaries for current platform"
	@echo "  make build-migrations   - Build the migrations binary"
	@echo "  make build-lobby-auth   - Build the lobby auth server binary"
	@echo "  make build-lobby-data   - Build the lobby data server binary"
	@echo "  make build-lobby-view   - Build the lobby view server binary"
	@echo "  make build-map-router   - Build the map router server binary"
	@echo "  make build-map-instance - Build the map instance server binary"
	@echo "  make build-all          - Build for Linux, macOS, and Windows"
	@echo "  make build-linux        - Build for Linux (amd64, arm64)"
	@echo "  make build-darwin       - Build for macOS (amd64, arm64)"
	@echo "  make build-windows      - Build for Windows (amd64)"
	@echo ""
	@echo "Run targets:"
	@echo "  make run-migrations     - Build and run database migrations"
	@echo "  make run-lobby-auth     - Build and run the auth server"
	@echo "  make run-lobby-data     - Build and run the data server"
	@echo "  make run-lobby-view     - Build and run the view server"
	@echo "  make run-map-router     - Build and run the map router server"
	@echo "  make run-all            - Run all services (requires tmux)"
	@echo ""
	@echo "Test targets:"
	@echo "  make test               - Run all tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-short         - Run short tests only"
	@echo ""
	@echo "Code quality:"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter (requires golangci-lint)"
	@echo "  make vet                - Run go vet"
	@echo "  make check              - Run all quality checks"
	@echo ""
	@echo "Dependency management:"
	@echo "  make deps               - Download and tidy dependencies"
	@echo "  make update             - Update dependencies to latest versions"
	@echo ""
	@echo "Clean targets:"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make clean-all          - Deep clean including Go cache"
	@echo ""
	@echo "Install targets:"
	@echo "  make install            - Install binaries to GOPATH/bin"
	@echo "  make uninstall          - Remove binaries from GOPATH/bin"
	@echo ""
	@echo "Development:"
	@echo "  make dev                - Show development setup instructions"
	@echo "  make watch              - Run with hot reload (requires air)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build       - Build Docker images"
	@echo ""
	@echo "Other:"
	@echo "  make version            - Show version information"
	@echo "  make help               - Show this help message"

# Default shell
SHELL := /bin/bash

# Prevent make from printing directory changes
MAKEFLAGS += --no-print-directory
