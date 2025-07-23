#!/bin/bash

echo "=== Cleaning up Docker Compose Dapr Demo ==="
echo ""

# Stop and remove all services, volumes, and orphaned containers
echo "Stopping and removing all services, volumes, and orphaned containers..."
docker compose down -v --remove-orphans

echo ""
echo "✓ Cleanup complete!"
echo ""
echo "Note: If you want to clean up unused Docker resources system-wide,"
echo "you can run: docker system prune -f"