# API-Contract-First Development Workflow

This document demonstrates the complete workflow for API-contract-first development using the tools and infrastructure provided in this repository.

## Overview

Contract-first development reverses the traditional approach:

**Traditional Approach**: Code → API Documentation
**Contract-First Approach**: API Contract → Code Implementation

## Benefits

1. **Consistency**: Implementation must match the contract
2. **Type Safety**: Generated types prevent runtime errors
3. **Documentation**: Contract serves as authoritative documentation
4. **Validation**: Automatic request/response validation
5. **Testing**: Contract can be used for testing compliance
6. **Client Generation**: Automatic client SDK generation

## Complete Example: Counter Actor

### Step 1: Define the API Contract

Create an OpenAPI specification that defines your API:

```yaml
# api-generation/schemas/openapi/counter-actor.yaml
openapi: 3.0.3
info:
  title: CounterActor API
  version: 1.0.0
paths:
  /{actorId}/method/increment:
    post:
      operationId: incrementCounter
      # ... rest of specification
```

### Step 2: Generate Code from Contract

```bash
cd api-generation
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
```

This generates:
- `generated/openapi/types.go` - Type definitions
- `generated/openapi/client.go` - HTTP client
- `generated/openapi/server.go` - Server interface

### Step 3: Implement Against Contract

```go
// internal/actor/contract_counter.go
package actor

import "github.com/shogotsuneto/dapr-actor-experiment/api-generation/generated/openapi"

type ContractCounterActor struct {
    actor.ServerImplBaseCtx
}

// Implement methods using generated types
func (c *ContractCounterActor) Increment(ctx context.Context) (*openapi.CounterState, error) {
    // Implementation must return the contract-defined type
}

func (c *ContractCounterActor) Set(ctx context.Context, request openapi.SetValueRequest) (*openapi.CounterState, error) {
    // Implementation must accept the contract-defined type
}
```

### Step 4: Validate Implementation

The generated types ensure compile-time contract compliance:

```go
// This won't compile if the contract changes
var state *openapi.CounterState = actor.Increment(ctx)

// Request validation is built into the generated types
request := openapi.SetValueRequest{
    Value: 42, // Type-safe
}
```

## Schema Comparison Examples

### OpenAPI 3.0 (Recommended for REST APIs)

**Best for**: REST APIs, HTTP services, public APIs

```yaml
openapi: 3.0.3
info:
  title: CounterActor API
  version: 1.0.0
paths:
  /{actorId}/method/increment:
    post:
      summary: Increment counter by 1
      parameters:
        - name: actorId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CounterState'
```

**Generation Command**:
```bash
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
```

**Generated Go Code**:
```go
type CounterState struct {
    Value int32 `json:"value"`
}

type ServerInterface interface {
    IncrementCounter(w http.ResponseWriter, r *http.Request, actorId ActorId)
}
```

### Protocol Buffers (Best for gRPC/High Performance)

**Best for**: High-performance services, microservices

```protobuf
syntax = "proto3";

service CounterActor {
  rpc Increment(google.protobuf.Empty) returns (CounterState);
}

message CounterState {
  int32 value = 1;
}
```

**Generation Command**:
```bash
# Requires protoc to be installed
./tools/scripts/generate.sh protobuf schemas/protobuf/counter-actor.proto
```

**Generated Go Code**:
```go
type CounterState struct {
    Value int32 `protobuf:"varint,1,opt,name=value,proto3" json:"value,omitempty"`
}

type CounterActorServer interface {
    Increment(context.Context, *empty.Empty) (*CounterState, error)
}
```

### JSON Schema (Best for Data Validation)

**Best for**: Data validation, configuration schemas

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "CounterState": {
      "type": "object",
      "properties": {
        "value": {
          "type": "integer",
          "format": "int32"
        }
      },
      "required": ["value"]
    }
  }
}
```

**Generation Command**:
```bash
./tools/scripts/generate.sh jsonschema schemas/jsonschema/counter-actor.json
```

### GraphQL (Best for Flexible APIs)

**Best for**: GraphQL APIs, client-driven queries

```graphql
type CounterActor {
  id: ID!
  value: Int!
}

type Mutation {
  incrementCounter(actorId: ID!): CounterActor
}
```

**Generation Command**:
```bash
./tools/scripts/generate.sh graphql schemas/graphql/counter-actor.graphql
```

## Integration with Existing Code

### Option 1: Replace Existing Implementation

```go
// Replace the original CounterActor with ContractCounterActor
s.RegisterActorImplFactoryContext(func() actor.ServerContext {
    return &actor.ContractCounterActor{}
})
```

### Option 2: Side-by-Side Comparison

```go
// Register both implementations for comparison
s.RegisterActorImplFactoryContext(func() actor.ServerContext {
    return &actor.CounterActor{} // Original
})

s.RegisterActorImplFactoryContext(func() actor.ServerContext {
    return &actor.ContractCounterActor{} // Contract-based
})
```

## Testing Contract Compliance

### 1. Type Safety Testing

```go
func TestContractCompliance(t *testing.T) {
    actor := &ContractCounterActor{}
    
    // This ensures the method signatures match the contract
    var _ func(context.Context) (*openapi.CounterState, error) = actor.Increment
    var _ func(context.Context, openapi.SetValueRequest) (*openapi.CounterState, error) = actor.Set
}
```

### 2. Schema Validation Testing

```go
func TestResponseValidation(t *testing.T) {
    response := &openapi.CounterState{Value: 42}
    
    // Validate against OpenAPI schema
    jsonData, _ := json.Marshal(response)
    // Use OpenAPI validator to ensure compliance
}
```

## Automated Workflows

### 1. CI/CD Integration

```yaml
# .github/workflows/api-generation.yml
name: API Generation
on: [push, pull_request]
jobs:
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Setup Go
        uses: actions/setup-go@v2
      - name: Install tools
        run: cd api-generation && ./tools/scripts/install.sh
      - name: Generate code
        run: cd api-generation && ./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
      - name: Verify no changes
        run: git diff --exit-code
```

### 2. Pre-commit Hooks

```bash
#!/bin/sh
# .git/hooks/pre-commit
cd api-generation
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml

if ! git diff --quiet --exit-code; then
    echo "Generated code is out of date. Please run generation and commit changes."
    exit 1
fi
```

## Best Practices

### 1. Schema Design

- **Start with operations**: Define what your API does
- **Design for evolution**: Use versioning and optional fields
- **Document everything**: Include descriptions and examples
- **Validate early**: Use schema validation in CI/CD

### 2. Implementation

- **Use generated types**: Don't create parallel type definitions
- **Validate inputs**: Use contract-defined validation rules
- **Handle errors**: Map to contract-defined error formats
- **Log contract info**: Include contract version in logs

### 3. Testing

- **Test contract compliance**: Ensure implementation matches schema
- **Test edge cases**: Validate boundary conditions defined in schema
- **Test evolution**: Ensure backward compatibility
- **Integration tests**: Test full request/response cycle

## Migration Strategy

### Phase 1: Parallel Implementation
1. Create contract for existing API
2. Generate code from contract
3. Implement contract-based version
4. Run both implementations side-by-side
5. Compare outputs for consistency

### Phase 2: Gradual Adoption
1. Route subset of traffic to contract implementation
2. Monitor for differences
3. Fix any discrepancies
4. Gradually increase traffic to contract version

### Phase 3: Full Migration
1. Route all traffic to contract implementation
2. Remove original implementation
3. Update documentation
4. Train team on contract-first workflow

## Tools and Resources

### Required Tools
- Go 1.19+
- `oapi-codegen` (installed via scripts)
- `protoc` (for Protocol Buffers)

### Optional Tools
- OpenAPI editors (Swagger Editor, Stoplight)
- Schema validators
- Mock servers
- API testing tools

### External Resources
- [OpenAPI Specification](https://swagger.io/specification/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [JSON Schema Specification](https://json-schema.org/)
- [GraphQL Schema Language](https://graphql.org/learn/schema/)

## Troubleshooting

### Common Issues

1. **Generation Fails**
   - Check schema syntax
   - Verify tool installation
   - Check file paths

2. **Type Mismatches**
   - Regenerate code after schema changes
   - Check import paths
   - Verify package names

3. **Runtime Errors**
   - Validate request/response formats
   - Check schema constraints
   - Review error handling

### Getting Help

1. Check tool documentation
2. Validate schema with online tools
3. Review generated code
4. Test with simple examples first

This workflow demonstrates how to implement true contract-first development, ensuring that your implementation always matches your API specification.