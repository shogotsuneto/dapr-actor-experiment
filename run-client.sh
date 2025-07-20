#!/bin/bash

echo "Running Dapr Actor Demo Client..."

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Actor service not running. Please run './run-server.sh' first."
    exit 1
fi

# Run the client
echo "Starting client demo..."
docker compose --profile client up --build client

echo "Client demo completed!"