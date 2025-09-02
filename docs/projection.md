# BankAccount Transactions Projection - Simplified

This document explains the simplified BankAccount transactions projection feature that demonstrates how events can be projected to queryable tables using go-simple-es-projector.

## Overview

The projection system consists of two main components:

1. **Projector Worker** (`cmd/projector`) - Uses go-simple-es-projector to continuously read events from `bankaccount_events` and projects them to a `transactions` table
2. **Simple Query Server** ([shogotsuneto/simple-query-server v0.0.2](https://github.com/shogotsuneto/simple-query-server)) - Provides account-based query endpoints without authentication

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
                                               ┌─────────────────┐
                                               │ Simple Query    │
                                               │     Server      │
                                               │ (Account-based  │
                                               │    Queries)     │
                                               └─────────────────┘
```

## Simplified Design

### No Authentication Required
- Query endpoints accept `account_id` parameter directly
- Simplified for experimental demonstration of projection patterns
- Focus on event sourcing and projection design patterns rather than security

### Account-Based Queries
- Queries filter by `account_id` parameter
- No user management or JWT authentication complexity
- Direct account access for demonstration purposes

## Transactions Table Schema

The `transactions` table stores simplified transaction data:

```sql
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id VARCHAR(255) NOT NULL,
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
  "ownerId": "account-alice", 
  "initialDeposit": 1000.0,
  "createdAt": "2025-08-26T04:14:35Z"
}
```
↓ Projects to:
```sql
INSERT INTO transactions (account_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'account_created', 1000.00, 'Account created with initial deposit', '2025-08-26T04:14:35Z', 1);
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
INSERT INTO transactions (account_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'deposit', 2500.00, 'Salary deposit', '2025-08-26T04:14:48Z', 2);
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
INSERT INTO transactions (account_id, transaction_type, amount, description, transaction_timestamp, event_version)
VALUES ('account-alice', 'withdrawal', 1200.00, 'Rent payment', '2025-08-26T04:14:48Z', 3);
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
# Simple configuration without authentication for experimental purposes
# No middleware - simplified for experimental demonstration
```

**Queries Configuration** (`queries.yaml`):
```yaml
queries:
  my_transactions:
    sql: "SELECT * FROM transactions WHERE account_id = :account_id ORDER BY transaction_timestamp DESC LIMIT :limit"
    params:
      - name: account_id
        type: string
      - name: limit
        type: int
  my_account_balance:
    sql: "SELECT account_id, SUM(CASE WHEN transaction_type IN ('account_created', 'deposit') THEN amount ELSE -amount END) as balance FROM transactions WHERE account_id = :account_id GROUP BY account_id"
    params:
      - name: account_id
        type: string
  my_transaction_summary:
    sql: "SELECT transaction_type, COUNT(*) as count, SUM(amount) as total_amount FROM transactions WHERE account_id = :account_id GROUP BY transaction_type ORDER BY count DESC"
    params:
      - name: account_id
        type: string
```

## Usage Examples

### 1. Get Account Transactions
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account-demo-alice", "limit": 10}'
```

### 2. Get Account Balance
```bash
curl -X POST http://localhost:8081/query/my_account_balance \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account-demo-alice"}'
```

### 3. Get Transaction Summary
```bash
curl -X POST http://localhost:8081/query/my_transaction_summary \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account-demo-alice"}'
```
```

## Simple Query Server API

### Authentication
All endpoints require a valid JWT token in the Authorization header:
```
Authorization: Bearer <jwt_token>
```

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
Execute an account-based query by providing the `account_id` parameter.

**Request Example:**
```bash
curl -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account-demo-alice", "limit": 5}'
```

**Response:**
```json
{
  "rows": [
    {
      "id": 1,
      "account_id": "account-demo-alice",
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

## Available Account-Based Queries

The simple-query-server provides these account-based queries:

- **`my_transactions`**: Get account transactions with limit parameter
- **`my_account_balance`**: Calculate balance for a specific account  
- **`my_transaction_summary`**: Count and sum transactions by type for a specific account

All queries filter results by the provided `account_id` parameter.

## Simplified Design

- **No Authentication**: Query endpoints accept direct `account_id` parameter  
- **Account-Based Access**: Queries filter by `account_id` for demonstration purposes
- **Focused on Patterns**: Emphasizes event sourcing and projection design patterns
- **Parameter Validation**: Query parameters are validated and type-checked for safety
- **SQL Injection Protection**: All queries use parameterized SQL with proper escaping

## Performance Considerations

- The projector processes events using go-simple-es-projector for efficient cursor-based consumption
- Event processing with configurable polling intervals (default: 10 seconds)
- Cursor-based positioning eliminates timestamp polling issues
- Indexed columns: account_id, transaction_type, transaction_timestamp, amount
- Unique constraint prevents duplicate projections
- Simple-query-server provides optimized query execution with parameter validation
- Background database connection management with automatic retry

## Testing

Run the simplified projection demo:
```bash
./scripts/test-projection.sh
```

This script:
1. Creates test transactions via BankAccount actors (JWT tokens still needed for actors)
2. Waits for projection to process events
3. Executes account-based queries without authentication
4. Demonstrates the projection pattern working correctly

Example queries executed:
- `POST /query/my_transactions` - List account transactions
- `POST /query/my_account_balance` - Calculate account balance
- `POST /query/my_transaction_summary` - Account transaction counts by type

## Integration with Existing System

The simplified projection system:
- ✅ Does not modify existing BankAccount actor implementation
- ✅ Uses the same PostgreSQL database for consistency
- ✅ Processes events asynchronously without affecting actor performance
- ✅ Provides account-based query capabilities while maintaining event sourcing benefits
- ✅ Can be deployed alongside existing services via Docker Compose
- ✅ Uses cursor-based event consumption for reliable resumption via go-simple-es-projector
- ✅ Leverages external simple-query-server v0.0.2 with simplified configuration
- ✅ Focuses on demonstrating projection patterns without authentication complexity