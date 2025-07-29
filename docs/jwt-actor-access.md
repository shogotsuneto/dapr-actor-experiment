# JWT Information Access in Dapr Actors

This document explains how JWT information (like user identity and roles) can be accessed and used within Dapr actors for authorization and resource ownership control.

## Architecture Overview

```
Client Request with JWT
    ↓
JWT Gateway (validates JWT, adds headers)
    ↓ (X-JWT-Subject, X-JWT-Role headers)
Dapr Sidecar
    ↓
Actor Service (can access headers)
    ↓ (pass JWT info as parameters)
Actor Methods (implement authorization logic)
```

## JWT Information Flow

### 1. JWT Gateway Processing

The JWT Gateway validates JWT tokens and adds headers to the request:

```go
// From cmd/jwt-gateway/main.go:84-88
if sub, ok := claims["sub"]; ok {
    r.Header.Set("X-JWT-Subject", fmt.Sprintf("%v", sub))
}
if role, ok := claims["role"]; ok {
    r.Header.Set("X-JWT-Role", fmt.Sprintf("%v", role))
}
```

**Available Headers:**
- `X-JWT-Subject`: User identifier from the JWT token (e.g., "user123")
- `X-JWT-Role`: User role from the JWT token (e.g., "admin", "user")

### 2. Service-Level Access

At the Dapr service level, JWT headers are accessible through HTTP request context. However, individual actor methods don't have direct access to HTTP headers since Dapr abstracts the HTTP layer.

### 3. Actor-Level Authorization

To use JWT information in actors, you need to:

1. **Extract JWT information at the service level**
2. **Pass it as parameters to actor methods**
3. **Implement authorization logic within actors**

## Implementation Examples

### JWT-Aware Counter Actor

The `JWTAwareCounter` demonstrates how to implement ownership and role-based access control:

```go
// JWT context structure
type JWTContext struct {
    Subject string `json:"subject"` // From X-JWT-Subject header
    Role    string `json:"role"`    // From X-JWT-Role header
}

// Method with ownership check
func (c *JWTAwareCounter) GetWithOwnership(ctx context.Context, jwtCtx JWTContext) (*CounterStateWithOwner, error) {
    // Get ownership information
    ownership, err := c.getOwnership(ctx)
    if err != nil {
        return nil, err
    }
    
    // Check if user owns this resource
    if ownership.Owner != "" && ownership.Owner != jwtCtx.Subject {
        return nil, errors.New("access denied: you don't own this counter")
    }
    
    // Return state with ownership info
    return &CounterStateWithOwner{
        Value:      state.Value,
        Owner:      ownership.Owner,
        AccessedBy: jwtCtx.Subject,
    }, nil
}
```

### Service-Level JWT Extraction

```go
func extractJWTFromHeaders(req *http.Request) counter.JWTContext {
    return counter.JWTContext{
        Subject: req.Header.Get("X-JWT-Subject"),
        Role:    req.Header.Get("X-JWT-Role"),
    }
}
```

## Use Cases for JWT in Actors

### 1. Resource Ownership

```go
// Check if user owns the resource
if ownership.Owner != jwtCtx.Subject && jwtCtx.Role != "admin" {
    return errors.New("access denied")
}
```

### 2. Role-Based Authorization

```go
// Only admins can transfer ownership
if jwtCtx.Role != "admin" {
    return errors.New("admin role required")
}
```

### 3. Audit Logging

```go
// Log user actions for compliance
log.Printf("User %s (%s) accessed counter %s", 
    jwtCtx.Subject, jwtCtx.Role, c.ID())
```

### 4. Multi-Tenant Isolation

```go
// Ensure users only access their tenant's data
if !strings.HasPrefix(resourceID, jwtCtx.Subject+":") {
    return errors.New("cross-tenant access denied")
}
```

## Practical Implementation Steps

### Step 1: Create JWT-Aware Actor Methods

Add JWT context parameters to your actor methods:

```go
func (a *MyActor) SecureMethod(ctx context.Context, jwtCtx JWTContext, request MyRequest) (*MyResponse, error) {
    // Implement authorization logic
    if !a.hasPermission(jwtCtx, request) {
        return nil, errors.New("access denied")
    }
    
    // Execute business logic
    return a.processRequest(ctx, request, jwtCtx.Subject)
}
```

### Step 2: Create Service-Level Middleware

Extract JWT information at the service level:

```go
func jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        jwtCtx := counter.JWTContext{
            Subject: r.Header.Get("X-JWT-Subject"),
            Role:    r.Header.Get("X-JWT-Role"),
        }
        
        // Add to request context
        ctx := context.WithValue(r.Context(), "jwt", jwtCtx)
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}
```

### Step 3: Update Actor Invocations

Pass JWT context when calling actor methods:

```go
// Extract JWT context from request
jwtCtx := extractJWTFromHeaders(req)

// Call actor with JWT context
result, err := actorClient.InvokeMethod(ctx, &InvokeMethodRequest{
    ActorType: "MyActor",
    ActorID:   actorID,
    Method:    "SecureMethod",
    Data:      struct {
        JWTContext counter.JWTContext `json:"jwt_context"`
        Request    MyRequest          `json:"request"`
    }{
        JWTContext: jwtCtx,
        Request:    request,
    },
})
```

## Security Considerations

### 1. Header Validation
- Always validate that JWT headers are present and non-empty
- Don't trust headers from external sources - only use headers set by your JWT gateway

### 2. Authorization Logic
- Implement defense-in-depth: validate permissions at multiple levels
- Use explicit deny by default - only allow specific actions for specific roles

### 3. Audit Trail
- Log all authorization decisions for security monitoring
- Include user identity, action, resource, and result in logs

### 4. State Management
- Store ownership and permissions in actor state securely
- Validate state integrity on each access

## Current Implementation Status

✅ **JWT Gateway**: Sets `X-JWT-Subject` and `X-JWT-Role` headers  
✅ **Header Forwarding**: Headers are forwarded to Dapr sidecar  
📝 **Actor Access**: Demonstrated in examples (requires service-level extraction)  
📝 **Authorization Logic**: Example implementation provided  

## Next Steps

To fully implement JWT-aware actors in your application:

1. Create JWT-aware versions of your actor methods
2. Add service-level middleware to extract JWT headers
3. Update actor invocations to pass JWT context
4. Implement authorization logic based on your business requirements
5. Add comprehensive audit logging for security compliance

The foundation is in place with the JWT Gateway - the examples in this document show how to build upon it for complete authorization control.