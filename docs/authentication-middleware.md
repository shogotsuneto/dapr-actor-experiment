# Authentication Middleware

This document explains the authentication middleware added to the Dapr actor experiment, demonstrating how middleware can be integrated to provide user context to actors.

## Overview

The authentication middleware demonstrates how to:
- Add middleware to a Dapr service to authenticate requests
- Extract user information from tokens and make it accessible in actor methods
- Implement simple resource ownership validation using userID

The middleware works with Dapr's Bearer middleware component which handles JWT validation at the sidecar level.

## Configuration

### Dapr Bearer Middleware Component

The Bearer middleware component is configured to validate JWT tokens using JWKS:

```yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: bearer-token
spec:
  type: middleware.http.bearer
  version: v1
  metadata:
  - name: clientId
    value: "your-client-id"
  - name: issuer
    value: "http://jwks-mock-api:3000"
  - name: audience
    value: "your-audience"
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

The middleware extracts the `sub` claim from JWT tokens validated by Dapr Bearer middleware and stores it as userID in the request context:

```go
// Middleware configuration
jwtConfig := auth.JWTMiddlewareConfig{
    SkipPaths: []string{"/health", "/status"},
}

// Apply middleware
router.Use(auth.JWTMiddleware(jwtConfig))
```

When Dapr Bearer middleware validates a JWT token, it forwards the entire Authorization header to the application service. The application middleware then parses the JWT token to extract user information:

```go
// Extract the JWT token from Authorization header
authHeader := r.Header.Get("Authorization")
jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

// Parse JWT token to extract user ID from 'sub' claim
userID, err := extractSubFromJWT(jwtToken)
```

The middleware extracts the `sub` (subject) claim from the JWT token payload and makes it available through the context.

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
- **Dapr-native authentication** - uses Dapr Bearer middleware for token validation
- **Clear separation of concerns** - authentication logic separate from business logic

The example shows how middleware can provide user context to actors while keeping the implementation straightforward and focused on demonstrating core patterns.