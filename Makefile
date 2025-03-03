# Variables
BINARY_NAME=comma
GOBIN=$(shell go env GOPATH)/bin
PACKAGE=github.com/username/comma
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_FLAGS=-ldflags "-X main.version=$(VERSION)" -trimpath
COVERAGE_DIR=./coverage
GO_FILES=$(shell find . -name '*.go' -not -path './vendor/*')
GO_PACKAGES=$(shell go list ./... | grep -v /vendor/)

# Default target
.PHONY: default
default: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(BUILD_FLAGS) -o $(BINARY_NAME) .

# Run the application in development mode
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	@go run .

# Run specific command
.PHONY: run-cmd
run-cmd:
	@go run . $(CMD)

# Install the application locally
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_NAME) $(GOBIN)/

# Clean up build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(COVERAGE_DIR)
	@rm -f coverage.out

# Run all tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -race -coverprofile=coverage.out ./...

# Run tests with verbose output
.PHONY: test-verbose
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -race -v -coverprofile=coverage.out ./...

# Show test coverage
.PHONY: coverage
coverage: test
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	@go tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=coverage.out

# Open test coverage in browser
.PHONY: coverage-view
coverage-view: coverage
	@echo "Opening coverage report in browser..."
	@open $(COVERAGE_DIR)/coverage.html

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, please install it: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.51.2; \
	fi

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)

# Verify that the code is properly formatted
.PHONY: fmt-check
fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -s -l $(GO_FILES))" ]; then \
		echo "The following files are not properly formatted:"; \
		gofmt -s -l $(GO_FILES); \
		exit 1; \
	fi

# Check for suspicious constructs
.PHONY: vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run all checks
.PHONY: check
check: fmt-check vet lint test

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

# Setup git hook for development
.PHONY: setup-dev-hook
setup-dev-hook: build
	@echo "Setting up git hook for development..."
	@cp $(BINARY_NAME) $(GOBIN)/
	@$(BINARY_NAME) install-hook
	@echo "Git hook installed successfully!"

# Run with specific command examples
.PHONY: run-generate
run-generate:
	@go run . generate --verbose

.PHONY: run-tui
run-tui:
	@go run . tui

.PHONY: run-setup
run-setup:
	@go run . setup

.PHONY: run-analyze
run-analyze:
	@go run . analyze

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  run             - Run the application in development mode"
	@echo "  run-cmd CMD=xx  - Run a specific command (e.g., make run-cmd CMD='generate --verbose')"
	@echo "  run-generate    - Run the generate command with verbose flag"
	@echo "  run-tui         - Run the interactive terminal UI"
	@echo "  run-setup       - Run the configuration UI"
	@echo "  run-analyze     - Run the repository analysis"
	@echo "  install         - Install the application locally"
	@echo "  setup-dev-hook  - Install the application and set up the git hook"
	@echo "  clean           - Clean up build artifacts"
	@echo "  test            - Run all tests"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  coverage        - Show test coverage"
	@echo "  coverage-view   - Open test coverage in browser"
	@echo "  lint            - Run linter"
	@echo "  deps            - Install dependencies"
	@echo "  fmt             - Format code"
	@echo "  fmt-check       - Verify that the code is properly formatted"
	@echo "  vet             - Check for suspicious constructs"
	@echo "  check           - Run all checks (fmt-check, vet, lint, test)"
	@echo "  release         - Create a new release"
	@echo "  help            - Show this help message"