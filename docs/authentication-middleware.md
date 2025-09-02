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
  name: bearer-middleware
spec:
  type: middleware.http.bearer
  version: v1
  metadata:
  - name: jwksURL
    value: "http://jwks-mock-api:3000/.well-known/jwks.json"
  - name: issuer
    value: "http://jwks-mock-api:3000"
  - name: audience
    value: "dapr-actor-service"
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

The middleware works in conjunction with Dapr Bearer middleware to provide authentication:

1. **Dapr Bearer Middleware**: Validates JWT tokens using JWKS at the sidecar level
2. **Application Middleware**: Parses the validated JWT token to extract user context

```go
// Middleware configuration
jwtConfig := auth.JWTMiddlewareConfig{
    SkipPaths: []string{
        "/health", 
        "/status",
        "/v1.0/healthz",  // Dapr health check
        "/dapr/config",   // Dapr internal config endpoint
    },
}

// Apply middleware
router.Use(auth.JWTMiddleware(jwtConfig))
```

**Authentication Flow:**
1. Client sends request with `Authorization: Bearer <token>` header
2. Dapr Bearer middleware validates the JWT token using JWKS
3. If valid, Dapr forwards the complete `Authorization` header to the application
4. Application middleware parses the JWT token to extract the `sub` claim for user identification

```go
// Extract the JWT token from Authorization header
authHeader := r.Header.Get("Authorization")
if !strings.HasPrefix(authHeader, "Bearer ") {
    http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
    return
}

jwtToken := strings.TrimPrefix(authHeader, "Bearer ")

// Parse JWT token payload to extract user ID from 'sub' claim
// Since Dapr already validated the token, we can safely decode it
userID, err := extractSubFromJWT(jwtToken)
```

The middleware extracts the `sub` (subject) claim from the JWT token payload and makes it available through the request context.

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
     -d '{"initialDeposit": 1000}' \
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

### Common HTTP Responses

- **401 Unauthorized**: Missing or invalid JWT token, or authentication required
- **403 Forbidden**: Valid token but insufficient permissions for the requested resource

### Error Messages

The middleware and actors return specific error messages for different scenarios:

- `"Authentication required"` - No Authorization header provided
- `"Invalid authorization header format"` - Authorization header doesn't start with "Bearer "
- `"Invalid JWT token format"` - JWT token structure is malformed or cannot be parsed
- `"insufficient permissions: cannot access this account"` - User trying to access resource they don't own

### Skip Paths

The middleware skips authentication for certain paths needed for service operation:

```go
SkipPaths: []string{
    "/health",        // Health check endpoint
    "/status",        // Status information endpoint  
    "/v1.0/healthz",  // Dapr health check
    "/dapr/config",   // Dapr internal configuration endpoint
}
```

## Testing

### Manual Testing

```bash
# Generate test token with user ID
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user-123"}, "expiresIn": 3600}'

# Test counter actor with authentication
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# Test bank account with proper ownership (userID matches actor ID)
curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"initialDeposit": 1000}' \
     http://localhost:3500/v1.0/actors/BankAccount/user-123/method/createAccount
```

### Integration Tests

```bash
# Start test services with Bearer middleware
docker compose -f test/integration/docker-compose.test.yml up -d

# Run all integration tests (includes authentication)
go test -v ./test/integration

# Run specific test files
go test -v ./test/integration -run TestBankAccount
go test -v ./test/integration -run TestCounter
go test -v ./test/integration -run TestMultiActorIntegration

# Clean up
docker compose -f test/integration/docker-compose.test.yml down
```

## Benefits

This authentication architecture provides:

- **Security**: JWT validation handled by Dapr Bearer middleware using industry-standard JWKS
- **Simplicity**: Application focuses on business logic, authentication handled at infrastructure level
- **Clean separation**: Authentication (Dapr) vs user context extraction (application middleware)
- **Standard compliance**: Uses standard JWT tokens and Authorization headers
- **Flexibility**: User context available in all actor methods through Go context
- **Minimal overhead**: JWT parsing only after Dapr validation, no additional HTTP calls

**Architecture Flow:**
```
Client Request + JWT Token
    ↓
Dapr Sidecar (Bearer Middleware)
    ├─ Validates JWT using JWKS
    ├─ Forwards Authorization header if valid
    ↓
Actor Service (JWT Middleware)  
    ├─ Parses JWT token payload
    ├─ Extracts user ID from 'sub' claim
    ├─ Injects user context
    ↓
Actor Methods (with user context)
```

The implementation demonstrates how to add authentication to any HTTP service using Dapr middleware while keeping the application code focused on business logic rather than complex authorization patterns.