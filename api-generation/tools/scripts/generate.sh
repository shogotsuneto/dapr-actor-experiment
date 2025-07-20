#!/bin/bash
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

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
API_GEN_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
BIN_DIR="$API_GEN_DIR/tools/bin"

# Add tools to PATH for this script
export PATH="$BIN_DIR:$PATH"

# Function to check if tool exists
check_tool() {
    local tool="$1"
    if ! command -v "$tool" &> /dev/null; then
        log_error "$tool not found. Please run install.sh first."
        exit 1
    fi
}

# Usage function
usage() {
    echo "Usage: $0 <schema-type> <schema-file> [output-dir]"
    echo ""
    echo "Currently supported schema types:"
    echo "  openapi     - Generate from OpenAPI 3.0 specification"
    echo ""
    echo "Future schema types (not yet implemented):"
    echo "  protobuf    - Protocol Buffer definition (planned)"
    echo "  jsonschema  - JSON Schema (planned)" 
    echo "  graphql     - GraphQL schema (planned)"
    echo ""
    echo "Examples:"
    echo "  $0 openapi schemas/openapi/counter-actor.yaml"
    echo ""
}

# Parse arguments
SCHEMA_TYPE="$1"
SCHEMA_FILE="$2"
OUTPUT_DIR="$3"

if [ -z "$SCHEMA_TYPE" ] || [ -z "$SCHEMA_FILE" ]; then
    usage
    exit 1
fi

# Set default output directory
if [ -z "$OUTPUT_DIR" ]; then
    # Generate to internal directory for integration with main project
    if [[ "$API_GEN_DIR" == */api-generation ]]; then
        PROJECT_ROOT="$(dirname "$API_GEN_DIR")"
        OUTPUT_DIR="$PROJECT_ROOT/internal/generated/$SCHEMA_TYPE"
    else
        OUTPUT_DIR="$API_GEN_DIR/generated/$SCHEMA_TYPE"
    fi
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Resolve schema file path
if [[ "$SCHEMA_FILE" == /* ]]; then
    SCHEMA_PATH="$SCHEMA_FILE"
else
    SCHEMA_PATH="$API_GEN_DIR/$SCHEMA_FILE"
fi

# Check if schema file exists
if [ ! -f "$SCHEMA_PATH" ]; then
    log_error "Schema file not found: $SCHEMA_PATH"
    exit 1
fi

log_info "=== API Code Generation ==="
log_info "Schema Type: $SCHEMA_TYPE"
log_info "Schema File: $SCHEMA_PATH"
log_info "Output Dir:  $OUTPUT_DIR"
log_info ""

# Generate based on schema type
case "$SCHEMA_TYPE" in
    "openapi")
        log_step "Generating OpenAPI code..."
        check_tool "types-generator"
        check_tool "interface-generator"
        
        # Generate types using custom generator
        log_info "Generating Go types..."
        "$BIN_DIR/types-generator" "$SCHEMA_PATH" "generated" "$OUTPUT_DIR"
        
        # Generate actor interface from OpenAPI specification using custom generator
        log_info "Generating actor interface..."
        "$BIN_DIR/interface-generator" "$SCHEMA_PATH" "generated" "$OUTPUT_DIR/interface.go"
        
        log_info "✓ OpenAPI code generated successfully"
        ;;
        
    "protobuf")
        log_step "Protocol Buffer generation not yet implemented"
        log_error "Protocol Buffer support is planned but not yet implemented."
        log_info "Only OpenAPI generation is currently available."
        exit 1
        ;;
        
    "jsonschema")
        log_step "JSON Schema generation not yet implemented"
        log_error "JSON Schema support is planned but not yet implemented."
        log_info "Only OpenAPI generation is currently available."
        exit 1
        ;;
        
    "graphql")
        log_step "GraphQL generation not yet implemented"
        log_error "GraphQL support is planned but not yet implemented."
        log_info "Only OpenAPI generation is currently available."
        exit 1
        ;;
        
    *)
        log_error "Unknown schema type: $SCHEMA_TYPE"
        usage
        exit 1
        ;;
esac

log_info ""
log_info "Generated files:"
find "$OUTPUT_DIR" -type f -name "*.go" | sort | sed 's/^/  /'

log_info ""
log_info "✓ Code generation completed successfully!"
log_info "Output directory: $OUTPUT_DIR"