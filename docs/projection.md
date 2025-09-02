# BankAccount Transactions Projection System

This document explains the BankAccount transactions projection feature that demonstrates CQRS (Command Query Responsibility Segregation) patterns using event sourcing with modern projection libraries.

## Overview

The projection system demonstrates a complete CQRS architecture with the following components:

1. **Projector Worker** (`cmd/projector`) - Uses go-simple-es-projector v0.0.2 to continuously read events from `bankaccount_events` and projects them to a `transactions` table
2. **Simple Query Server** ([shogotsuneto/simple-query-server v0.0.2](https://github.com/shogotsuneto/simple-query-server)) - Provides JWT-authenticated query endpoints with account holder access control

## CQRS Architecture

```
Command Side (Write)                 Event Store                    Query Side (Read)
┌─────────────────┐    Events     ┌──────────────────┐   Events   ┌─────────────────┐
│  BankAccount    │─────────────→ │  bankaccount_    │ ─────────→ │   Projector     │
│     Actors      │               │     events       │            │    Worker       │
│  (Commands)     │               │   (Event Store)  │            │ (Event→Table)   │
└─────────────────┘               └──────────────────┘            └─────────────────┘
                                                                            │
                                                                            ▼
                                                                  ┌─────────────────┐
                                                                  │  transactions   │
                                                                  │     table       │
                                                                  │  (Read Model)   │
                                                                  └─────────────────┘
                                                                            │
                                                                            ▼
                                                                 ┌─────────────────┐
                                                                 │ Simple Query    │
                                                                 │     Server      │
                                                                 │ (JWT Protected  │
                                                                 │    Queries)     │
                                                                 └─────────────────┘
```

This architecture demonstrates:
- **Command Side**: BankAccount actors handle commands and emit events
- **Event Store**: Immutable event log using go-simple-eventstore v0.0.9
- **Query Side**: Separate read model optimized for queries
- **Projection**: Automated transformation from events to queryable tables
- **Authentication**: JWT-based access control on the read side

## Modern Design Features

### JWT Authentication with Account Ownership
- All query endpoints require valid JWT tokens in Authorization header
- JWT `sub` claim is mapped to `user_id` parameter for automatic owner filtering
- Users can only access their own transaction data via `owner_id` filtering
- Demonstrates secure multi-tenant query patterns

### Enhanced Event Structure
- Events include explicit `AccountId` field for self-contained projection
- No dependency on stream ID parsing for account extraction
- Events: `AccountCreatedEventV1`, `MoneyDepositedEventV1`, `MoneyWithdrawnEventV1`
- Each event contains all necessary information for projection

### Cursor-Based Event Consumption
- Uses go-simple-eventstore v0.0.9 with cursor-based positioning
- Reliable resumption after restarts using cursor checkpoints
- Eliminates timestamp polling issues with precise event ordering

### Optimistic Insert Idempotency
- Simple idempotency using database unique constraints
- `UNIQUE(account_id, event_version)` prevents duplicate projections
- Uses `ON CONFLICT DO NOTHING` for automatic reprocessing safety
- No complex event tracking tables required

## Transactions Table Schema

The `transactions` table stores projected transaction data with owner information:

```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,  -- For JWT-based access control
    transaction_type VARCHAR(50) NOT NULL, -- 'account_created', 'deposit', 'withdrawal'
    amount DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    description TEXT,
    transaction_timestamp TIMESTAMPTZ NOT NULL,
    event_version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(account_id, event_version) -- Optimistic insert idempotency
);
```

## Enhanced Event Projection

The projector transforms self-contained events using direct field extraction:

### AccountCreatedV1 Events (Enhanced)
```json
{
  "accountId": "account-alice",    // ← New: explicit account ID
  "ownerId": "user-123", 
  "initialDeposit": 1000.0,
  "createdAt": "2025-08-26T04:14:35Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'user-123', 'account_created', 1000.00, 'Account created with initial deposit', '2025-08-26T04:14:35Z', 1);
```

### MoneyDepositedV1 Events (Enhanced)
```json
{
  "accountId": "account-alice",    // ← New: explicit account ID
  "ownerId": "user-123",           // ← New: owner information
  "amount": 2500.0,
  "description": "Salary deposit",
  "timestamp": "2025-08-26T04:14:48Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'user-123', 'deposit', 2500.00, 'Salary deposit', '2025-08-26T04:14:48Z', 2);
```

### MoneyWithdrawnV1 Events (Enhanced)
```json
{
  "accountId": "account-alice",    // ← New: explicit account ID
  "ownerId": "user-123",           // ← New: owner information
  "amount": 1200.0,
  "description": "Rent payment", 
  "timestamp": "2025-08-26T04:14:48Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'user-123', 'withdrawal', 1200.00, 'Rent payment', '2025-08-26T04:14:48Z', 3);
```

## Configuration

### Projector Configuration
Environment variables for go-simple-es-projector:
- `POSTGRES_CONNECTION_STRING` - Database connection string (default: postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable)
- `EVENTS_TABLE_NAME` - Event store table name (default: bankaccount_events)
- `PROJECTION_INTERVAL_SECONDS` - Processing interval (default: 10)

### Simple Query Server Configuration (JWT Enabled)
YAML configuration files mounted at `/configs`:

**Database Configuration** (`database.yaml`):
```yaml
type: "postgres"
dsn: "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable"
```

**Server Configuration** (`server.yaml`):
```yaml
# JWT/JWKS authentication middleware for account holder access
middleware:
  - type: "bearer-jwks"
    config:
      jwks_url: "http://jwks-mock-api:3000/.well-known/jwks.json"
      required: true
      fallback_ttl: "10m"
      enable_health_check: true
      claims_mapping:
        sub: "user_id"        # Map JWT 'sub' claim to 'user_id' parameter
        name: "user_name"     # Map JWT 'name' claim to 'user_name' parameter
      issuer: "http://jwks-mock-api:3000"
      audience: "dapr-actor-service"
```

**Queries Configuration** (`queries.yaml`):
```yaml
queries:
  my_transactions:
    sql: "SELECT * FROM transactions WHERE owner_id = :user_id ORDER BY transaction_timestamp DESC LIMIT :limit"
    middleware_params:        # ← JWT-provided parameters
      - name: user_id
        type: string
    params:                   # ← User-provided parameters
      - name: limit
        type: int
  
  my_account_balance:
    sql: "SELECT account_id, SUM(CASE WHEN transaction_type IN ('account_created', 'deposit') THEN amount ELSE -amount END) as balance FROM transactions WHERE owner_id = :user_id GROUP BY account_id ORDER BY balance DESC"
    middleware_params:
      - name: user_id
        type: string
  
  my_transaction_summary:
    sql: "SELECT transaction_type, COUNT(*) as count, SUM(amount) as total_amount FROM transactions WHERE owner_id = :user_id GROUP BY transaction_type ORDER BY count DESC"
    middleware_params:
      - name: user_id
        type: string
```

## Usage Examples (JWT Required)

All queries require valid JWT tokens for authentication.

### 1. Get JWT Token
```bash
# Generate JWT token for account holder
TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"claims": {"sub": "user-123"}, "expiresIn": 3600}' | \
  jq -r '.token')
```

### 2. Get Account Transactions
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"limit": 10}'
```

### 3. Get Account Balance
```bash
curl -X POST http://localhost:8081/query/my_account_balance \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
```

### 4. Get Transaction Summary
```bash
curl -X POST http://localhost:8081/query/my_transaction_summary \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
```
## Simple Query Server API

### Authentication
All endpoints require a valid JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

JWT `sub` claim is automatically mapped to `user_id` parameter for owner filtering.

## Query API Endpoints

### GET /queries
List all available queries:
```json
{
  "queries": [
    "my_transactions", 
    "my_account_balance", 
    "my_transaction_summary"
  ]
}
```

### POST /query/{query_name}
Execute a user-scoped query. JWT `sub` claim automatically provides `user_id` parameter.

**Request Example:**
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"limit": 5}'
```

**Response:**
```json
{
  "rows": [
    {
      "id": 1,
      "account_id": "account-demo-alice",
      "owner_id": "user-123",
      "transaction_type": "account_created",
      "amount": "1000.00",
      "description": "Account created with initial deposit",
      "transaction_timestamp": "2025-08-26T04:14:35.180657Z",
      "event_version": 1,
      "created_at": "2025-08-26T04:20:37.702225Z"
    }
  ]
}
```

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "database": {"connected": true}
}
```

## Available Account Holder Queries

The simple-query-server provides these JWT-authenticated, user-scoped queries:

- **`my_transactions`**: Get user's transactions with limit parameter (automatically filtered by JWT `sub` claim)
- **`my_account_balance`**: Calculate balance for all user's accounts (automatically filtered by JWT `sub` claim)
- **`my_transaction_summary`**: Count and sum transactions by type for user (automatically filtered by JWT `sub` claim)

All queries automatically filter results by the JWT `sub` claim mapped to `user_id` parameter.

## Modern CQRS Design

- **JWT Authentication**: Query endpoints require valid JWT tokens with automatic `user_id` mapping from JWT `sub` claim
- **Multi-Tenant Security**: Users can only access their own transaction data via owner-based filtering
- **Self-Contained Events**: Events include explicit `accountId` and `ownerId` fields for robust projection
- **Cursor-Based Consumption**: Uses go-simple-eventstore v0.0.9 for reliable event consumption and resumption
- **Optimistic Insert Idempotency**: Database unique constraints provide automatic reprocessing safety
- **Parameter Validation**: Query parameters are validated and type-checked for safety
- **SQL Injection Protection**: All queries use parameterized SQL with proper escaping

## Performance Considerations

- The projector processes events using go-simple-es-projector v0.0.2 for efficient cursor-based consumption
- Cursor-based event positioning with go-simple-eventstore v0.0.9 eliminates timestamp polling issues
- Optimistic insert idempotency via database unique constraints (no complex tracking required)
- Indexed columns: account_id, owner_id, transaction_type, transaction_timestamp, amount
- Simple-query-server v0.0.2 provides JWT-authenticated query execution with parameter validation
- Background database connection management with automatic retry and JWKS health monitoring

## Testing

Run the projection demo with JWT authentication:
```bash
./scripts/test-projection.sh
```

This script:
1. Creates test transactions via BankAccount actors (requires JWT tokens for actor commands)
2. Waits for projection to process events
3. Executes JWT-authenticated queries demonstrating user-scoped data access
4. Demonstrates the complete CQRS pattern working correctly

Example queries executed (all require JWT authentication):
- `POST /query/my_transactions` - List user's transactions (filtered by JWT `sub` claim)
- `POST /query/my_account_balance` - Calculate user's account balances (filtered by JWT `sub` claim)
- `POST /query/my_transaction_summary` - User's transaction counts by type (filtered by JWT `sub` claim)

## Integration with CQRS System

The modern projection system:
- ✅ Demonstrates complete CQRS architecture with separated command and query responsibilities
- ✅ Uses the same PostgreSQL database for consistency between command and query sides
- ✅ Processes events asynchronously without affecting actor performance
- ✅ Provides JWT-authenticated, user-scoped query capabilities while maintaining event sourcing benefits
- ✅ Can be deployed alongside existing services via Docker Compose
- ✅ Uses cursor-based event consumption for reliable resumption via go-simple-es-projector v0.0.2
- ✅ Leverages external simple-query-server v0.0.2 with JWT authentication and middleware parameter mapping
- ✅ Focuses on demonstrating modern CQRS patterns with secure, multi-tenant access control