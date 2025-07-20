# API Generation Tools and Templates

This directory contains tools, templates, and configurations for API-contract-first development.

## Overview

This module supports generating Go code from various API schema definitions, enabling contract-first development workflows where:

1. **API contracts are defined first** using standard schema languages
2. **Code is generated** from these contracts
3. **Implementation follows the contract** rather than the reverse

## Supported Schema Types

### 1. OpenAPI 3.0 (Recommended)
- **File Extension**: `.yaml`, `.yml`, `.json`
- **Use Case**: REST APIs, HTTP services
- **Generation Target**: Go interfaces, types, validators
- **Tools**: `oapi-codegen`, `go-swagger`

### 2. Protocol Buffers (gRPC)
- **File Extension**: `.proto`
- **Use Case**: gRPC services, high-performance APIs
- **Generation Target**: Go structs, gRPC service definitions
- **Tools**: `protoc`, `protoc-gen-go`

### 3. JSON Schema
- **File Extension**: `.json`
- **Use Case**: Data validation, simple type definitions
- **Generation Target**: Go structs with validation tags
- **Tools**: Custom generators, `go-jsonschema`

### 4. GraphQL Schema Definition Language
- **File Extension**: `.graphql`, `.gql`
- **Use Case**: GraphQL APIs, flexible query interfaces
- **Generation Target**: Go resolvers, types
- **Tools**: `gqlgen`

### 5. AsyncAPI
- **File Extension**: `.yaml`, `.yml`
- **Use Case**: Event-driven APIs, message-based systems
- **Generation Target**: Event handlers, message types
- **Tools**: Custom templates

## Directory Structure

```
api-generation/                    # API contract-first development tools
├── schemas/                       # 📄 API schema definitions (source)
│   ├── openapi/                   #     OpenAPI 3.0 specifications
│   ├── protobuf/                  #     Protocol Buffer definitions  
│   ├── jsonschema/                #     JSON Schema files
│   ├── graphql/                   #     GraphQL schema definitions
│   └── asyncapi/                  #     AsyncAPI specifications
├── tools/                         # 🔧 Generation tools and scripts
│   ├── bin/                       #     Installed tool binaries
│   └── scripts/                   #     Installation and generation scripts
└── docs/                          # 📚 Documentation and examples

# Generated code location (outside api-generation):
../internal/generated/             # 🤖 Generated code output (integration)
├── openapi/                       #     Generated from OpenAPI schemas
├── protobuf/                      #     Generated from Protocol Buffers
└── ...                           #     Other generated code types
```

## 🎯 Key Principles

### Separation of Concerns

1. **📄 Schemas** (`schemas/`): Source of truth API contracts
2. **🔧 Tooling** (`tools/`): Generation and validation tools  
3. **🤖 Generated Code** (`../internal/generated/`): Output integrated with main project
4. **📚 Documentation** (`docs/`): Workflows, examples, and guidance

### Tool Installation Strategy

Only currently used tools are installed by default:
- ✅ **OpenAPI tools**: `oapi-codegen` (actively used)
- ⏳ **Other tools**: Available on-demand (protoc, gqlgen, etc.)
│   ## Quick Start

### 1. Install Tools
```bash
cd api-generation
./tools/scripts/install.sh
```

### 2. Generate Code from Schema
```bash
# Generate from OpenAPI (most common)
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml

# Generated code appears in ../internal/generated/openapi/
ls ../internal/generated/openapi/
# types.go  client.go  server.go
```

### 3. Use Generated Code in Your Application
```go
// Import generated types
import generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"

// Use generated types for contract compliance
func (c *CounterActor) Increment(ctx context.Context) (*generated.CounterState, error) {
    // Implementation MUST match the OpenAPI contract
    // ...
}
```

### 4. Run the Contract-Based Actor
```bash
# Build the main server
go build -o bin/server ./cmd/server

# Run with contract-generated types (default and only mode)
./bin/server
```

## Integration with Main Server

The main Dapr server (`cmd/server`) uses the contract-based CounterActor implementation with generated OpenAPI types for type safety and contract compliance.

Check the `/status` endpoint to see actor information.