.PHONY: build clean test test-unit test-integration test-integration-quick help

# Default target
all: build

# Build all binaries
build:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/server ./cmd/server
	@echo "Building client..."
	@go build -o bin/client ./cmd/client

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -v -short ./...

# Run integration tests (requires Docker)
test-integration:
	@echo "Running integration tests..."
	@echo "Starting test services with Docker Compose..."
	@docker compose -f test/integration/docker-compose.test.yml up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Running tests..."
	@go test -v ./test/integration/... -timeout=5m || (echo "Tests failed, stopping services..." && docker compose -f test/integration/docker-compose.test.yml down && exit 1)
	@echo "Stopping test services..."
	@docker compose -f test/integration/docker-compose.test.yml down

# Run integration tests assuming services are already running
test-integration-quick:
	@echo "Running integration tests (assuming services are running)..."
	@echo "Make sure services are started with: docker compose -f test/integration/docker-compose.test.yml up -d"
	@go test -v ./test/integration/... -timeout=2m

# Display help
help:
	@echo "Available targets:"
	@echo "  build                   - Build server and client binaries"
	@echo "  clean                   - Remove build artifacts"
	@echo "  test                    - Run all tests (unit + integration)"
	@echo "  test-unit               - Run unit tests only"
	@echo "  test-integration        - Run integration tests (starts/stops Docker services)"
	@echo "  test-integration-quick  - Run integration tests (assumes services running)"
	@echo "  help                    - Show this help message"