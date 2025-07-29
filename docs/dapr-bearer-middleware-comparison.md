# Dapr Bearer Middleware vs Custom JWT Gateway

This document compares the current custom JWT Gateway implementation with Dapr's built-in Bearer middleware for JWT validation.

## Current Implementation: Custom JWT Gateway

### Architecture
```
Client ──[JWT]──▶ JWT Gateway ──▶ Dapr Sidecar ──▶ Actor Service
                      ↓
                 JWKS Server
```

### Advantages
- ✅ **Selective Protection**: Only `/v1.0/actors/*` endpoints require JWT
- ✅ **Custom Logic**: Full control over validation logic
- ✅ **Header Injection**: Adds `X-JWT-Subject` and `X-JWT-Role` headers
- ✅ **External Access**: Works with any client (web, mobile, API)

### Disadvantages
- ❌ **Additional Component**: Requires running separate JWT Gateway service  
- ❌ **Network Hop**: Extra latency from proxy layer
- ❌ **Maintenance**: Custom code to maintain and update

## Alternative: Dapr Bearer Middleware

### Overview
Dapr provides a built-in Bearer middleware that can validate JWT tokens with JWKS support.

**Documentation**: https://docs.dapr.io/reference/components-reference/supported-middleware/middleware-bearer/

### Configuration

```yaml
# middleware-bearer.yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: bearer-middleware
spec:
  type: middleware.http.bearer
  version: v1
  metadata:
  - name: jwksURL
    value: "http://jwks-server:3000/.well-known/jwks.json"
  - name: issuer
    value: "http://localhost:3000"
  - name: audience
    value: "dev-api"
  - name: forwardPayload
    value: "true"
  - name: authHeaderName
    value: "authorization"
```

### Application Configuration

```yaml
# app-config.yaml
apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: dapr-config
spec:
  httpPipeline:
    handlers:
    - name: bearer-middleware
      type: middleware.http.bearer
  # Apply to specific app or globally
  accessControl:
    defaultAction: allow
    trustDomain: "public"
```

### Architecture with Dapr Middleware
```
Client ──[JWT]──▶ Dapr Sidecar ──[Bearer Middleware]──▶ Actor Service
                      ↓
                 JWKS Server
```

## Comparison

| Feature | Custom JWT Gateway | Dapr Bearer Middleware |
|---------|-------------------|------------------------| 
| **Setup Complexity** | High (custom service) | Medium (config files) |
| **Selective Protection** | ✅ Custom logic | ❌ Applied to all endpoints |
| **JWKS Support** | ✅ Yes | ✅ Yes |
| **Header Injection** | ✅ Custom headers | ✅ Standard payload forward |
| **Performance** | Medium (extra hop) | High (native integration) |
| **Maintenance** | High (custom code) | Low (Dapr managed) |
| **Flexibility** | ✅ Full control | ❌ Limited configuration |
| **Production Ready** | Custom testing needed | ✅ Battle-tested by Dapr |

## Implementation Options

### Option 1: Replace with Dapr Bearer Middleware

**Pros:**
- Native Dapr integration
- Less custom code to maintain  
- Better performance (no extra network hop)
- Battle-tested implementation

**Cons:**
- Less flexibility in validation logic
- Applies to all endpoints (no selective protection)
- May require significant configuration changes

### Option 2: Hybrid Approach

Use Dapr Bearer middleware for core validation and custom gateway for selective protection:

```yaml
# Dapr config with bearer middleware for all requests
apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: dapr-config
spec:
  httpPipeline:
    handlers:
    - name: bearer-middleware
      type: middleware.http.bearer
  # Configure access control
  accessControl:
    defaultAction: allow
```

### Option 3: Keep Current Implementation

Continue with custom JWT Gateway for maximum flexibility.

## Migration Guide: Custom Gateway → Dapr Bearer Middleware

### Step 1: Create Middleware Component

```yaml
# components/middleware-bearer.yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: bearer-middleware  
spec:
  type: middleware.http.bearer
  version: v1
  metadata:
  - name: jwksURL
    value: "http://jwks-server:3000/.well-known/jwks.json"
  - name: issuer  
    value: "http://localhost:3000"
  - name: audience
    value: "dev-api"
  - name: forwardPayload
    value: "true"
```

### Step 2: Update Dapr Configuration

```yaml
# configs/dapr-config.yaml
apiVersion: dapr.io/v1alpha1
kind: Configuration
metadata:
  name: dapr-config
spec:
  httpPipeline:
    handlers:
    - name: bearer-middleware
      type: middleware.http.bearer
```

### Step 3: Update Docker Compose

```yaml
# docker-compose.yml
services:
  actor-service-dapr:
    image: "daprio/daprd:latest"
    command: [
      "./daprd",
      "-app-id", "actor-service",
      "-app-port", "8080", 
      "-dapr-http-port", "3500",
      "-config", "/components/dapr-config.yaml",
      "-components-path", "/components"
    ]
    volumes:
      - "./components:/components"
      - "./configs:/configs"
    ports:
      - "3500:3500"  # Direct access (no JWT gateway needed)
```

### Step 4: Remove JWT Gateway

- Remove `jwt-gateway` service from docker-compose.yml
- Remove `cmd/jwt-gateway/` directory
- Remove JWT Gateway Dockerfile

### Step 5: Update Client Calls

Clients now call Dapr sidecar directly:

```bash
# Before (through JWT Gateway)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get

# After (direct to Dapr with Bearer middleware)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:3500/v1.0/actors/Counter/counter-1/method/get
```

## Recommendations

### For Development/Testing
- **Current Implementation**: Provides maximum flexibility for experimentation
- Good for understanding JWT flows and custom validation logic

### For Production
- **Dapr Bearer Middleware**: More robust, battle-tested, and maintainable
- Consider hybrid approach if selective protection is required

### Decision Matrix

| Priority | Recommendation |
|----------|---------------|
| **Flexibility** | Custom JWT Gateway |
| **Maintenance** | Dapr Bearer Middleware |  
| **Performance** | Dapr Bearer Middleware |
| **Selective Protection** | Custom JWT Gateway |
| **Production Readiness** | Dapr Bearer Middleware |

## Next Steps

1. **Evaluate Requirements**: Determine if selective protection is critical
2. **Test Bearer Middleware**: Implement in development environment
3. **Performance Testing**: Compare latency and throughput
4. **Migration Planning**: If switching, plan phased rollout
5. **Documentation Update**: Update guides for chosen approach

The Dapr Bearer middleware is a compelling alternative that reduces maintenance overhead while providing robust JWT validation. Consider it for production deployments where simplicity and reliability are priorities over maximum flexibility.