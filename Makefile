.PHONY: build clean test run-server run-client help

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

# Run the server locally
run-server: build
	@echo "Starting server..."
	@dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \
		--components-path ./configs/dapr --config ./configs/dapr/config.yaml -- ./bin/server

# Run the client demo
run-client: build
	@echo "Starting client..."
	@dapr run --app-id client --dapr-http-port 3501 -- ./bin/client

# Display help
help:
	@echo "Available targets:"
	@echo "  build       - Build server and client binaries"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  run-server  - Build and run server with Dapr"
	@echo "  run-client  - Build and run client with Dapr"
	@echo "  help        - Show this help message"