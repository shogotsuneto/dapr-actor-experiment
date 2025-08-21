# External Data Store Usage with Dapr Actors

This document explains how to integrate external data stores with Dapr actors, demonstrated through the WalletActor implementation using the `go-simple-eventstore` library.

## Overview

While Dapr provides built-in state management through StateManager, there are scenarios where you might want to use specialized external libraries or databases:

- **Specialized Event Stores**: When you need advanced event sourcing capabilities
- **Legacy Systems Integration**: When integrating with existing database infrastructure  
- **Advanced Query Capabilities**: When you need complex querying not available in basic state stores
- **Multi-tenancy**: When you need tenant-specific database connections
- **Performance Optimization**: When you need specialized storage optimizations

## Architecture Comparison

### Traditional Dapr StateManager Approach (BankAccount)
```go
// Uses Dapr's built-in state management
func (b *BankAccount) appendEvent(ctx context.Context, eventType AccountEventEventType, eventData interface{}) error {
    // Load existing events from Dapr state store
    events, err := b.getAllEvents(ctx)
    if err != nil {
        return err
    }
    
    // Append new event
    events = append(events, event)
    
    // Store back to Dapr StateManager
    eventsKey := "events"
    return b.GetStateManager().Set(ctx, eventsKey, events)
}
```

### External Event Store Approach (Wallet)
```go
// Uses external go-simple-eventstore library
func (w *Wallet) appendExternalEvent(ctx context.Context, eventType WalletEventEventType, eventData interface{}) error {
    // Convert to external event store format
    event := eventstore.Event{
        ID:        uuid.New().String(),
        Type:      string(eventType),
        Data:      dataBytes,
        Timestamp: time.Now(),
        Metadata: map[string]string{
            "actorType": ActorTypeWallet,
            "actorId":   w.ID(),
        },
    }
    
    // Use external event store directly
    streamID := fmt.Sprintf("wallet-%s", w.ID())
    return w.eventStore.Append(streamID, []eventstore.Event{event}, -1)
}
```

## Key Implementation Patterns

### 1. Global Singleton Pattern

The external event store connection is shared across all actor instances using a singleton pattern:

```go
// Global event store instance (singleton pattern)
var globalEventStore eventstore.EventStore

// SetGlobalEventStore sets the shared event store instance that all Wallet actors will use.
// This demonstrates how actors can share global singletons like database connection pools.
func SetGlobalEventStore(store eventstore.EventStore) {
    globalEventStore = store
    log.Printf("Wallet: Global event store configured")
}

// Each actor instance gets a reference to the shared store
func NewWallet() *Wallet {
    return &Wallet{
        eventStore: globalEventStore,
    }
}
```

**Benefits:**
- **Resource Efficiency**: Single connection pool shared across all actors
- **Configuration Management**: Central configuration point for external dependencies
- **Lifecycle Management**: Easy to initialize and cleanup external resources

### 2. Server Initialization

The external event store is initialized once at server startup:

```go
func main() {
    // Initialize external event store for Wallet actors (singleton pattern demonstration)
    log.Println("Initializing external event store (in-memory) for Wallet actors...")
    externalEventStore := memory.NewInMemoryEventStore()
    wallet.SetGlobalEventStore(externalEventStore)
    log.Printf("External event store configured - Wallet actors will use go-simple-eventstore")
    
    // ... rest of server setup
}
```

### 3. Event Store Abstraction

The implementation uses the `eventstore.EventStore` interface, allowing for different backends:

```go
type EventStore interface {
    // Append adds new events to the given stream
    Append(streamID string, events []Event, expectedVersion int) error
    
    // Load retrieves events for the given stream
    Load(streamID string, opts LoadOptions) ([]Event, error)
}
```

**Available Implementations:**
- **In-Memory**: `memory.NewInMemoryEventStore()` (for development/testing)
- **PostgreSQL**: `postgres.NewPostgresEventStore(db, "events")` (for production)
- **Custom**: Implement the interface for any database

## Configuration Examples

### In-Memory Event Store (Development)
```go
// Simple in-memory store for development and testing
externalEventStore := memory.NewInMemoryEventStore()
wallet.SetGlobalEventStore(externalEventStore)
```

### PostgreSQL Event Store (Production)
```go
// Production-ready PostgreSQL backend
db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=password dbname=eventstore sslmode=disable")
if err != nil {
    log.Fatal(err)
}

// Initialize schema
if err := postgres.InitSchema(db, "wallet_events"); err != nil {
    log.Fatal(err)
}

// Create event store
externalEventStore := postgres.NewPostgresEventStore(db, "wallet_events")
wallet.SetGlobalEventStore(externalEventStore)
```

## Event Stream Design

### Stream Naming Convention
Each wallet actor uses a unique stream ID based on its actor ID:

```go
streamID := fmt.Sprintf("wallet-%s", w.ID())
```

This ensures:
- **Isolation**: Each actor instance has its own event stream
- **Discoverability**: Easy to identify which events belong to which actor
- **Scalability**: Supports unlimited actor instances

### Event Metadata
Events include metadata for debugging and auditing:

```go
event := eventstore.Event{
    ID:        uuid.New().String(),
    Type:      string(eventType),
    Data:      dataBytes,
    Timestamp: time.Now(),
    Metadata: map[string]string{
        "actorType": ActorTypeWallet,
        "actorId":   w.ID(),
    },
}
```

## Error Handling and Reliability

### Connection Management
```go
func (w *Wallet) ensureStateLoaded(ctx context.Context) error {
    if w.eventStore == nil {
        return fmt.Errorf("external event store not configured")
    }
    // ... rest of implementation
}
```

### Graceful Degradation
```go
func NewWallet() *Wallet {
    if globalEventStore == nil {
        log.Printf("WARNING: Wallet created without global event store. External persistence disabled.")
    }
    
    return &Wallet{
        eventStore: globalEventStore,
    }
}
```

## Performance Considerations

### In-Memory State Caching
The WalletActor maintains the same caching strategy as other actors:

```go
type Wallet struct {
    actor.ServerImplBaseCtx
    
    // Reference to shared event store (singleton)
    eventStore eventstore.EventStore
    
    // Ephemeral in-memory state for fast access (cached from events)
    cachedState    *WalletState
    stateLoaded    bool  // Track if state has been loaded from events
    walletExists   bool  // Track if wallet exists to avoid repeated checks
}
```

**Benefits:**
- **Fast Operations**: O(1) access time after initial load
- **Reduced Database Load**: Events only loaded once per actor activation
- **Consistency**: State synchronized with events as operations are performed

### Lazy Loading Strategy
```go
func (w *Wallet) ensureStateLoaded(ctx context.Context) error {
    if w.stateLoaded {
        return nil // State already loaded and cached
    }
    
    // Load state from external event store only when needed
    state, err := w.computeStateFromExternalEvents(ctx)
    // ... cache the state
    w.stateLoaded = true
    return nil
}
```

## Testing External Event Stores

### Test Script Example
```bash
# Test wallet operations with external event store
WALLET_ID="test-wallet-001"
BASE_URL="http://localhost:3500/v1.0/actors/Wallet/$WALLET_ID/method"

# Create wallet
curl -X POST "$BASE_URL/CreateWallet" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"ownerName": "John Doe", "currency": "USD", "initialBalance": 100.00}'

# Add funds  
curl -X POST "$BASE_URL/AddFunds" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 50.00, "description": "Monthly allowance"}'

# Get transaction history from external event store
curl -X GET "$BASE_URL/GetTransactions" \
  -H "Authorization: Bearer $TOKEN"
```

### Integration with CI/CD
The external event store is fully compatible with existing test infrastructure:

```bash
# All tests pass including external event store functionality
make test-unit        # Unit tests
make test-integration  # Integration tests with Docker
./scripts/test-multi-actors.sh  # Manual testing
```

## Migration Strategies

### Gradual Migration
You can gradually migrate from Dapr StateManager to external stores:

1. **Start**: All actors use Dapr StateManager
2. **Add**: New actor types use external stores (like WalletActor)
3. **Migrate**: Gradually move existing actors to external stores
4. **Complete**: All actors use external stores if needed

### Data Migration
```go
// Example: Migrate BankAccount events to external store
func MigrateBankAccountToExternalStore(actorID string) error {
    // 1. Load events from Dapr StateManager
    // 2. Convert to external event store format  
    // 3. Append to external store
    // 4. Verify data integrity
    // 5. Switch actor implementation
}
```

## Production Considerations

### Database Connection Pooling
```go
// Configure PostgreSQL with connection pooling
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(25)
db.SetConnMaxLifetime(5 * time.Minute)

store := postgres.NewPostgresEventStore(db, "events")
```

### High Availability
- **Database Clustering**: Use PostgreSQL clustering for HA
- **Read Replicas**: Configure read replicas for query performance
- **Backup Strategy**: Regular backups of event store data
- **Monitoring**: Monitor connection health and query performance

### Security
- **Connection Security**: Use TLS for database connections
- **Access Control**: Implement proper database user permissions
- **Audit Logging**: Log all event store operations
- **Encryption**: Encrypt sensitive event data

## Comparison Matrix

| Aspect | Dapr StateManager | External Event Store |
|--------|------------------|---------------------|
| **Setup Complexity** | Simple | Moderate |
| **Performance** | Good | Excellent (with optimization) |
| **Query Capabilities** | Basic | Advanced |
| **Event Sourcing** | Manual implementation | Native support |
| **Multi-tenancy** | Limited | Full control |
| **Scaling** | Automatic | Manual configuration |
| **Vendor Lock-in** | Dapr ecosystem | Independent |
| **Debugging** | Dapr tools | Database tools |

## Conclusion

External data stores provide powerful capabilities for specialized use cases while maintaining compatibility with Dapr's actor model. The WalletActor demonstrates how to:

- Share global resources across actors (singleton pattern)
- Integrate third-party libraries for specialized functionality
- Maintain performance through caching strategies
- Preserve the actor programming model while using external persistence

This approach is ideal when you need capabilities beyond what Dapr's built-in state management provides, while still leveraging the benefits of the actor pattern for building scalable, distributed applications.