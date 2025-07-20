#!/bin/bash

echo "=== Cleaning up Docker Compose Dapr Demo ==="
echo ""

# Stop and remove all services
echo "Stopping and removing all services..."
docker compose down -v

# Remove any orphaned containers
echo "Removing orphaned containers..."
docker compose down --remove-orphans

# Clean up unused images (optional)
echo "Cleaning up unused Docker resources..."
docker system prune -f

echo ""
echo "✓ Cleanup complete!"