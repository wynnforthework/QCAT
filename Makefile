.PHONY: test test-unit test-integration test-e2e test-benchmark test-coverage clean build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=qcat
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
TEST_TIMEOUT=30m
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/qcat

# Build optimizer
build-optimizer:
	$(GOBUILD) -o optimizer -v ./cmd/optimizer

# Build config tool
build-config:
	$(GOBUILD) -o qcat-config -v ./cmd/config

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/server

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -timeout $(TEST_TIMEOUT) ./internal/...

# Run unit tests with coverage
test-coverage:
	@echo "Running unit tests with coverage..."
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -timeout $(TEST_TIMEOUT) ./internal/...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -tags=integration -timeout $(TEST_TIMEOUT) ./tests/integration/...

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	$(GOTEST) -v -tags=e2e -timeout $(TEST_TIMEOUT) ./tests/e2e/...

# Run benchmark tests
test-benchmark:
	@echo "Running benchmark tests..."
	$(GOTEST) -bench=. -benchmem -run=^$$ ./internal/...

# Run performance tests
test-performance:
	@echo "Running performance tests..."
	$(GOTEST) -v -tags=performance -timeout $(TEST_TIMEOUT) ./tests/performance/...

# Run all tests including E2E
test-all: test-unit test-integration test-e2e

# Run tests in short mode (skip long-running tests)
test-short:
	@echo "Running tests in short mode..."
	$(GOTEST) -short -v ./internal/...

# Run specific test
test-run:
	@echo "Running specific test: $(TEST)"
	$(GOTEST) -v -run $(TEST) ./...

# Generate test report
test-report: test-coverage
	@echo "Generating test report..."
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

# Lint code
lint:
	@echo "Running linters..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...
	govulncheck ./...

# Run all quality checks
quality: fmt vet lint security

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint
	$(GOGET) -u github.com/securecodewarrior/gosec/v2/cmd/gosec
	$(GOGET) -u golang.org/x/vuln/cmd/govulncheck

# Start development environment
dev-up:
	@echo "Starting development environment..."
	docker-compose -f docker-compose.dev.yml up -d

# Stop development environment
dev-down:
	@echo "Stopping development environment..."
	docker-compose -f docker-compose.dev.yml down

# Run database migrations
migrate-up:
	@echo "Running database migrations..."
	migrate -path ./internal/database/migrations -database "$(DATABASE_URL)" up

# Rollback database migrations
migrate-down:
	@echo "Rolling back database migrations..."
	migrate -path ./internal/database/migrations -database "$(DATABASE_URL)" down

# Generate mocks
generate-mocks:
	@echo "Generating mocks..."
	mockgen -source=internal/cache/cache.go -destination=internal/mocks/cache_mock.go
	mockgen -source=internal/database/database.go -destination=internal/mocks/database_mock.go

# Configuration management
config-validate:
	@echo "Validating configuration..."
	$(GOBUILD) -o qcat-config -v ./cmd/config
	./qcat-config -validate

config-generate:
	@echo "Generating environment template..."
	$(GOBUILD) -o qcat-config -v ./cmd/config
	./qcat-config -generate

config-encrypt:
	@echo "Encrypting string..."
	$(GOBUILD) -o qcat-config -v ./cmd/config
	./qcat-config -encrypt "$(TEXT)"

config-decrypt:
	@echo "Decrypting string..."
	$(GOBUILD) -o qcat-config -v ./cmd/config
	./qcat-config -decrypt "$(TEXT)"

# Start local development
start-local:
	@echo "Starting local development environment..."
	chmod +x scripts/start_local.sh
	./scripts/start_local.sh

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  build-linux     - Build for Linux"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Download dependencies"
	@echo "  test            - Run unit and integration tests"
	@echo "  test-unit       - Run unit tests"
	@echo "  test-coverage   - Run unit tests with coverage"
	@echo "  test-integration- Run integration tests"
	@echo "  test-e2e        - Run E2E tests"
	@echo "  test-benchmark  - Run benchmark tests"
	@echo "  test-all        - Run all tests"
	@echo "  test-short      - Run tests in short mode"
	@echo "  test-report     - Generate test report"
	@echo "  lint            - Run linters"
	@echo "  fmt             - Format code"
	@echo "  vet             - Vet code"
	@echo "  security        - Run security scan"
	@echo "  quality         - Run all quality checks"
	@echo "  install-tools   - Install development tools"
	@echo "  dev-up          - Start development environment"
	@echo "  dev-down        - Stop development environment"
	@echo "  migrate-up      - Run database migrations"
	@echo "  migrate-down    - Rollback database migrations"
	@echo "  generate-mocks  - Generate mocks"
	@echo "  config-validate - Validate configuration"
	@echo "  config-generate - Generate environment template"
	@echo "  config-encrypt  - Encrypt string (TEXT=string)"
	@echo "  config-decrypt  - Decrypt string (TEXT=string)"
	@echo "  start-local     - Start local development environment"
	@echo "  help            - Show this help"
