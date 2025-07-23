.PHONY: build clean test test-unit test-integration help

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
	@echo "Note: This will start Docker Compose services and may take several minutes"
	@go test -v ./test/integration/... -timeout=10m

# Display help
help:
	@echo "Available targets:"
	@echo "  build           - Build server and client binaries"
	@echo "  clean           - Remove build artifacts"
	@echo "  test            - Run all tests (unit + integration)"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration - Run integration tests (requires Docker)"
	@echo "  help            - Show this help message"