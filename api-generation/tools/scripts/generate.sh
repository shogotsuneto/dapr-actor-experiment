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
    echo "Schema types:"
    echo "  openapi     - Generate from OpenAPI 3.0 specification"
    echo "  protobuf    - Generate from Protocol Buffer definition"
    echo "  jsonschema  - Generate from JSON Schema"
    echo "  graphql     - Generate from GraphQL schema"
    echo ""
    echo "Examples:"
    echo "  $0 openapi schemas/openapi/counter-actor.yaml"
    echo "  $0 protobuf schemas/protobuf/counter.proto"
    echo "  $0 jsonschema schemas/jsonschema/counter.json"
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
        check_tool "oapi-codegen"
        
        # Generate types
        log_info "Generating Go types..."
        oapi-codegen -generate types \
            -package generated \
            -o "$OUTPUT_DIR/types.go" \
            "$SCHEMA_PATH"
        
        # Generate client
        log_info "Generating client code..."
        oapi-codegen -generate client \
            -package generated \
            -o "$OUTPUT_DIR/client.go" \
            "$SCHEMA_PATH"
        
        # Generate server interface
        log_info "Generating server interface..."
        oapi-codegen -generate gorilla \
            -package generated \
            -o "$OUTPUT_DIR/server.go" \
            "$SCHEMA_PATH"
        
        log_info "✓ OpenAPI code generated successfully"
        ;;
        
    "protobuf")
        log_step "Generating Protocol Buffer code..."
        log_error "Protocol Buffer tools not installed. Run install.sh with protobuf support."
        log_info "To add protobuf support: modify install.sh to include protoc-gen-go tools"
        exit 1
        ;;
        
    "jsonschema")
        log_step "Generating JSON Schema code..."
        log_error "JSON Schema tools not installed. Run install.sh with jsonschema support."
        log_info "To add jsonschema support: modify install.sh to include go-jsonschema"
        exit 1
        ;;
        
    "graphql")
        log_step "Generating GraphQL code..."
        log_error "GraphQL tools not installed. Run install.sh with graphql support."
        log_info "To add graphql support: modify install.sh to include gqlgen"
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