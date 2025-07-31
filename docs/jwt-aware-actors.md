# JWT-Aware Actors

This document explains the JWT authentication and authorization features added to the Dapr actor experiment.

## Overview

The JWT-aware actors feature allows actors to access verified JWT token information and enforce resource ownership and role-based access control. The authentication uses OAuth 2.0 token introspection (RFC 7662) via an external JWKS Mock API service, providing production-ready token validation with proper separation of concerns.

## Features

### OAuth 2.0 Token Introspection
- **Automatic token extraction** from `Authorization: Bearer <token>` headers
- **Token introspection** using OAuth 2.0 compliant `/introspect` endpoint
- **RSA signature validation** via JWKS (JSON Web Key Set)
- **Standard claims validation** (issuer, audience, expiration, etc.)
- **Custom claims support** for application-specific data

### Role-Based Access Control
- **Admin roles**: `admin`, `counter_admin`, `bank_admin`
- **Role checking** via `auth.HasRole(ctx, "role_name")`
- **Method-level authorization** in actor implementations

### Resource Ownership
- **Ownership validation** by comparing JWT subject/user_id with actor ID
- **Multi-identifier support** (subject, user_id, username, email)
- **Ownership checking** via `auth.IsResourceOwner(ctx, resourceID)`

### Actor Context Integration
- **JWT claims injection** into request context using `context.WithValue`
- **Easy claims access** in actor methods via helper functions
- **Transparent integration** with existing actor implementations

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │    │                 │
│     Client      │───▶│  JWT Middleware │───▶│   JWKS Mock     │───▶│  Actor Method   │
│  (with JWT)     │    │  (introspects   │    │   API Service   │    │ (accesses JWT   │
│                 │    │   via HTTP)     │    │ (/introspect)   │    │    claims)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │                        │
                                ▼                        ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
                       │                 │    │                 │    │                 │
                       │ OAuth 2.0 Token │    │  RSA Signature  │    │  Authorization  │
                       │  Introspection  │    │   Validation    │    │     Logic       │
                       │    (RFC 7662)   │    │   via JWKS      │    │                 │
                       └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Configuration

### Environment Variables

```bash
# JWKS Mock API introspection endpoint URL
JWKS_INTROSPECT_URL=http://jwks-mock-api:3000/introspect

# Required issuer claim (optional)
JWT_ISSUER=http://jwks-mock-api:3000

# Required audience claim (optional)  
JWT_AUDIENCE=dapr-actor-service
```

### JWKS Mock API Configuration

The JWKS Mock API service is configured in docker-compose.yml:

```yaml
jwks-mock-api:
  image: ghcr.io/shogotsuneto/jwks-mock-api:v0.0.4
  ports:
    - "3000:3000"
  environment:
    - PORT=3000
    - JWT_ISSUER=http://jwks-mock-api:3000
    - JWT_AUDIENCE=dapr-actor-service
    - KEY_COUNT=2
    - KEY_IDS=key-1,key-2
```

## JWT Claims Structure

### Standard Claims
- `sub` (Subject): User identifier
- `iss` (Issuer): Token issuer
- `aud` (Audience): Intended audience
- `exp` (Expiration): Token expiration time
- `iat` (Issued At): Token issuance time
- `nbf` (Not Before): Token validity start time

### Custom Claims
- `user_id`: Application-specific user identifier
- `username`: Human-readable username
- `email`: User email address
- `roles`: Array of role strings

### Example JWT Payload
```json
{
  "sub": "user-123",
  "user_id": "user-123",
  "username": "john_doe",
  "email": "john@example.com",
  "roles": ["user", "counter_admin"],
  "iss": "dapr-actor-test",
  "aud": "dapr-actor-service",
  "iat": 1609459200,
  "exp": 1609462800
}
```

## Actor Implementation Examples

### Counter Actor with Role-Based Access

```go
func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
    // Check if user has admin role for set operations
    if !auth.HasRole(ctx, "admin") && !auth.HasRole(ctx, "counter_admin") {
        userID := auth.GetUserIdentifier(ctx)
        return nil, errors.New("insufficient permissions: admin role required")
    }
    
    // ... rest of the implementation
}
```

### Bank Account Actor with Ownership Validation

```go
func (b *BankAccount) GetBalance(ctx context.Context) (*BankAccountState, error) {
    // Check if user can access this account
    if !auth.IsResourceOwner(ctx, b.ID()) && !auth.HasRole(ctx, "admin") {
        return nil, errors.New("insufficient permissions: cannot access this account")
    }
    
    // ... rest of the implementation
}
```

### JWT Claims Access

```go
func (a *Actor) SomeMethod(ctx context.Context) error {
    // Get JWT claims
    claims, ok := auth.GetJWTClaims(ctx)
    if !ok {
        return errors.New("no JWT claims found")
    }
    
    // Access specific claims
    userID := auth.GetUserIdentifier(ctx)
    hasAdminRole := auth.HasRole(ctx, "admin")
    isOwner := auth.IsResourceOwner(ctx, resourceID)
    
    // Use claims for business logic
    log.Printf("User %s (roles: %v) accessing resource", userID, claims.Roles)
    
    return nil
}
```

## API Usage Examples

### Token Generation

Generate tokens using the JWKS Mock API:

```bash
# Generate admin token
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "admin-user",
      "user_id": "admin-user",
      "username": "admin",
      "email": "admin@example.com",
      "roles": ["admin", "counter_admin", "bank_admin"]
    },
    "expiresIn": 3600
  }'

# Generate regular user token
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "user-123",
      "user_id": "user-123",
      "username": "john_doe", 
      "email": "john@example.com",
      "roles": ["user"]
    },
    "expiresIn": 3600
  }'
```

### Successful Operations

```bash
# Extract token from generation response
ADMIN_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "admin", "roles": ["admin"]}, "expiresIn": 3600}' | \
  jq -r '.token')

# Access counter with admin token
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# Create user's own bank account
USER_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user-123", "roles": ["user"]}, "expiresIn": 3600}' | \
  jq -r '.token')

curl -X POST \
     -H "Authorization: Bearer $USER_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"ownerName": "John Doe", "initialDeposit": 1000}' \
     http://localhost:3500/v1.0/actors/BankAccount/user-123/method/createAccount
```

### Authorization Failures

```bash
# Missing token (401 Unauthorized)
curl http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# Regular user trying to set counter (403 Forbidden)
USER_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user", "roles": ["user"]}, "expiresIn": 3600}' | \
  jq -r '.token')

curl -X POST \
     -H "Authorization: Bearer $USER_TOKEN" \
     -H "Content-Type: application/json" \
     -d '{"value": 42}' \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/set

# User trying to access another user's account (403 Forbidden)
curl -H "Authorization: Bearer $USER_TOKEN" \
     http://localhost:3500/v1.0/actors/BankAccount/other-user/method/getBalance
```

## Testing

### Manual Testing

Use the JWKS Mock API to create test tokens:

```bash
# Generate test tokens
curl -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "user-123",
      "user_id": "user-123", 
      "username": "john_doe",
      "email": "john@example.com",
      "roles": ["user"]
    },
    "expiresIn": 3600
  }'

# Run comprehensive JWT tests
./scripts/test-jwt-actors.sh
```

### Integration Tests

Run the automated integration tests:

```bash
# Start test services
docker compose -f test/integration/docker-compose.test.yml up -d

# Run JWT-specific tests
go test -v ./test/integration -run TestJWT

# Clean up
docker compose -f test/integration/docker-compose.test.yml down
```

## Security Considerations

### Production Deployment

1. **Use dedicated OAuth provider**: Replace JWKS Mock API with production OAuth 2.0 provider
2. **RSA/ECDSA signatures**: Tokens are signed using asymmetric cryptography via JWKS
3. **Set proper issuer/audience**: Validate token source and destination
4. **Enable HTTPS**: Always use TLS in production
5. **Key rotation support**: JWKS service supports multiple keys for seamless rotation

### Development vs Production

```bash
# Development (using JWKS Mock API)
JWKS_INTROSPECT_URL=http://jwks-mock-api:3000/introspect
JWT_ISSUER=http://jwks-mock-api:3000

# Production (using real OAuth provider)
JWKS_INTROSPECT_URL=https://auth.yourcompany.com/oauth2/introspect
JWT_ISSUER=https://auth.yourcompany.com
JWT_AUDIENCE=dapr-actor-service
```

## Middleware Configuration

The JWT middleware supports OAuth 2.0 introspection configuration:

```go
jwtConfig := auth.JWTMiddlewareConfig{
    IntrospectURL: "http://jwks-mock-api:3000/introspect",
    RequiredIssuer: "http://jwks-mock-api:3000",
    RequiredAudience: "dapr-actor-service",
    SkipPaths: []string{"/health", "/status"},
}
```

## Error Handling

### Common Error Responses

- **401 Unauthorized**: Missing or invalid JWT token
- **403 Forbidden**: Valid token but insufficient permissions
- **400 Bad Request**: Malformed token or invalid claims

### Error Messages

- `"Missing Authorization header"`
- `"Invalid Authorization header format"`
- `"Invalid JWT token: <details>"`
- `"insufficient permissions: <reason>"`

## Best Practices

1. **Principle of Least Privilege**: Grant minimal required permissions
2. **Resource Ownership**: Prefer ownership-based access over broad permissions
3. **Logging**: Log authentication/authorization events for audit
4. **Error Messages**: Provide clear but not revealing error messages
5. **Token Expiration**: Use reasonable token lifetimes

## Limitations

1. **External Service Dependency**: Requires JWKS Mock API or OAuth provider for token validation
2. **Network Latency**: Token validation involves HTTP calls to introspection endpoint
3. **Service Availability**: Authentication depends on external service uptime
4. **Memory Storage**: Claims are stored in request context only

## Future Enhancements

- **OAuth Provider Integration**: Support for popular OAuth 2.0 providers (Auth0, Okta, etc.)
- **Token Caching**: Cache introspection responses to reduce network calls
- **Fallback Mechanisms**: Local validation fallback when introspection service is unavailable
- **RBAC Extensions**: More sophisticated role hierarchies and permissions
- **Audit Logging**: Structured audit trail for security events