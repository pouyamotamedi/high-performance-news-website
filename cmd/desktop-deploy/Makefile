# Makefile for Desktop Deployment Application

APP_NAME := news-deploy
VERSION := 1.0.0
BUILD_DIR := dist
GO_FILES := $(shell find . -name '*.go' -not -path './dist/*')

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build:
	@echo "Building $(APP_NAME) for current platform..."
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "Building $(APP_NAME) v$(VERSION) for all platforms..."
	go run build.go build

# Build for specific platforms
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-windows-amd64/$(APP_NAME).exe .

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-amd64/$(APP_NAME) .

.PHONY: build-macos
build-macos:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64/$(APP_NAME) .

.PHONY: build-macos-arm
build-macos-arm:
	@echo "Building for macOS ARM64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(VERSION)" \
		-o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64/$(APP_NAME) .

# Development targets
.PHONY: dev
dev:
	@echo "Starting development server..."
	go run . --dev --no-browser

.PHONY: run
run:
	@echo "Starting $(APP_NAME)..."
	go run .

# Testing
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: benchmark
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Code quality
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Packaging
.PHONY: package
package: build-all
	@echo "Creating packages..."
	cd $(BUILD_DIR) && \
	for dir in */; do \
		if [ -d "$$dir" ]; then \
			echo "Packaging $$dir..."; \
			tar -czf "$${dir%/}.tar.gz" "$$dir"; \
		fi \
	done
	@echo "Packages created in $(BUILD_DIR)/"

.PHONY: package-zip
package-zip: build-all
	@echo "Creating ZIP packages..."
	cd $(BUILD_DIR) && \
	for dir in */; do \
		if [ -d "$$dir" ]; then \
			echo "Creating ZIP for $$dir..."; \
			zip -r "$${dir%/}.zip" "$$dir"; \
		fi \
	done
	@echo "ZIP packages created in $(BUILD_DIR)/"

# Docker (for containerized builds)
.PHONY: docker-build
docker-build:
	@echo "Building with Docker..."
	docker build -t $(APP_NAME):$(VERSION) .

# Installation
.PHONY: install
install: build
	@echo "Installing $(APP_NAME)..."
	sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "$(APP_NAME) installed to /usr/local/bin/"

.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(APP_NAME)..."
	sudo rm -f /usr/local/bin/$(APP_NAME)
	@echo "$(APP_NAME) uninstalled"

# Cleanup
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

.PHONY: clean-all
clean-all: clean
	@echo "Cleaning all artifacts including dependencies..."
	go clean -modcache
	@echo "All clean"

# Release
.PHONY: release
release: clean test lint build-all package
	@echo "Release $(VERSION) ready in $(BUILD_DIR)/"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  build-all      - Build for all platforms"
	@echo "  build-windows  - Build for Windows"
	@echo "  build-linux    - Build for Linux"
	@echo "  build-macos    - Build for macOS Intel"
	@echo "  build-macos-arm- Build for macOS ARM64"
	@echo "  dev            - Start development server"
	@echo "  run            - Run the application"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  benchmark      - Run benchmarks"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  vet            - Run go vet"
	@echo "  deps           - Download dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  package        - Create tar.gz packages"
	@echo "  package-zip    - Create ZIP packages"
	@echo "  install        - Install to /usr/local/bin"
	@echo "  uninstall      - Remove from /usr/local/bin"
	@echo "  clean          - Clean build artifacts"
	@echo "  clean-all      - Clean all artifacts"
	@echo "  release        - Create release build"
	@echo "  help           - Show this help"

# Variables for debugging
.PHONY: vars
vars:
	@echo "APP_NAME: $(APP_NAME)"
	@echo "VERSION: $(VERSION)"
	@echo "BUILD_DIR: $(BUILD_DIR)"
	@echo "GO_FILES: $(GO_FILES)"