# API Generation Tools and Templates

This directory contains tools, templates, and configurations for API-contract-first development.

## Overview

This module supports generating Go code from various API schema definitions, enabling contract-first development workflows where:

1. **API contracts are defined first** using standard schema languages
2. **Code is generated** from these contracts
3. **Implementation follows the contract** rather than the reverse

## Supported Schema Types

### OpenAPI 3.0 (Currently Implemented)
- **File Extension**: `.yaml`, `.yml`, `.json`
- **Use Case**: REST APIs, HTTP services, Dapr actor APIs
- **Generation Target**: Go interfaces, types, validators, client code
- **Tools**: `oapi-codegen`
- **Status**: ✅ **Fully implemented and tested**

### Future Schema Support
Additional schema types could be supported in the future:
- **Protocol Buffers**: For gRPC services and high-performance APIs
- **JSON Schema**: For data validation and simple type definitions  
- **GraphQL SDL**: For GraphQL APIs and flexible query interfaces
- **AsyncAPI**: For event-driven APIs and messaging systems

*Currently, only OpenAPI 3.0 is implemented and actively tested.*

## Directory Structure

```
api-generation/                    # API contract-first development tools
├── schemas/                       # 📄 API schema definitions (source)
│   └── openapi/                   #     OpenAPI 3.0 specifications (implemented)
├── tools/                         # 🔧 Generation tools and scripts
│   ├── bin/                       #     Installed tool binaries
│   └── scripts/                   #     Installation and generation scripts
└── docs/                          # 📚 Documentation and examples

# Generated code location (outside api-generation):
../internal/generated/             # 🤖 Generated code output (integration)
└── openapi/                       #     Generated from OpenAPI schemas
```

## 🎯 Key Principles

### Separation of Concerns

1. **📄 Schemas** (`schemas/`): Source of truth API contracts
2. **🔧 Tooling** (`tools/`): Generation and validation tools  
3. **🤖 Generated Code** (`../internal/generated/`): Output integrated with main project
4. **📚 Documentation** (`docs/`): Workflows, examples, and guidance

### Tool Installation Strategy

Only currently implemented and tested tools are installed:
- ✅ **OpenAPI tools**: `oapi-codegen` (actively used and tested)
- ⏳ **Other tools**: Could be added in the future when needed
│   ## Quick Start

### 1. Install Tools
```bash
cd api-generation
./tools/scripts/install.sh
```

### 2. Generate Code from Schema
```bash
# Generate from OpenAPI (only currently supported format)
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