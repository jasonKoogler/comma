# Project variables
APP_NAME := comma
MAIN_PATH := ./main.go
BIN_DIR := ./bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS := -ldflags="-s -w -X main.version=$(VERSION)"
PACKAGES := $(shell go list ./... | grep -v /vendor/)

# Go commands
GO := go
GOBUILD := $(GO) build
GOTEST := $(GO) test
GOLINT := golangci-lint

# Determine OS and Architecture
ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(APP_NAME).exe
	PLATFORM := windows
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		BINARY_NAME := $(APP_NAME)
		PLATFORM := linux
	endif
	ifeq ($(UNAME_S),Darwin)
		BINARY_NAME := $(APP_NAME)
		PLATFORM := macos
	endif
endif

# Default target
.PHONY: all
all: clean build

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@$(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Cross-compile for multiple platforms
.PHONY: build-all
build-all: build-linux build-windows build-macos

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-$(VERSION)-linux-amd64 $(MAIN_PATH)
	@echo "Linux build complete"

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BIN_DIR)
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-$(VERSION)-windows-amd64.exe $(MAIN_PATH)
	@echo "Windows build complete"

.PHONY: build-macos
build-macos:
	@echo "Building for macOS..."
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BIN_DIR)/$(APP_NAME)-$(VERSION)-macos-amd64 $(MAIN_PATH)
	@echo "macOS build complete"

# Install the application locally
.PHONY: install
install:
	@echo "Installing $(APP_NAME)..."
	@$(GO) install $(BUILD_FLAGS) $(MAIN_PATH)
	@echo "Installation complete"

# Run the application in CLI mode
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	@$(BIN_DIR)/$(BINARY_NAME) 2>&1

# Run a specific CLI command
.PHONY: run-cmd
run-cmd: build
	@echo "Running $(APP_NAME) $(CMD)..."
	@$(BIN_DIR)/$(BINARY_NAME) $(CMD) $(ARGS)

# Run the TUI interface
.PHONY: run-tui
run-tui: build
	@echo "Running $(APP_NAME) TUI..."
	@$(BIN_DIR)/$(BINARY_NAME) tui

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@$(GOTEST) -v $(PACKAGES)

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -coverprofile=coverage.out $(PACKAGES)
	@$(GO) tool cover -html=coverage.out

# Run linter
.PHONY: lint
lint:
	@echo "Linting code..."
	@$(GOLINT) run ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

# Setup development environment
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@go mod tidy
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development setup complete"

# Generate documentation
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@go doc -all > DOCUMENTATION.md
	@echo "Documentation generated"

# Create a new release
.PHONY: release
release:
	@echo "Creating release $(VERSION)..."
	@if [ -z "$(VERSION)" ]; then \
		echo "No version specified. Use: make release VERSION=v1.0.0"; \
		exit 1; \
	fi
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)
	@echo "Building release binaries for $(VERSION)..."
	@make build-all VERSION=$(VERSION)
	@echo "Release $(VERSION) created and built successfully!"

# Show help
.PHONY: help
help:
	@echo "Comma Makefile Help"
	@echo "---------------"
	@echo "make                 - Build the application"
	@echo "make build-all       - Build for Linux, Windows, and macOS"
	@echo "make run             - Run the application"
	@echo "make run-cmd CMD=... - Run a specific command (e.g., make run-cmd CMD=generate)"
	@echo "make run-tui         - Run the TUI interface"
	@echo "make test            - Run tests"
	@echo "make lint            - Run linter"
	@echo "make clean           - Remove build artifacts"
	@echo "make install         - Install locally"
	@echo "make release VERSION=v1.0.0 - Create a new versioned release"
	@echo "make help            - Show this help message"