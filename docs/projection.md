# BankAccount Transactions Projection

This document explains the BankAccount transactions projection feature that demonstrates how to query list-like data from the event store.

## Overview

The projection system consists of two main components:

1. **Projector Worker** (`cmd/projector`) - Continuously reads events from `bankaccount_events` and projects them to a `transactions` table
2. **Simple Query Server** ([shogotsuneto/simple-query-server](https://github.com/shogotsuneto/simple-query-server)) - Provides predefined query endpoints with parameter validation

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
                                                │ (REST API with  │
                                                │predefined queries)│
                                                └─────────────────┘
```

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

**Queries Configuration** (`queries.yaml`):
```yaml
queries:
  recent_transactions:
    sql: "SELECT * FROM transactions ORDER BY transaction_timestamp DESC LIMIT :limit"
    params:
      - name: limit
        type: int
  # ... additional predefined queries
```

## Usage Examples

### 1. List Available Queries
```bash
curl http://localhost:8081/queries
```

### 2. Get Recent Transactions
```bash
curl -X POST http://localhost:8081/query/recent_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}'
```

### 3. Calculate Account Balances
```bash
curl -X POST http://localhost:8081/query/account_balances \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 4. Transaction Summary by Type
```bash
curl -X POST http://localhost:8081/query/transaction_type_summary \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 5. Find Large Transactions
```bash
curl -X POST http://localhost:8081/query/large_transactions \
  -H "Content-Type: application/json" \
  -d '{"min_amount": 1000.0}'
```

### 6. Get Transactions for Specific Account
```bash
curl -X POST http://localhost:8081/query/account_transactions \
  -H "Content-Type: application/json" \
  -d '{"account_id": "account-demo-alice"}'
```

## Simple Query Server API

### GET /queries
List all available predefined queries.

**Response:**
```json
{
  "queries": [
    "recent_transactions",
    "account_balances", 
    "account_transactions",
    "transaction_type_summary",
    "daily_volume",
    "large_transactions",
    "transactions_by_owner",
    "transaction_count",
    "all_transactions",
    "transactions_by_type"
  ]
}
```

### POST /query/{query_name}
Execute a predefined query with parameters.

**Request Example:**
```bash
curl -X POST http://localhost:8081/query/recent_transactions \
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
Health check endpoint.

**Response:**
```json
{
  "database": {"connected": true},
  "status": "healthy"
}
```

## Available Predefined Queries

The simple-query-server provides these predefined queries:

- **`recent_transactions`**: Get the most recent transactions with limit parameter
- **`account_balances`**: Calculate balance per account from all transactions
- **`account_transactions`**: Get all transactions for a specific account ID
- **`transaction_type_summary`**: Count and sum transactions by type
- **`daily_volume`**: Show transaction volume grouped by day
- **`large_transactions`**: Find transactions over a specified amount threshold
- **`transactions_by_owner`**: Get transactions for a specific owner ID
- **`transaction_count`**: Get total number of transactions
- **`all_transactions`**: Get all transactions with optional limit
- **`transactions_by_type`**: Filter transactions by specific type

## Security

- Query execution is limited to predefined queries only (no arbitrary SQL)
- Parameter validation and type checking for query safety
- SQL injection protection through parameterized queries
- Database connection with authentication
- No direct SQL interface exposed to users

## Performance Considerations

- The projector processes events using cursor-based consumption for efficient resumption
- Event processing with configurable polling intervals (default: 10 seconds)
- Cursor-based positioning eliminates timestamp polling issues
- Indexed columns: account_id, owner_id, transaction_type, transaction_timestamp, amount
- Unique constraint prevents duplicate projections
- Simple-query-server provides optimized query execution with parameter validation
- Background database connection management with automatic retry

## Testing

Run the projection demo:
```bash
./scripts/test-projection.sh
```

This script:
1. Creates test transactions via BankAccount actors
2. Waits for projection to process events
3. Executes various predefined queries using the simple-query-server API
4. Displays results in JSON format

Example queries executed:
- `POST /query/all_transactions` - List all transactions
- `POST /query/transaction_type_summary` - Transaction counts by type
- `POST /query/account_balances` - Calculated account balances
- `POST /query/recent_transactions` - Most recent activity

## Integration with Existing System

The projection system:
- ✅ Does not modify existing BankAccount actor implementation
- ✅ Uses the same PostgreSQL database for consistency
- ✅ Processes events asynchronously without affecting actor performance
- ✅ Provides predefined query capabilities while maintaining event sourcing benefits
- ✅ Can be deployed alongside existing services via Docker Compose
- ✅ Uses cursor-based event consumption for reliable resumption
- ✅ Leverages external simple-query-server for robust query execution