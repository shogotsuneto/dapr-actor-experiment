# JWT Validation with JWKS

This project now includes JWT (JSON Web Token) validation for Dapr actor endpoints using JWKS (JSON Web Key Set) for key management. This implementation provides secure access control to actor endpoints while maintaining compatibility for non-actor endpoints.

## Architecture

```
Client ──[JWT Token]──▶ JWT Gateway (port 3500) ──▶ Dapr Sidecar (port 3501) ──▶ Actor Service (port 8080)
                              ↓
                         JWKS Server (port 3001)
                         validates token signature
```

## Features

- **🔐 JWT Validation**: Actor endpoints require valid JWT tokens
- **🔑 JWKS Integration**: Public keys fetched from JWKS endpoint for signature validation
- **⚡ Selective Protection**: Only `/v1.0/actors/*` endpoints require JWT validation
- **🚫 Automatic Rejection**: Invalid/missing JWT tokens return HTTP 401 Unauthorized
- **🔄 Auto-Refresh**: JWKS keys are automatically refreshed every hour
- **📊 Comprehensive Logging**: Detailed logs for validation attempts and results

## Components

### 1. JWKS Mock Server (`jwks-server`)
- **Port**: 3001 (external)
- **Purpose**: Provides JWKS endpoints and JWT token generation
- **Image**: `shogotsuneto/jwks-mock-api:v0.0.3`
- **Endpoints**:
  - `GET /.well-known/jwks.json` - JWKS endpoint
  - `POST /generate-token` - Generate JWT tokens for testing
  - `GET /health` - Health check

### 2. JWT Gateway (`jwt-gateway`)
- **Port**: 3500 (external, replaces direct Dapr access)
- **Purpose**: Validates JWT tokens and proxies requests to Dapr sidecar
- **Protection**: Only `/v1.0/actors/*` endpoints require JWT validation
- **Pass-through**: Health checks and non-actor endpoints work without JWT

### 3. Dapr Sidecar (`actor-service-dapr`)
- **Port**: 3501 (internal)
- **Purpose**: Dapr runtime for actor invocation
- **Access**: Only accessible through JWT Gateway

### 4. Actor Service (`actor-service`)
- **Port**: 8080 (internal)
- **Purpose**: Hosts Counter and BankAccount actors
- **Status**: Shows JWT validation configuration

## Configuration

JWT validation is configured via environment variables:

```bash
# JWKS configuration
JWKS_URL=http://jwks-server:3000/.well-known/jwks.json
JWT_ISSUER=http://localhost:3000
JWT_AUDIENCE=dev-api

# Gateway configuration
PROXY_PORT=3500
DAPR_ENDPOINT=http://actor-service-dapr:3501
```

## Quick Start

1. **Start all services**:
   ```bash
   docker compose up -d
   ```

2. **Generate a JWT token**:
   ```bash
   curl -X POST http://localhost:3001/generate-token \
     -H "Content-Type: application/json" \
     -d '{"claims": {"sub": "user123", "role": "admin"}, "expiresIn": 3600}'
   ```

3. **Test JWT validation**:
   ```bash
   # Run comprehensive test suite
   ./scripts/test-jwt-validation.sh
   ```

## Testing JWT Validation

### 1. Without JWT Token (Should Fail)
```bash
curl http://localhost:3500/v1.0/actors/CounterActor/test/method/get
# Expected: HTTP 401 Unauthorized
```

### 2. With Valid JWT Token (Should Pass)
```bash
TOKEN="your-jwt-token-here"
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:3500/v1.0/actors/CounterActor/test/method/get
# Expected: Passes JWT validation, reaches Dapr
```

### 3. Non-Actor Endpoints (No JWT Required)
```bash
curl http://localhost:3500/v1.0/healthz
# Expected: HTTP 204 No Content (no JWT required)
```

## Token Generation Examples

### Basic Token
```bash
curl -X POST http://localhost:3001/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user123", "role": "user"}, "expiresIn": 3600}'
```

### Token with Custom Claims
```bash
curl -X POST http://localhost:3001/generate-token \
  -H "Content-Type: application/json" \
  -d '{
    "claims": {
      "sub": "user123",
      "name": "John Doe",
      "role": "admin",
      "permissions": ["read", "write"],
      "department": "Engineering"
    },
    "expiresIn": 7200
  }'
```

## Validation Behavior

| Endpoint Type | JWT Required | Behavior |
|---------------|--------------|----------|
| `/v1.0/actors/*` | ✅ Yes | Validates JWT, rejects if invalid/missing |
| `/v1.0/healthz` | ❌ No | Passes through without validation |
| `/health` | ❌ No | Passes through without validation |
| Service invocation | ❌ No | Passes through without validation |

## JWT Claims Validation

The JWT middleware validates:
- **Signature**: Using RSA public keys from JWKS
- **Issuer (`iss`)**: Must match `JWT_ISSUER` environment variable
- **Audience (`aud`)**: Must match `JWT_AUDIENCE` environment variable
- **Expiration (`exp`)**: Token must not be expired
- **Issued At (`iat`)**: Token must have valid issued time

## Monitoring and Logging

### JWT Gateway Logs
```bash
# View JWT validation logs
docker compose logs jwt-gateway

# Example log entries:
# JWT validation successful for GET /v1.0/actors/CounterActor/test/method/get, subject: user123
# JWT validation failed: authorization header not found
# JWT validation failed: invalid issuer: expected http://localhost:3000, got http://other-issuer
```

### Actor Service Status
```bash
# Check JWT configuration status
curl http://localhost:8080/status

# With JWT (when available):
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/status
```

## Security Considerations

1. **HTTPS in Production**: Use HTTPS for all JWT communication in production
2. **Token Expiration**: Set appropriate token expiration times
3. **Key Rotation**: JWKS keys are automatically refreshed every hour
4. **Issuer Validation**: Always validate the JWT issuer
5. **Audience Validation**: Ensure tokens are intended for your service

## Troubleshooting

### Common Issues

1. **JWT validation failed: authorization header not found**
   - Missing `Authorization: Bearer <token>` header
   - Solution: Add proper Authorization header

2. **JWT validation failed: invalid issuer**
   - Token issuer doesn't match expected issuer
   - Solution: Check `JWT_ISSUER` environment variable

3. **JWT validation failed: invalid audience**
   - Token audience doesn't match expected audience  
   - Solution: Check `JWT_AUDIENCE` environment variable

4. **JWT validation failed: failed to parse token**
   - Malformed or invalid JWT token
   - Solution: Generate a new valid token

### Debug Commands

```bash
# Check service health
curl http://localhost:3001/health  # JWKS Server
curl http://localhost:3500/v1.0/healthz  # JWT Gateway

# View JWKS
curl http://localhost:3001/.well-known/jwks.json

# Check logs
docker compose logs jwt-gateway
docker compose logs jwks-server
docker compose logs actor-service
```

## Implementation Details

### JWT Middleware (`internal/auth/jwt_middleware.go`)
- Uses `github.com/golang-jwt/jwt/v5` for JWT parsing
- Uses `github.com/MicahParks/keyfunc/v2` for JWKS integration
- Configurable issuer and audience validation
- Automatic JWKS key refresh

### JWT Gateway (`cmd/jwt-gateway/main.go`)
- HTTP reverse proxy with JWT validation
- Selective endpoint protection
- Proper error handling and logging
- Headers forwarding to downstream services

## Migration from Non-JWT Setup

If upgrading from a setup without JWT validation:

1. **Backward Compatibility**: Non-actor endpoints continue to work
2. **Actor Endpoints**: Now require JWT tokens
3. **Port Changes**: External clients should connect to port 3500 (JWT Gateway) instead of port 3501 (Dapr directly)
4. **Token Generation**: Use JWKS server endpoints to generate tokens for testing

## Development and Testing

For development without JWT validation:
1. Stop the JWT Gateway: `docker compose stop jwt-gateway`
2. Access Dapr directly: `http://localhost:3501` (if port is exposed)
3. Or disable JWT by modifying docker-compose.yml

For automated testing:
```bash
# Run JWT validation test suite
./scripts/test-jwt-validation.sh

# Expected output: All tests should pass with ✅ status
```