# API Generation Examples

This directory contains complete examples demonstrating API-contract-first development using OpenAPI 3.0 specifications.

> **Note**: Currently, only OpenAPI 3.0 generation is implemented and tested. Examples of other schema types are provided for educational comparison purposes.

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

## Schema Format Comparison (Educational)

See how the same API concept can be expressed in different schema languages:

### OpenAPI 3.0 (✅ Currently Implemented)
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

### Protocol Buffers (📚 Reference Only)
```protobuf
service CounterActor {
  rpc Increment(google.protobuf.Empty) returns (CounterState);
}

message CounterState {
  int32 value = 1;
}
```

### GraphQL (📚 Reference Only)
```graphql
type Mutation {
  incrementCounter(actorId: ID!): CounterActor
}

type CounterActor {
  value: Int!
}
```

### JSON Schema (📚 Reference Only)
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

### 1. Generate Code (OpenAPI Only)

```bash
# Generate from OpenAPI (currently supported)
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
```

### 2. See Generated Code

```bash
# See what was generated
ls -la ../internal/generated/openapi/
cat ../internal/generated/openapi/types.go
```

### 3. Build Examples

```bash
# Build main server with contract-based implementation
cd ..
go mod tidy
go build -o bin/server ./cmd/server
```

## Key Takeaways

1. **OpenAPI Implementation**: Currently, only OpenAPI 3.0 generation is fully implemented and tested
2. **Generated Code**: The `oapi-codegen` tool generates clean, type-safe Go code
3. **Type Safety**: OpenAPI generation provides compile-time type safety
4. **Future Expansion**: Other schema types could be added when needed:
   - **Protocol Buffers**: For high performance and microservices
   - **JSON Schema**: For data validation and configuration
   - **GraphQL**: For flexible queries and frontend-driven APIs

## Next Steps

1. Try modifying the schemas
2. Regenerate and see how code changes
3. Implement your own contract-based actor
4. Add validation and error handling
5. Create tests that verify contract compliance

This demonstrates the power of contract-first development: define your API once, generate type-safe code, implement with confidence.