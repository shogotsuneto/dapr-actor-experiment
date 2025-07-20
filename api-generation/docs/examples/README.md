# API Generation Examples

This directory contains complete examples demonstrating API-contract-first development with different schema types.

## Quick Start Example

The fastest way to see contract-first development in action:

```bash
# 1. Install tools
cd api-generation
./tools/scripts/install.sh

# 2. Generate from OpenAPI
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml

# 3. See generated code
ls -la generated/openapi/

# 4. Build with contract-based implementation
cd ..
go build -o bin/server ./cmd/server
```

## Example: OpenAPI → Go Implementation

### 1. Start with the Contract

```yaml
# schemas/openapi/counter-actor.yaml
openapi: 3.0.3
info:
  title: CounterActor API
  version: 1.0.0
paths:
  /{actorId}/method/increment:
    post:
      operationId: incrementCounter
      responses:
        '200':
          content:
            application/json:
              schema:
                type: object
                properties:
                  value:
                    type: integer
```

### 2. Generate Go Code

```bash
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
```

### 3. Implement Using Generated Types

```go
// internal/actor/counter.go
package actor

import (
    "context"
    "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

type CounterActor struct {
    // actor.ServerImplBaseCtx
}

func (c *CounterActor) Increment(ctx context.Context) (*openapi.CounterState, error) {
    // Implementation MUST return the contract-defined type
    return &openapi.CounterState{Value: 42}, nil
}
```

## Comparison Demo

See how the same API can be defined in different schema languages:

### OpenAPI (REST-focused)
```yaml
paths:
  /{actorId}/method/increment:
    post:
      responses:
        '200':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/CounterState'
```

### Protocol Buffers (Performance-focused)
```protobuf
service CounterActor {
  rpc Increment(google.protobuf.Empty) returns (CounterState);
}

message CounterState {
  int32 value = 1;
}
```

### GraphQL (Query-focused)
```graphql
type Mutation {
  incrementCounter(actorId: ID!): CounterActor
}

type CounterActor {
  value: Int!
}
```

### JSON Schema (Validation-focused)
```json
{
  "definitions": {
    "CounterState": {
      "type": "object",
      "properties": {
        "value": {"type": "integer"}
      }
    }
  }
}
```

## Running the Examples

### 1. Generate All Schemas

```bash
# Generate from all schema types
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
./tools/scripts/generate.sh jsonschema schemas/jsonschema/counter-actor.json

# Protocol Buffers (requires protoc)
if command -v protoc &> /dev/null; then
    ./tools/scripts/generate.sh protobuf schemas/protobuf/counter-actor.proto
fi
```

### 2. Compare Generated Code

```bash
# See what was generated
find generated/ -name "*.go" -exec echo "=== {} ===" \; -exec head -20 {} \;
```

### 3. Build Examples

```bash
# Build main server with contract-based implementation
cd ..
go mod tidy
go build -o bin/server ./cmd/server
```

## Key Takeaways

1. **Same API, Different Schemas**: The same business logic can be expressed in multiple schema languages
2. **Generated Code Varies**: Each tool generates different Go code styles
3. **Type Safety**: All approaches provide compile-time type safety
4. **Use Case Matters**: Choose schema type based on your specific needs:
   - **OpenAPI**: REST APIs, public APIs, documentation
   - **Protocol Buffers**: High performance, microservices
   - **JSON Schema**: Data validation, configuration
   - **GraphQL**: Flexible queries, frontend-driven APIs

## Next Steps

1. Try modifying the schemas
2. Regenerate and see how code changes
3. Implement your own contract-based actor
4. Add validation and error handling
5. Create tests that verify contract compliance

This demonstrates the power of contract-first development: define your API once, generate type-safe code, implement with confidence.