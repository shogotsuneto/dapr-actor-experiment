#!/bin/bash

echo "=== Cleaning up Docker-based Dapr Demo ==="
echo ""

echo "Stopping containers..."
docker stop actor-service dapr-sidecar redis-dapr 2>/dev/null || true

echo "Removing containers..."
docker rm actor-service dapr-sidecar redis-dapr 2>/dev/null || true

echo "Removing network..."
docker network rm dapr-demo 2>/dev/null || true

echo "Removing image..."
docker rmi dapr-actor-demo 2>/dev/null || true

echo ""
echo "✓ Cleanup complete!"