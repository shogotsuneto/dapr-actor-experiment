# API Schema Definition Methods Comparison

This document compares various approaches to defining API schemas for contract-first development, specifically in the context of Go services and Dapr actors.

## Overview

Contract-first development requires choosing an appropriate schema definition method. Each approach has distinct advantages, tooling, and use cases.

## Comparison Matrix

| Aspect | OpenAPI 3.0 | Protocol Buffers | JSON Schema | GraphQL SDL | AsyncAPI |
|--------|-------------|------------------|-------------|-------------|----------|
| **Primary Use Case** | REST/HTTP APIs | gRPC Services | Data Validation | GraphQL APIs | Event-Driven APIs |
| **Schema Language** | YAML/JSON | Protocol Buffer Language | JSON | GraphQL SDL | YAML/JSON |
| **Type Safety** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Performance** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **Ecosystem Maturity** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Learning Curve** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **Documentation Quality** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Go Tooling Quality** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |

## Detailed Analysis

### 1. OpenAPI 3.0 (Swagger)

**Best for**: REST APIs, HTTP services, public APIs

**Advantages**:
- Industry standard for REST API documentation
- Excellent tooling ecosystem (Swagger UI, code generators)
- Human-readable YAML/JSON format
- Built-in validation and documentation
- Wide adoption and community support

**Disadvantages**:
- Limited to HTTP/REST paradigm
- Can become verbose for complex schemas
- No built-in support for real-time/streaming

**Go Tools**:
- `oapi-codegen`: Modern, well-maintained generator
- `go-swagger`: Mature but complex
- `swagger-codegen`: Legacy but functional

**Example Use Case**: Dapr actor HTTP API definitions

```yaml
paths:
  /{actorId}/method/increment:
    post:
      summary: Increment counter
      parameters:
        - name: actorId
          in: path
          required: true
          schema:
            type: string
```

### 2. Protocol Buffers (gRPC)

**Best for**: High-performance services, microservices, language-agnostic APIs

**Advantages**:
- Excellent performance (binary serialization)
- Strong type safety
- Built-in versioning support
- Language-agnostic
- Streaming support

**Disadvantages**:
- Binary format (not human-readable)
- Requires gRPC infrastructure
- Steeper learning curve
- Less suitable for public APIs

**Go Tools**:
- `protoc-gen-go`: Official compiler
- `protoc-gen-go-grpc`: gRPC service generation
- `buf`: Modern Protocol Buffer toolchain

**Example Use Case**: Internal microservice communication

```protobuf
service CounterActor {
  rpc Increment(google.protobuf.Empty) returns (CounterState);
  rpc Get(google.protobuf.Empty) returns (CounterState);
}

message CounterState {
  int32 value = 1;
}
```

### 3. JSON Schema

**Best for**: Data validation, simple type definitions, configuration schemas

**Advantages**:
- Simple and focused on data validation
- Lightweight
- JSON-native
- Good for configuration files

**Disadvantages**:
- Limited to data structures (no operations)
- No built-in API semantics
- Fewer code generation options
- Not suitable for full API definitions

**Go Tools**:
- `go-jsonschema`: Basic code generation
- Custom generators using `github.com/xeipuuv/gojsonschema`

**Example Use Case**: Configuration validation, message schemas

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "value": {"type": "integer"}
  }
}
```

### 4. GraphQL Schema Definition Language

**Best for**: GraphQL APIs, flexible query interfaces, client-driven APIs

**Advantages**:
- Flexible query language
- Strong type system
- Self-documenting
- Excellent developer tooling

**Disadvantages**:
- GraphQL-specific (not suitable for REST)
- Complexity for simple use cases
- Caching challenges
- Learning curve for traditional REST developers

**Go Tools**:
- `gqlgen`: Code-first and schema-first generator
- `graphql-go`: Manual implementation helpers

**Example Use Case**: Flexible data APIs, frontend-driven applications

```graphql
type CounterActor {
  id: ID!
  value: Int!
}

type Mutation {
  incrementCounter(actorId: ID!): CounterActor
  setCounter(actorId: ID!, value: Int!): CounterActor
}
```

### 5. AsyncAPI

**Best for**: Event-driven APIs, message-based systems, pub/sub architectures

**Advantages**:
- Designed for asynchronous communication
- Message-oriented thinking
- Good for event sourcing
- Supports multiple protocols

**Disadvantages**:
- Newer standard (less mature tooling)
- Limited Go ecosystem
- Not suitable for synchronous APIs
- Fewer learning resources

**Go Tools**:
- Limited - mostly custom templates
- Community-driven generators

**Example Use Case**: Event-driven architectures, message queues

```yaml
channels:
  counter/incremented:
    publish:
      message:
        payload:
          type: object
          properties:
            actorId: {type: string}
            newValue: {type: integer}
```

## Recommendations by Use Case

### For Dapr Actor APIs (Current Project)
**Recommended: OpenAPI 3.0**
- Perfect fit for HTTP-based actor method invocation
- Excellent documentation and validation
- Strong Go tooling with `oapi-codegen`
- Industry standard for REST APIs

### For High-Performance Microservices
**Recommended: Protocol Buffers + gRPC**
- Superior performance
- Strong typing
- Built-in streaming
- Language agnostic

### For Event-Driven Systems
**Recommended: AsyncAPI + Protocol Buffers**
- AsyncAPI for message flow documentation
- Protocol Buffers for message definitions
- Best of both worlds

### For Public APIs
**Recommended: OpenAPI 3.0**
- Industry standard
- Best documentation tools
- Widest client support

## Implementation Strategy for This Project

Given our Dapr actor context, we'll focus on **OpenAPI 3.0** as the primary approach because:

1. **Perfect fit**: Dapr actors use HTTP method invocation
2. **Tooling maturity**: `oapi-codegen` provides excellent Go generation
3. **Documentation**: Built-in API documentation via Swagger UI
4. **Validation**: Automatic request/response validation
5. **Industry standard**: Widely understood and adopted

We'll also provide examples of other approaches for comparison and educational purposes.

## Next Steps

1. Implement OpenAPI 3.0 code generation
2. Create working examples for each schema type
3. Develop custom templates for Dapr-specific patterns
4. Build tooling for automated generation workflows