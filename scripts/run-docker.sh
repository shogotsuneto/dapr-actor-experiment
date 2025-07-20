#!/bin/bash

echo "=== Docker-based Dapr Actor Demo (No CLI Required) ==="
echo ""
echo "This script runs the Dapr actor demo using Docker containers only."
echo "No Dapr CLI installation required!"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

echo "✓ Docker found"

# Create a Docker network for our services
echo "Creating Docker network..."
docker network create dapr-demo 2>/dev/null || echo "✓ Network already exists"

# Start Redis
echo "Starting Redis..."
docker run -d --name redis-dapr --network dapr-demo -p 6379:6379 redis:7-alpine 2>/dev/null || {
    docker start redis-dapr 2>/dev/null || echo "✓ Redis already running"
}

# Build the Go application
echo "Building Go application..."
docker build -t dapr-actor-demo .

# Start Dapr sidecar using Docker
echo "Starting Dapr sidecar..."
docker run -d \
    --name dapr-sidecar \
    --network dapr-demo \
    -p 3500:3500 \
    -p 3501:3501 \
    -v "$(pwd)/configs/dapr:/components" \
    daprio/daprd:latest \
    ./daprd \
    --app-id actor-service \
    --app-port 8080 \
    --dapr-http-port 3500 \
    --dapr-grpc-port 3501 \
    --components-path /components \
    --log-level info

# Wait for sidecar to be ready
echo "Waiting for Dapr sidecar to be ready..."
sleep 5

# Start the Go application
echo "Starting actor service..."
docker run -d \
    --name actor-service \
    --network dapr-demo \
    -p 8080:8080 \
    dapr-actor-demo

echo ""
echo "🚀 Services started successfully!"
echo ""
echo "You can now test the actor service:"
echo ""
echo "1. Get counter value:"
echo "   curl http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/get"
echo ""
echo "2. Increment counter:"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/increment"
echo ""
echo "3. Set counter value:"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/set \\"
echo "        -H 'Content-Type: application/json' -d '{\"value\": 42}'"
echo ""
echo "4. Run comprehensive tests:"
echo "   ./scripts/test-actor.sh"
echo ""
echo "To stop all services:"
echo "   docker stop actor-service dapr-sidecar redis-dapr"
echo "   docker rm actor-service dapr-sidecar redis-dapr"
echo "   docker network rm dapr-demo"