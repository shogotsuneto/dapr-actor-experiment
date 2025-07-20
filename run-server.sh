#!/bin/bash

echo "Building and starting Dapr Actor Demo..."

# Build and start the actor service with Redis
echo "Starting actor service and Redis..."
docker compose up -d --build redis actor-service actor-service-dapr

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Check service health
echo "Checking service health..."
curl -f http://localhost:3500/v1.0/healthz || {
    echo "Dapr sidecar not ready"
    exit 1
}

curl -f http://localhost:8080/health || {
    echo "Actor service not ready"
    exit 1
}

echo "Services are ready!"
echo "Actor service is running on http://localhost:8080"
echo "Dapr sidecar is running on http://localhost:3500"
echo ""
echo "You can test the actor with:"
echo "  curl http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/get"
echo ""
echo "Or run the client demo with:"
echo "  ./run-client.sh"
echo ""
echo "To stop services:"
echo "  docker compose down"