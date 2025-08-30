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

# Comprehensive testing commands
test-unit: ## Run unit tests with 95% coverage requirement
	@echo "Running unit tests with coverage tracking..."
	@mkdir -p test-results
	CGO_ENABLED=1 go test -v -race -coverprofile=test-results/coverage.out \
		-covermode=atomic -timeout=30m \
		./internal/models/... ./internal/repositories/... ./internal/services/... \
		./internal/api/... ./internal/auth/... ./internal/validation/... ./pkg/...
	@go tool cover -func=test-results/coverage.out | grep total | awk '{print "Coverage: " $$3}'
	@go tool cover -html=test-results/coverage.out -o test-results/coverage.html
	@echo "Unit test coverage report: test-results/coverage.html"

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@mkdir -p test-results
	CGO_ENABLED=1 go test -v -tags=integration -timeout=30m \
		./internal/integration/... ./internal/repositories/...

test-benchmark: ## Run performance benchmarks
	@echo "Running performance benchmarks..."
	@mkdir -p test-results
	go test -bench=. -benchmem -benchtime=10s -timeout=10m \
		./internal/models/... ./internal/repositories/... ./internal/services/... ./pkg/... \
		| tee test-results/benchmark.txt

test-comprehensive: ## Run comprehensive test suite with all validations
	@echo "Running comprehensive test suite..."
	@mkdir -p test-results
	@$(MAKE) test-unit
	@$(MAKE) test-integration
	@$(MAKE) test-benchmark
	@echo "Comprehensive testing complete. Results in test-results/"

test-coverage-check: test-unit ## Validate coverage meets 95% requirement
	@echo "Validating coverage requirements..."
	@COVERAGE=$$(go tool cover -func=test-results/coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < 95" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below required 95%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets requirement (≥95%)"; \
	fi

test-parallel: ## Run tests in parallel with optimal performance
	@echo "Running tests in parallel..."
	@mkdir -p test-results
	CGO_ENABLED=1 go test -v -race -parallel=$$(nproc) -coverprofile=test-results/coverage.out \
		-covermode=atomic -timeout=30m ./...

test-clean: ## Clean test artifacts and results
	@echo "Cleaning test artifacts..."
	rm -rf test-results/
	rm -f coverage.out coverage.html
	rm -f *.test
	rm -f *.prof

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

# AI Code Validation
ai-validate: ## Run AI code validation on all Go files
	@echo "Running AI code validation..."
	go run cmd/ai-validator/main.go -dir . -format text -severity medium

ai-validate-critical: ## Run AI code validation for critical issues only
	@echo "Running AI code validation (critical issues only)..."
	go run cmd/ai-validator/main.go -dir . -format text -severity critical

ai-validate-business: ## Run business logic validation
	@echo "Running business logic validation..."
	go run cmd/ai-validator/main.go -dir . -format text -business

ai-validate-json: ## Run AI code validation with JSON output
	@echo "Running AI code validation (JSON output)..."
	go run cmd/ai-validator/main.go -dir . -format json > reports/ai-validation.json

ai-validate-file: ## Run AI code validation on specific file (usage: make ai-validate-file FILE=path/to/file.go)
	@if [ -z "$(FILE)" ]; then echo "Usage: make ai-validate-file FILE=path/to/file.go"; exit 1; fi
	go run cmd/ai-validator/main.go -file $(FILE) -format text -verbose

build-ai-validator: ## Build AI validator binary
	@echo "Building AI validator..."
	go build -o bin/ai-validator cmd/ai-validator/main.go

# CI pipeline
ci: deps quality ai-validate build ## Run CI pipeline with AI validation