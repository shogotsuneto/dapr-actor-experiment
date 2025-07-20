#!/bin/bash

echo "Starting Dapr Actor Demo with Docker Compose..."
echo ""

# Use the main Docker script for consistency
exec "$(dirname "$0")/run-docker.sh"