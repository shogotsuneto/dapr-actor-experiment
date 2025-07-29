# Accessing JWT Subject (userId) in Dapr Actors

This document demonstrates patterns for accessing JWT subject information when using Dapr's Bearer middleware for JWT authentication with actors.

## Current Status & Limitations

✅ **JWT Validation Works**: Bearer middleware successfully validates JWT tokens for actor calls
❌ **Direct Header Access**: JWT headers are not directly accessible within actor methods
💡 **Workaround Available**: Service-mediated patterns can extract and pass JWT info to actors

## Implementation Patterns

### 1. Direct Actor Calls (JWT Validated, Subject Not Accessible)

When you call actors directly through Dapr's actor API, Bearer middleware validates the JWT but the subject is not directly accessible within the actor method:

```go
// Standard actor method - JWT validated by middleware but subject not accessible
func (c *Counter) Get(ctx context.Context) (*CounterState, error) {
    // ✅ JWT was validated by Bearer middleware before reaching here
    // ❌ JWT subject (userId) is not directly accessible in this context
    log.Printf("🔐 Actor method called - JWT validated but subject not accessible")
    
    return c.getState(ctx)
}
```

**Usage:**
```bash
# JWT token required - validated by Bearer middleware
curl -H "Authorization: Bearer $TOKEN" \
     "http://localhost:3500/v1.0/actors/CounterActor/my-counter/method/get"
```

### 2. JWT-Aware Actor Methods (Accept User Info as Parameters)

Create actor methods that accept user information as parameters, to be called by service handlers that can extract JWT information:

```go
// JWT-aware method - receives user info as parameter
func (c *Counter) GetWithUser(ctx context.Context, userID string) (*CounterState, error) {
    // ✅ JWT subject (userId) is now accessible as a parameter
    log.Printf("🔐 JWT Subject (userId) accessed within actor: %s", userID)
    
    state, err := c.getState(ctx)
    if err != nil {
        return nil, err
    }
    
    // Implement user-specific logic:
    log.Printf("📊 Counter value %d accessed by user: %s", state.Value, userID)
    
    return state, nil
}
```

### 3. Service-Mediated Pattern (Recommended for JWT Access)

Create service invocation handlers that can potentially access JWT headers and call actors with user information:

```go
// Service handler that could access JWT headers (in a full implementation)
func counterWithJWTHandler(ctx context.Context, in *common.InvocationEvent) (*common.Content, error) {
    // In a complete implementation, JWT headers would be accessible here
    // For now, this demonstrates the pattern
    log.Printf("🔐 Service handler - JWT headers would be accessible here")
    
    // Extract userID from JWT headers (implementation depends on Dapr version)
    userID := "extracted-from-jwt" // Would be extracted from headers
    
    // Call actor method with extracted user info
    // (This would require Dapr client to call actor internally)
    
    return &common.Content{
        Data:        []byte(`{"status": "JWT pattern demo"}`),
        ContentType: "application/json",
    }, nil
}
```

**Usage:**
```bash
# Call service handler that can access JWT and forward to actors
curl -H "Authorization: Bearer $TOKEN" \
     "http://localhost:3500/v1.0/invoke/actor-service/method/counter-with-jwt"
```

## Testing the Current Implementation

The current implementation demonstrates JWT validation working with actors:

```bash
# Start the services
docker compose up -d

# Test JWT validation (this works)
./scripts/test-jwt-validation.sh

# Test the patterns (demonstrates concepts)
./scripts/test-jwt-subject-access.sh
```

## What Works Today

✅ **JWT Validation**: Bearer middleware validates JWT tokens for all actor calls
✅ **Access Control**: Invalid/missing JWTs are rejected with HTTP 401
✅ **Actor Protection**: All actor endpoints require valid JWT authentication
✅ **Service Handlers**: Can potentially access JWT headers for extraction
✅ **Parameter Passing**: Actors can receive user info via method parameters

## What Requires Additional Implementation

❌ **Direct Header Access**: Need to research exact header forwarding mechanism
❌ **Automatic Extraction**: Service handlers need full JWT header extraction logic
❌ **Actor Client**: Service handlers need Dapr client to call actors internally

## Key Points

- **Bearer middleware works perfectly** for JWT validation on actor endpoints
- **JWT subject is not directly accessible** within actor methods today
- **Service-mediated pattern is the recommended approach** for JWT subject access
- **User information can be passed to actors** via method parameters
- **JWT validation happens before** any actor method is called

## Example Log Output

When testing, you'll see:

```
🔐 Actor method called - JWT validated but subject not accessible
🔐 JWT Subject (userId) accessed within actor: demo-user-123
📊 Counter value 0 accessed by user: demo-user-123
```

This shows both the limitation (no direct access) and the solution (parameter passing).