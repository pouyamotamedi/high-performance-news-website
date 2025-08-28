# High Performance News Website Makefile

.PHONY: help build run test lint fmt vet clean docker-up docker-down migrate-up migrate-down deps

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build the application
build: ## Build the application binary
	@echo "Building application..."
	go build -o bin/server cmd/server/main.go

# Run the application
run: ## Run the application
	@echo "Starting application..."
	go run cmd/server/main.go

# Run tests
test: ## Run all tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test ## Run tests and show coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Vet code
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Clean build artifacts
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker commands
docker-up: ## Start Docker services
	@echo "Starting Docker services..."
	docker-compose up -d

docker-up-windows: ## Start Docker services (Windows optimized)
	@echo "Starting Docker services (Windows)..."
	docker-compose -f docker-compose.windows.yml up -d

docker-down: ## Stop Docker services
	@echo "Stopping Docker services..."
	docker-compose down

docker-clean: ## Clean up Docker containers and volumes
	@echo "Cleaning Docker containers and volumes..."
	docker-compose down -v
	docker system prune -f

docker-logs: ## Show Docker logs
	docker-compose logs -f

# Database migrations
migrate-up: ## Run database migrations up
	@echo "Running migrations up..."
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/news_website?sslmode=disable" up

migrate-down: ## Run database migrations down
	@echo "Running migrations down..."
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/news_website?sslmode=disable" down

migrate-create: ## Create a new migration file (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=migration_name"; exit 1; fi
	migrate create -ext sql -dir migrations $(NAME)

# Install dependencies
deps: ## Install/update dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Install development tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/air-verse/air@latest

# Development workflow
dev-setup: install-tools deps docker-up ## Set up development environment
	@echo "Development environment setup complete!"
	@echo "Run 'make dev' to start development server with hot reload"

dev: ## Start development server with hot reload
	air

dev-windows: ## Start development server with hot reload (Windows)
	air -c .air.windows.toml

dev-simple: ## Start development server without hot reload
	go run cmd/server/main.go

# Production build
build-prod: ## Build for production
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o bin/server cmd/server/main.go

# Quality checks
quality: fmt vet lint test ## Run all quality checks

# CI pipeline
ci: deps quality build ## Run CI pipeline