#!/bin/bash
set -e

echo "=== Installing API Generation Tools ==="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker and try again."
    exit 1
fi

# Pull the external generator Docker image
log_info "Pulling external generator Docker image..."
docker pull ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.1

log_info "✓ External Docker-based generator configured"
log_info "✓ Installation complete!"