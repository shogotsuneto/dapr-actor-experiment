# BankAccount Transactions Projection with JWT Authentication

This document explains the BankAccount transactions projection feature that demonstrates how account holders can securely query their own transaction data using JWT authentication.

## Overview

The projection system consists of two main components:

1. **Projector Worker** (`cmd/projector`) - Continuously reads events from `bankaccount_events` and projects them to a `transactions` table
2. **Simple Query Server with JWT Authentication** ([shogotsuneto/simple-query-server v0.0.2](https://github.com/shogotsuneto/simple-query-server)) - Provides secure, account holder-specific query endpoints

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  BankAccount    │───▶│  bankaccount_    │───▶│   Projector     │
│     Actors      │    │     events       │    │    Worker       │
│                 │    │   (Event Store)  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                          │
                                                          ▼
                                                ┌─────────────────┐
                                                │  transactions   │
                                                │     table       │
                                                │  (Projection)   │
                                                └─────────────────┘
                                                          │
                                                          ▼
┌─────────────────┐    ┌─────────────────┐              ┌─────────────────┐
│   JWKS Mock     │───▶│   JWT Token     │───────────── │ Simple Query    │
│     API         │    │ Authentication  │              │     Server      │
│  (Token Gen)    │    │   Middleware    │              │ (Account Holder │
└─────────────────┘    └─────────────────┘              │   Queries Only) │
                                                         └─────────────────┘
```

## Security Model

### JWT Authentication
- **Required Authentication**: All query endpoints require valid JWT tokens
- **Account Holder Access**: Users can only query their own transaction data
- **JWKS Verification**: Tokens are verified using the JWKS Mock API endpoint
- **Claim Mapping**: The JWT `sub` claim is mapped to `user_id` SQL parameter for data filtering

### Access Control
- Queries are automatically filtered by the authenticated user's ID (`user_id` from JWT `sub` claim)
- Cross-account access is prevented by SQL-level filtering
- No administrative or global queries are available to regular users

## Transactions Table Schema

The `transactions` table stores flattened transaction data:

```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id VARCHAR(255) NOT NULL,
    owner_id VARCHAR(255) NOT NULL,
    owner_name VARCHAR(255) NOT NULL,
    transaction_type VARCHAR(50) NOT NULL, -- 'account_created', 'deposit', 'withdrawal'
    amount DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    description TEXT,
    transaction_timestamp TIMESTAMPTZ NOT NULL,
    event_version BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(account_id, event_version) -- Ensure each event is projected once
);
```

## Event Projection

The projector transforms these event types:

### AccountCreatedV1 Events
```json
{
  "ownerName": "Alice Demo",
  "ownerId": "account-alice", 
  "initialDeposit": 1000.0,
  "createdAt": "2025-08-26T04:14:35Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, owner_name, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'account-alice', 'Alice Demo', 'account_created', 1000.00, 'Account created with initial deposit', '2025-08-26T04:14:35Z', 1);
```

### MoneyDepositedV1 Events  
```json
{
  "amount": 2500.0,
  "description": "Salary deposit",
  "timestamp": "2025-08-26T04:14:48Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, owner_name, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'account-alice', 'Alice Demo', 'deposit', 2500.00, 'Salary deposit', '2025-08-26T04:14:48Z', 2);
```

### MoneyWithdrawnV1 Events
```json
{
  "amount": 1200.0,
  "description": "Rent payment", 
  "timestamp": "2025-08-26T04:14:48Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, owner_id, owner_name, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'account-alice', 'Alice Demo', 'withdrawal', 1200.00, 'Rent payment', '2025-08-26T04:14:48Z', 3);
```

## Configuration

### Projector Configuration
Environment variables:
- `POSTGRES_CONNECTION_STRING` - Database connection string (default: postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable)
- `EVENTS_TABLE_NAME` - Event store table name (default: bankaccount_events)
- `PROJECTION_INTERVAL_SECONDS` - Processing interval (default: 10)

### Simple Query Server Configuration  
YAML configuration files mounted at `/configs`:

**Database Configuration** (`database.yaml`):
```yaml
type: "postgres"
dsn: "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable"
```

**Server Configuration** (`server.yaml`):
```yaml
middleware:
  - type: "bearer-jwks"
    config:
      jwks_url: "http://jwks-mock-api:3000/.well-known/jwks.json"
      required: true                                              
      fallback_ttl: "10m"                                         
      enable_health_check: true                                   
      claims_mapping:                                             
        sub: "user_id"                                            
        name: "user_name"                                         
      issuer: "http://jwks-mock-api:3000"                        
      audience: "dapr-actor-service"                             
```

**Queries Configuration** (`queries.yaml`):
```yaml
queries:
  my_transactions:
    sql: "SELECT * FROM transactions WHERE owner_id = :user_id ORDER BY transaction_timestamp DESC LIMIT :limit"
    params:
      - name: user_id
        type: string
      - name: limit
        type: int
  # ... additional account holder queries
```

## Usage Examples

### 1. Generate JWT Token
```bash
# Generate token for Alice
ALICE_TOKEN=$(curl -s http://localhost:3000/generate-token \
  -H "Content-Type: application/json" \
  -d '{"sub": "account-demo-alice", "name": "Alice Demo", "exp_minutes": 60}' | jq -r '.token')
```

### 2. Get My Transactions (Authenticated)
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}'
```

### 3. Get My Account Balance (Authenticated)
```bash
curl -X POST http://localhost:8081/query/my_account_balance \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 4. Get My Transaction Summary (Authenticated)
```bash
curl -X POST http://localhost:8081/query/my_transaction_summary \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 5. Unauthenticated Request (Will Fail)
```bash
# This will return 401 Unauthorized
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 5}'
```

## Simple Query Server API

### Authentication
All endpoints require a valid JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

### GET /queries
List all available account holder queries.

**Response:**
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
Execute an account holder query. The JWT `sub` claim automatically becomes the `user_id` parameter.

**Request Example:**
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"limit": 5}'
```

**Response:**
```json
{
  "rows": [
    {
      "id": 1,
      "account_id": "account-demo-alice",
      "owner_id": "account-demo-alice", 
      "owner_name": "Alice Demo",
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
Health check endpoint with JWKS middleware status.

**Response:**
```json
{
  "status": "healthy",
  "database": {"connected": true},
  "middleware": {
    "bearer-jwks(http://jwks-mock-api:3000/.well-known/jwks.json)": {
      "healthy": true
    }
  }
}
```

## Available Account Holder Queries

The simple-query-server provides these secure, account holder-specific queries:

- **`my_transactions`**: Get the authenticated user's transactions with limit parameter
- **`my_account_balance`**: Calculate balance for the authenticated user's accounts  
- **`my_transaction_summary`**: Count and sum transactions by type for the authenticated user

All queries automatically filter results by the authenticated user's ID (`user_id` from JWT `sub` claim).

## Security

- **JWT Authentication Required**: All query endpoints require valid JWT tokens
- **Account Holder Access Only**: Users can only query their own transaction data through automatic `user_id` filtering
- **JWKS Verification**: Tokens are verified using the JWKS Mock API endpoint at `/.well-known/jwks.json`
- **No Cross-Account Access**: SQL queries are filtered by `user_id` to prevent access to other users' data
- **Parameter Validation**: Query parameters are validated and type-checked for safety
- **SQL Injection Protection**: All queries use parameterized SQL with proper escaping
- **No Administrative Queries**: No global or admin-level queries are available to regular users

## Performance Considerations

- The projector processes events using cursor-based consumption for efficient resumption
- Event processing with configurable polling intervals (default: 10 seconds)
- Cursor-based positioning eliminates timestamp polling issues
- Indexed columns: account_id, owner_id, transaction_type, transaction_timestamp, amount
- Unique constraint prevents duplicate projections
- Simple-query-server provides optimized query execution with parameter validation
- Background database connection management with automatic retry

## Testing

Run the projection demo with JWT authentication:
```bash
./scripts/test-projection.sh
```

This script:
1. Generates JWT tokens for different users (Alice and Bob)
2. Creates test transactions via BankAccount actors using JWT authentication
3. Waits for projection to process events
4. Executes account holder queries using JWT tokens
5. Demonstrates security by showing cross-account access prevention

Example authenticated queries executed:
- `POST /query/my_transactions` - List user's transactions
- `POST /query/my_account_balance` - User's calculated account balance
- `POST /query/my_transaction_summary` - User's transaction counts by type
- Security demonstration showing unauthorized access attempts fail

## Integration with Existing System

The projection system:
- ✅ Does not modify existing BankAccount actor implementation
- ✅ Uses the same PostgreSQL database for consistency
- ✅ Processes events asynchronously without affecting actor performance
- ✅ Provides secure account holder query capabilities while maintaining event sourcing benefits
- ✅ Can be deployed alongside existing services via Docker Compose
- ✅ Uses cursor-based event consumption for reliable resumption
- ✅ Leverages external simple-query-server v0.0.2 with JWT authentication middleware
- ✅ Maintains backward compatibility with existing JWT authentication system