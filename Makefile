.PHONY: build clean test help

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

# Test the application
test:
	@echo "Running tests..."
	@go test -v ./...

# Display help
help:
	@echo "Available targets:"
	@echo "  build       - Build server and client binaries"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  help        - Show this help message"