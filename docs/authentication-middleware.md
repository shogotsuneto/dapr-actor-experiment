# Authentication Middleware

This document explains the authentication middleware added to the Dapr actor experiment, demonstrating how middleware can be integrated to provide user context to actors.

## Overview

The authentication middleware demonstrates how to:
- Add middleware to a Dapr service to authenticate requests
- Extract user information from tokens and make it accessible in actor methods
- Implement simple resource ownership validation using userID

The middleware uses OAuth 2.0 token introspection via an external JWKS Mock API service for token validation.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │    │                 │
│     Client      │───▶│  Authentication │───▶│   JWKS Mock     │───▶│  Actor Method   │
│  (with token)   │    │   Middleware    │    │   API Service   │    │ (accesses user  │
│                 │    │                 │    │ (/introspect)   │    │    context)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │                        │
                                ▼                        ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
                       │                 │    │                 │    │                 │
                       │ Token           │    │  Token          │    │  UserID Access  │
                       │ Introspection   │    │  Validation     │    │  via Context    │
                       │                 │    │                 │    │                 │
                       └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Configuration

### Environment Variables

```bash
# JWKS Mock API introspection endpoint URL
JWKS_INTROSPECT_URL=http://jwks-mock-api:3000/introspect
```

### JWKS Mock API Setup

The JWKS Mock API service is configured in docker-compose.yml:

```yaml
jwks-mock-api:
  image: ghcr.io/shogotsuneto/jwks-mock-api:v0.0.4
  ports:
    - "3000:3000"
  environment:
    - PORT=3000
```

## Middleware Implementation

The middleware extracts the `sub` claim from introspected tokens and stores it as userID in the request context:

```go
// Middleware configuration
jwtConfig := auth.JWTMiddlewareConfig{
    IntrospectURL: "http://jwks-mock-api:3000/introspect",
    SkipPaths: []string{"/health", "/status"},
}

// Apply middleware
router.Use(auth.JWTMiddleware(jwtConfig))
```

## Actor Context Access

Actors can access the authenticated user's ID through the context:

```go
func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
    // Get user ID from context
    userID, ok := auth.GetUserID(ctx)
    if !ok {
        return nil, errors.New("authentication required")
    }
    
    // Use userID in business logic
    log.Printf("User %s setting counter value", userID)
    
    // ... rest of implementation
}
```

## Example Usage

### Token Generation

Generate tokens using the JWKS Mock API:

```bash
# Generate user token
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "user-123"
    },
    "expiresIn": 3600
  }'
```

### Authenticated Requests

```bash
# Extract token from generation response
TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user-123"}, "expiresIn": 3600}' | \
  jq -r '.token')

# Make authenticated request to counter actor
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# Access user's own bank account (userID matches actor ID)
curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"ownerName": "John Doe", "initialDeposit": 1000}' \
     http://localhost:3500/v1.0/actors/BankAccount/user-123/method/createAccount
```

## Resource Ownership Example

The bank account actor demonstrates simple resource ownership where users can only access accounts matching their userID:

```go
func (b *BankAccount) GetBalance(ctx context.Context) (*BankAccountState, error) {
    // Check if user can access this account
    userID, ok := auth.GetUserID(ctx)
    if !ok || userID != b.ID() {
        return nil, errors.New("insufficient permissions: cannot access this account")
    }
    
    // ... rest of implementation
}
```

## Error Handling

### Common Responses

- **401 Unauthorized**: Missing or invalid token
- **403 Forbidden**: Valid token but insufficient permissions

### Error Messages

- `"Missing Authorization header"`
- `"Invalid Authorization header format"`
- `"Invalid JWT token: <details>"`
- `"insufficient permissions: cannot access this account"`

## Testing

### Manual Testing

```bash
# Generate test token
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user-123"}, "expiresIn": 3600}'

# Run integration tests
./scripts/test-jwt-actors.sh
```

### Integration Tests

```bash
# Start test services
docker compose -f test/integration/docker-compose.test.yml up -d

# Run authentication tests
go test -v ./test/integration -run TestAuth

# Clean up
docker compose -f test/integration/docker-compose.test.yml down
```

## Benefits

This approach provides:
- **Simple middleware integration** - demonstrates how to add authentication to any HTTP service
- **Clean context access** - userID available in all actor methods through standard Go context
- **Minimal complexity** - focuses on core middleware concepts rather than complex authorization
- **External token validation** - delegates token complexity to external service
- **Clear separation of concerns** - authentication logic separate from business logic

The example shows how middleware can provide user context to actors while keeping the implementation straightforward and focused on demonstrating core patterns.