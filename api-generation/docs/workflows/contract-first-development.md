# API-Contract-First Development Workflow

A minimal demo of Dapr actors application in Go, demonstrating actor state management and method invocation patterns with API-contract-first development.

This document demonstrates the complete workflow for API-contract-first development using OpenAPI 3.0 (the only currently implemented schema format).

## Overview

Contract-first development reverses the traditional approach:
- **Traditional**: Code → API Documentation
- **Contract-First**: API Contract → Code Implementation

## Benefits

1. **Consistency**: Implementation must match the contract
2. **Type Safety**: Generated types prevent runtime errors
3. **Documentation**: Contract serves as authoritative documentation
4. **Validation**: Automatic request/response validation
5. **Client Generation**: Automatic client SDK generation

## Complete Example: Counter Actor

### Step 1: Define the API Contract

Create an OpenAPI specification:

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

### Step 2: Generate Code from Contract

```bash
cd api-generation
./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
```

This generates type definitions, HTTP client, and server interface in `../internal/generated/openapi/`.

### Step 3: Implement Against Contract

```go
// internal/actor/counter.go
package actor

import "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"

type CounterActor struct {
    actor.ServerImplBaseCtx
}

// Implementation must use generated types
func (c *CounterActor) Increment(ctx context.Context) (*openapi.CounterState, error) {
    // Contract-compliant implementation
    state, err := c.getState(ctx)
    if err != nil {
        return nil, err
    }
    state.Value++
    return state, c.setState(ctx, state)
}
```

### Step 4: Validate Implementation

The generated types ensure compile-time contract compliance:

```go
// This won't compile if the contract changes
var state *openapi.CounterState = actor.Increment(ctx)

// Request validation is built into the generated types
request := openapi.SetValueRequest{Value: 42} // Type-safe
```

## Best Practices

### Schema Design
- Start with operations: Define what your API does
- Design for evolution: Use versioning and optional fields
- Document everything: Include descriptions and examples
- Validate early: Use schema validation in CI/CD

### Implementation
- Use generated types: Don't create parallel type definitions
- Validate inputs: Use contract-defined validation rules
- Handle errors: Map to contract-defined error formats
- Log contract info: Include contract version in logs

### Testing
- Test contract compliance: Ensure implementation matches schema
- Test edge cases: Validate boundary conditions defined in schema
- Integration tests: Test full request/response cycle

## Resources

- [OpenAPI Specification](https://swagger.io/specification/)
- [Dapr Actors Documentation](https://docs.dapr.io/developing-applications/building-blocks/actors/)

This workflow demonstrates how to implement contract-first development, ensuring that your implementation always matches your API specification.