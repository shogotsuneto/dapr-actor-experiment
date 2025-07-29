#!/bin/bash
# Dapr Actor Code Generator
# Generates Go actor code from OpenAPI schemas using external Docker generator

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Usage function
usage() {
    echo "Dapr Actor Code Generator"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  generate    - Generate actor code from OpenAPI schema (default)"
    echo "  install     - Install code generation tools"
    echo "  clean       - Clean generated code"
    echo "  help        - Show this help message"
    echo ""
    echo "Options for generate:"
    echo "  -s, --schema <file>    - OpenAPI schema file (default: api-generation/schemas/openapi/multi-actors.yaml)"
    echo "  -o, --output <dir>     - Output directory (default: internal/)"
    echo ""
    echo "Examples:"
    echo "  $0                     # Generate with defaults"
    echo "  $0 generate            # Same as above"
    echo "  $0 install             # Install tools"
    echo "  $0 clean               # Clean generated files"
    echo ""
}

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default values
DEFAULT_SCHEMA="$SCRIPT_DIR/api-generation/schemas/openapi/multi-actors.yaml"
DEFAULT_OUTPUT="$SCRIPT_DIR/internal"
COMMAND="generate"
SCHEMA_FILE="$DEFAULT_SCHEMA"
OUTPUT_DIR="$DEFAULT_OUTPUT"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        generate|install|clean|help)
            COMMAND="$1"
            shift
            ;;
        -s|--schema)
            SCHEMA_FILE="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -h|--help)
            COMMAND="help"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Execute command
case "$COMMAND" in
    "help")
        usage
        exit 0
        ;;
        
    "install")
        log_info "=== Installing Code Generation Tools ==="
        cd "$SCRIPT_DIR/api-generation"
        ./tools/scripts/install.sh
        log_info "✓ Installation complete!"
        ;;
        
    "clean")
        log_info "=== Cleaning Generated Code ==="
        log_step "Removing generated actor files..."
        
        # Clean counter actor generated files
        rm -f "$SCRIPT_DIR/internal/counter/types.go"
        rm -f "$SCRIPT_DIR/internal/counter/api.go"  
        rm -f "$SCRIPT_DIR/internal/counter/factory.go"
        
        # Clean bankaccount actor generated files
        rm -f "$SCRIPT_DIR/internal/bankaccount/types.go"
        rm -f "$SCRIPT_DIR/internal/bankaccount/api.go"
        rm -f "$SCRIPT_DIR/internal/bankaccount/factory.go"
        
        log_info "✓ Generated code cleaned (implementation files preserved)"
        ;;
        
    "generate")
        log_info "=== Generating Actor Code ==="
        
        # Check if schema file exists
        if [ ! -f "$SCHEMA_FILE" ]; then
            log_error "Schema file not found: $SCHEMA_FILE"
            exit 1
        fi
        
        # Ensure Docker is available  
        if ! command -v docker &> /dev/null; then
            log_error "Docker is not installed. Please install Docker and try again."
            exit 1
        fi
        
        # Ensure Docker image is available
        if ! docker image inspect ghcr.io/shogotsuneto/dapr-actor-gen:v0.0.2 >/dev/null 2>&1; then
            log_warn "Docker generator image not found. Installing now..."
            cd "$SCRIPT_DIR/api-generation"
            ./tools/scripts/install.sh
            cd - > /dev/null
        fi
        
        log_step "Generating from schema: $SCHEMA_FILE"
        log_step "Output directory: $OUTPUT_DIR"
        
        # Run generation
        cd "$SCRIPT_DIR/api-generation"
        ./tools/scripts/generate.sh openapi "$SCHEMA_FILE"
        
        log_info "✓ Actor code generation completed successfully!"
        log_info ""
        log_info "Generated files:"
        find "$SCRIPT_DIR/internal" -name "*.go" -path "*/counter/*" -o -path "*/bankaccount/*" | grep -E "(types|api|factory)\.go$" | sort | sed 's/^/  /'
        ;;
        
    *)
        log_error "Unknown command: $COMMAND"
        usage
        exit 1
        ;;
esac