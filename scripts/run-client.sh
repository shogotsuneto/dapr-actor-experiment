#!/bin/bash

echo "Running Dapr Actor Demo Client..."

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Actor service not running. Please run './scripts/run-server.sh' first."
    exit 1
fi

# Build client if needed
if [ ! -f "./bin/client" ]; then
    echo "Building client..."
    go build -o bin/client ./cmd/client
fi

# Run the client using Dapr
echo "Starting client demo..."
dapr run --app-id client --dapr-http-port 3501 -- ./bin/client

echo "Client demo completed!"