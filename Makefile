.PHONY: build clean test test-unit test-integration test-integration-quick test-integration-docker test-integration-kind generate generate-install generate-clean help

# Default target
all: build

# Build all binaries
build:
	@echo "Building server..."
	@mkdir -p bin
	@go build -o bin/server ./cmd/server
	@echo "Building client..."
	@go build -o bin/client ./cmd/client
	@echo "Building projector..."
	@go build -o bin/projector ./cmd/projector

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/

# Generate actor code from OpenAPI schema
generate:
	@echo "Generating actor code from OpenAPI schema..."
	@echo "Schema file: schemas/openapi/multi-actors.yaml"
	@echo "Output directory: internal/"
	@docker run --rm -u root \
		-v "$(PWD)/schemas/openapi/multi-actors.yaml:/input.yaml" \
		-v "$(PWD)/internal:/output" \
		ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.5 \
		/input.yaml /output
	@echo "✓ Actor code generation completed successfully!"

# Install code generation tools
generate-install:
	@echo "Installing code generation tools..."
	@echo "Pulling external generator Docker image..."
	@docker pull ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.5
	@echo "✓ Installation complete!"

# Clean generated code
generate-clean:
	@echo "Cleaning generated actor code..."
	@rm -rf internal/counter/types.go internal/counter/api.go internal/counter/factory.go
	@rm -rf internal/bankaccount/types.go internal/bankaccount/api.go internal/bankaccount/factory.go
	@echo "✓ Generated code cleaned (implementation files preserved)"

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

# Run integration tests inside Docker container
test-integration-docker:
	@echo "Running integration tests inside Docker container..."
	@echo "Starting test services with Docker Compose..."
	@docker compose -f test/integration/docker-compose.test.yml up -d --build
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Running tests in Docker container..."
	@docker compose -f test/integration/docker-compose.test.yml --profile test-runner run --rm test-runner || (echo "Tests failed, stopping services..." && docker compose -f test/integration/docker-compose.test.yml down && exit 1)
	@echo "Stopping test services..."
	@docker compose -f test/integration/docker-compose.test.yml down

# Run integration tests against Kind cluster
test-integration-kind:
	@echo "Running integration tests against Kind cluster..."
	@echo "Running integration tests via NodePort services..."
	@DAPR_HTTP_ENDPOINT="http://localhost:3500" \
	 go test -count=1 -v ./test/integration/... -timeout=5m
	@echo "Integration tests completed successfully!"

# Display help
help:
	@echo "Available targets:"
	@echo "  build                     - Build server and client binaries"
	@echo "  clean                     - Remove build artifacts"
	@echo "  generate                  - Generate actor code from OpenAPI schema"
	@echo "  generate-install          - Install code generation tools (Docker-based)"
	@echo "  generate-clean            - Clean generated actor code (preserves implementations)"
	@echo "  test                      - Run all tests (unit + integration)"
	@echo "  test-unit                 - Run unit tests only"
	@echo "  test-integration          - Run integration tests (starts/stops Docker services)"
	@echo "  test-integration-quick    - Run integration tests (assumes services running)"
	@echo "  test-integration-docker   - Run integration tests inside Docker container"
	@echo "  test-integration-kind     - Run integration tests against Kind cluster"
	@echo "  help                      - Show this help message"

