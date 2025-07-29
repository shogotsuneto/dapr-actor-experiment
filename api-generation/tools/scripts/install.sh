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

# Check if Go is installed
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.19+ and try again."
    exit 1
fi

log_info "Go version: $(go version)"

# Create tools directory if it doesn't exist
TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$TOOLS_DIR/bin"
mkdir -p "$BIN_DIR"

log_info "Installing tools to: $BIN_DIR"

# Function to install Go tool
install_go_tool() {
    local tool_path="$1"
    local binary_name="$2"
    local version="$3"
    
    if [ -z "$version" ]; then
        log_info "Installing $binary_name (latest)..."
        GOBIN="$BIN_DIR" go install "$tool_path@latest"
    else
        log_info "Installing $binary_name ($version)..."
        GOBIN="$BIN_DIR" go install "$tool_path@$version"
    fi
    
    if [ -f "$BIN_DIR/$binary_name" ]; then
        log_info "✓ $binary_name installed successfully"
    else
        log_error "✗ Failed to install $binary_name"
        return 1
    fi
}

# Install OpenAPI tools (using external Docker generator)
log_info "Setting up external Docker-based code generation tools..."

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker and try again."
    exit 1
fi

# Pull the external generator Docker image
log_info "Pulling external generator Docker image..."
docker pull ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.1

# Create wrapper script for the Docker-based generator
log_info "Creating Docker generator wrapper..."
cat > "$BIN_DIR/generator" << 'EOF'
#!/bin/bash
# Docker wrapper for dapr-actor-gen
# Usage: generator <openapi-file> <base-output-dir>

if [ $# -ne 2 ]; then
    echo "Usage: generator <openapi-file> <base-output-dir>" >&2
    exit 1
fi

SCHEMA_FILE="$1"
OUTPUT_DIR="$2"

# Get absolute paths
SCHEMA_FILE=$(realpath "$SCHEMA_FILE")
OUTPUT_DIR=$(realpath "$OUTPUT_DIR")

# Find templates directory (relative to the generator wrapper location)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATES_DIR="$SCRIPT_DIR/../generator/templates"

if [ ! -d "$TEMPLATES_DIR" ]; then
    echo "Error: Templates directory not found at $TEMPLATES_DIR" >&2
    exit 1
fi

TEMPLATES_DIR=$(realpath "$TEMPLATES_DIR")

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Run the Docker generator with mounted schema, output, and templates
docker run --rm -u root \
    -v "$SCHEMA_FILE:/input.yaml" \
    -v "$OUTPUT_DIR:/output" \
    -v "$TEMPLATES_DIR:/root/templates" \
    ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.1 \
    /input.yaml /output
EOF

chmod +x "$BIN_DIR/generator"
log_info "✓ Docker generator wrapper created successfully"

# Check for external dependencies
log_info "Checking external dependencies..."

log_info "ℹ️  External Docker-based generator configured (replaces internal generator)"

# Create PATH export script
PATH_SCRIPT="$TOOLS_DIR/scripts/setup-env.sh"
cat > "$PATH_SCRIPT" << EOF
#!/bin/bash
# Source this file to add API generation tools to your PATH
export PATH="$BIN_DIR:\$PATH"
echo "API generation tools added to PATH"
echo "Available tools:"
ls -1 "$BIN_DIR" | sed 's/^/  - /'
EOF
chmod +x "$PATH_SCRIPT"

log_info "✓ Installation complete!"
log_info ""
log_info "To use the tools, either:"
log_info "  1. Add $BIN_DIR to your PATH"
log_info "  2. Source $PATH_SCRIPT"
log_info "  3. Use the generation scripts in tools/scripts/"
log_info ""
log_info "Available tools:"
ls -1 "$BIN_DIR" 2>/dev/null | sed 's/^/  - /' || log_warn "No tools found in $BIN_DIR"