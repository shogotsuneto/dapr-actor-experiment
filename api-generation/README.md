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
api-generation/
├── schemas/                    # API schema definitions
│   ├── openapi/               # OpenAPI specifications
│   ├── protobuf/              # Protocol Buffer definitions
│   ├── jsonschema/            # JSON Schema files
│   ├── graphql/               # GraphQL schema definitions
│   └── asyncapi/              # AsyncAPI specifications
├── templates/                 # Code generation templates
│   ├── openapi/               # OpenAPI generation templates
│   ├── protobuf/              # Protocol Buffer templates
│   └── common/                # Shared templates
├── generated/                 # Generated code output
│   ├── openapi/               # OpenAPI generated code
│   ├── protobuf/              # Protocol Buffer generated code
│   └── interfaces/            # Common interfaces
├── tools/                     # Generation tools and scripts
│   ├── generators/            # Custom code generators
│   ├── validators/            # Schema validators
│   └── scripts/               # Build and generation scripts
└── docs/                      # Documentation and comparisons
    ├── comparisons/           # Schema method comparisons
    ├── examples/              # Usage examples
    └── workflows/             # Development workflows
```

## Getting Started

1. **Install Dependencies**: Run `./tools/scripts/install.sh`
2. **Define Schema**: Place your API schema in the appropriate `schemas/` subdirectory
3. **Generate Code**: Run `./tools/scripts/generate.sh <schema-type> <schema-file>`
4. **Use Generated Code**: Import generated interfaces and types in your implementation

## Examples

See `docs/examples/` for complete examples of each schema type and generation workflow.

## Future Plans

This module is designed to be extracted into a separate repository for broader use across projects requiring API-contract-first development.