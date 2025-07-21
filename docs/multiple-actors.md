# Multiple Actor Support & Event Sourcing Demo

This implementation demonstrates support for multiple actor types in a single application, showcasing two different persistence patterns:

## Actor Types

### 1. CounterActor (State-Based Pattern)
- **Type**: `CounterActor`
- **Pattern**: State-based persistence
- **Storage**: Current value only
- **Operations**: `get`, `increment`, `decrement`, `set`

**Characteristics:**
```go
// Stores only current state
type CounterState struct {
    Value int32 `json:"value"`
}

// When increment is called:
// 1. Load current state: {value: 5}
// 2. Modify state: {value: 6}
// 3. Save new state: {value: 6}
// 4. Previous state is lost
```

### 2. BankAccountActor (Event-Sourced Pattern)
- **Type**: `BankAccountActor` 
- **Pattern**: Event sourcing
- **Storage**: Event history + computed state
- **Operations**: `createAccount`, `deposit`, `withdraw`, `getBalance`, `getHistory`

**Characteristics:**
```go
// Stores sequence of events
type StoredEvent struct {
    EventID   string      `json:"eventId"`
    EventType string      `json:"eventType"`
    Timestamp time.Time   `json:"timestamp"`
    Data      interface{} `json:"data"`
}

// When deposit is called:
// 1. Append event: {type: "MoneyDeposited", amount: 100, timestamp: now}
// 2. Current state computed by replaying ALL events
// 3. Full history preserved
```

## Key Differences

| Aspect | CounterActor (State-Based) | BankAccountActor (Event-Sourced) |
|--------|---------------------------|----------------------------------|
| **Storage** | Current value only | Complete event history |
| **History** | No historical data | Full audit trail |
| **Performance** | Fast (direct state access) | Slower (event replay) |
| **Complexity** | Simple | More complex |
| **Auditability** | None | Complete |
| **Storage Size** | Minimal | Grows with usage |
| **Rollback** | Not possible | Can reconstruct any point in time |

## Event Types (BankAccountActor)

### AccountCreated
```json
{
  "eventType": "AccountCreated",
  "data": {
    "ownerName": "John Doe",
    "initialDeposit": 1000.0,
    "createdAt": "2024-01-15T10:30:00Z"
  }
}
```

### MoneyDeposited
```json
{
  "eventType": "MoneyDeposited", 
  "data": {
    "amount": 250.0,
    "description": "Salary deposit",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### MoneyWithdrawn
```json
{
  "eventType": "MoneyWithdrawn",
  "data": {
    "amount": 50.0,
    "description": "ATM withdrawal", 
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

## Generator Enhancements

The OpenAPI generator now supports multiple actor types in a single schema file:

### Multiple ActorType Tags
```yaml
paths:
  /CounterActor/{actorId}/method/get:
    get:
      tags:
        - "ActorType:CounterActor"
        
  /BankAccountActor/{actorId}/method/deposit:
    post:
      tags:
        - "ActorType:BankAccountActor"
```

### Generated Output
```go
// Multiple actor type constants
const ActorTypeCounterActor = "CounterActor"
const ActorTypeBankAccountActor = "BankAccountActor"

// Separate interfaces for each actor
type CounterActorAPIContract interface { ... }
type BankAccountActorAPIContract interface { ... }

// Separate factory functions
func NewCounterActorFactoryContext(...) func() actor.ServerContext
func NewBankAccountActorFactoryContext(...) func() actor.ServerContext
```

## Usage Examples

### CounterActor (State-Based)
```bash
# Get current value
curl http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/get

# Increment
curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/increment

# Set to specific value
curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/set \
  -H "Content-Type: application/json" \
  -d '{"value": 42}'
```

### BankAccountActor (Event-Sourced)
```bash
# Create account
curl -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/createAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "John Doe", "initialDeposit": 1000.0}'

# Deposit money
curl -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 250.0, "description": "Salary"}'

# Get transaction history (shows event sourcing!)
curl http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/getHistory
```

## When to Use Each Pattern

### Use State-Based (like CounterActor) when:
- Simple state requirements
- Performance is critical
- No audit trail needed
- Storage efficiency important

### Use Event-Sourced (like BankAccountActor) when:
- Audit trail required
- Complex business logic
- Need to answer "what happened when"
- Regulatory compliance needed
- Ability to reconstruct state at any point in time

## Testing

Run the test suite to verify both patterns:
```bash
go test ./internal/actor
```

Or use the comprehensive test script:
```bash
./scripts/test-multi-actors.sh
```