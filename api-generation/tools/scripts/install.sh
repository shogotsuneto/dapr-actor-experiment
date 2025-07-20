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

# Install OpenAPI tools (currently used)
log_info "Installing OpenAPI code generation tools..."
install_go_tool "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen" "oapi-codegen"

# Check for external dependencies (if needed for future expansion)
log_info "Checking external dependencies..."

# Note: Additional tools can be installed when needed:
# - protoc (for Protocol Buffers)
# - Additional schema validation tools
log_info "ℹ️  Only OpenAPI tools installed (other tools available on demand)"

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