# JWT-Aware Actors

This document explains the JWT authentication and authorization features added to the Dapr actor experiment.

## Overview

The JWT-aware actors feature allows actors to access verified JWT token information and enforce resource ownership and role-based access control. The authentication happens at the HTTP middleware layer within the same runtime container, without requiring external gateways.

## Features

### JWT Token Validation
- **Automatic token extraction** from `Authorization: Bearer <token>` headers
- **Signature validation** using HMAC (HS256), RSA, or ECDSA algorithms
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
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│     Client      │───▶│  JWT Middleware │───▶│  Actor Method   │
│  (with JWT)     │    │   (validates &  │    │ (accesses JWT   │
│                 │    │  injects claims)│    │    claims)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │                 │    │                 │
                       │  Context with   │    │  Authorization  │
                       │  JWT Claims     │    │     Logic       │
                       └─────────────────┘    └─────────────────┘
```

## Configuration

### Environment Variables

```bash
# JWT secret key for HMAC signature validation
JWT_SECRET=your-secret-key-here

# Required issuer claim (optional)
JWT_ISSUER=your-issuer

# Allow insecure mode for testing (optional)
JWT_INSECURE_MODE=false
```

### Default Test Configuration

For development and testing, the system uses these defaults if no environment variables are set:

```bash
JWT_SECRET=test-secret-key-do-not-use-in-production
JWT_ISSUER=dapr-actor-test
JWT_INSECURE_MODE=false
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

### Successful Operations

```bash
# Generate a valid token
ADMIN_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Access counter with admin token
curl -H "Authorization: Bearer $ADMIN_TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# Create user's own bank account
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

1. **Use strong secret keys**: Generate cryptographically secure keys
2. **Use RSA/ECDSA**: Consider asymmetric keys for better security
3. **Set proper issuer/audience**: Validate token source and destination
4. **Enable HTTPS**: Always use TLS in production
5. **Token rotation**: Implement regular key rotation

### Development vs Production

```bash
# Development (using defaults)
JWT_SECRET=test-secret-key-do-not-use-in-production
JWT_INSECURE_MODE=false

# Production (secure configuration)
JWT_SECRET=$(openssl rand -base64 32)
JWT_ISSUER=your-auth-service
JWT_AUDIENCE=dapr-actor-service
```

## Middleware Configuration

The JWT middleware supports flexible configuration:

```go
jwtConfig := auth.JWTMiddlewareConfig{
    SecretKey: []byte("your-secret-key"),
    RequiredIssuer: "your-issuer",
    RequiredAudience: "your-audience",
    SkipPaths: []string{"/health", "/status"},
    AllowInsecure: false, // Never true in production
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

1. **Single Secret**: Currently supports one HMAC secret per service
2. **Synchronous Validation**: No support for async token validation
3. **No Token Revocation**: No built-in token blacklist support
4. **Memory Storage**: Claims are stored in request context only

## Future Enhancements

- **Multiple Keys**: Support for key rotation with multiple valid keys
- **JWK Support**: JSON Web Key (JWK) integration
- **Token Revocation**: Redis-based token blacklist
- **RBAC Extensions**: More sophisticated role hierarchies
- **Audit Logging**: Structured audit trail for security events