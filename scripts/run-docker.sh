#!/bin/bash

echo "=== Docker-based Dapr Actor Demo (No CLI Required) ==="
echo ""
echo "This script runs the Dapr actor demo using Docker Compose."
echo "Using multi-stage Docker build to compile the Go binary inside containers."
echo ""

# Check if Docker and Docker Compose are available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker not found. Please install Docker first."
    exit 1
fi

if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose not found. Please install Docker Compose first."
    exit 1
fi

echo "✓ Docker and Docker Compose found"

# Build and start the services using Docker Compose
echo "Building and starting services with Docker Compose..."
docker compose up -d --build

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 15

# Check service health
echo "Checking service health..."

# Check if actor service health endpoint responds (no JWT required)
if curl -f http://localhost:8080/health &>/dev/null; then
    echo "✓ Actor service is ready"
else
    echo "❌ Actor service not ready, checking logs..."
    docker compose logs actor-service
    exit 1
fi

# Check if Dapr sidecar is responding (will return 401 due to JWT requirement, but means it's running)
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:3500/v1.0/healthz)
if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "200" ]; then
    echo "✓ Dapr sidecar is ready (JWT authentication active)"
else
    echo "❌ Dapr sidecar not ready (HTTP $HTTP_CODE), checking logs..."
    docker compose logs actor-service-dapr
    exit 1
fi

echo ""
echo "🚀 Services started successfully!"
echo ""
echo "You can now test the actor service:"
echo ""
echo "1. Run comprehensive tests:"
echo "   ./scripts/test-multi-actors.sh       # Test all actors"
echo "   ./scripts/test-counter-actor.sh      # Test CounterActor only"
echo "   ./scripts/test-bank-account-actor.sh # Test BankAccountActor only"
echo ""
echo "2. Get counter value:"
echo "   curl http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/Get"
echo ""
echo "3. Increment counter:"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/Increment"
echo ""
echo "4. Set counter value:"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/Set \\"
echo "        -H 'Content-Type: application/json' -d '{\"value\": 42}'"
echo ""
echo "To stop all services:"
echo "   docker compose down"