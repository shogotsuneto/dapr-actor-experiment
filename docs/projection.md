# BankAccount Transactions Projection

This document explains the BankAccount transactions projection feature that demonstrates how to query list-like data from the event store.

## Overview

The projection system consists of two main components:

1. **Projector Worker** (`cmd/projector`) - Continuously reads events from `bankaccount_events` and projects them to a `transactions` table
2. **Query Server** (`cmd/query-server`) - Provides a SQL query interface with JSON results

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
                                                │  Query Server   │
                                                │ (SQL Interface) │
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
- `PROJECTION_INTERVAL_SECONDS` - Processing interval (default: 30)

### Query Server Configuration  
Environment variables:
- `POSTGRES_CONNECTION_STRING` - Database connection string
- `QUERY_SERVER_PORT` - HTTP server port (default: 8081)

## Usage Examples

### 1. Query All Transactions
```bash
curl -X POST http://localhost:8081/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT * FROM transactions ORDER BY transaction_timestamp DESC LIMIT 10"}'
```

### 2. Calculate Account Balances
```bash
curl -X POST http://localhost:8081/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT account_id, owner_name, SUM(CASE WHEN transaction_type IN ('"'"'account_created'"'"', '"'"'deposit'"'"') THEN amount ELSE -amount END) as balance FROM transactions GROUP BY account_id, owner_name ORDER BY balance DESC"}'
```

### 3. Transaction Summary by Type
```bash
curl -X POST http://localhost:8081/query \
  -H "Content-Type: application/json" \
  -d '{"sql": "SELECT transaction_type, COUNT(*) as count, SUM(amount) as total_amount FROM transactions GROUP BY transaction_type ORDER BY count DESC"}'
```

### 4. Web Interface
Visit http://localhost:8081 for an interactive web interface with example queries.

## Query Server API

### POST /query
Execute SQL queries on the transactions table.

**Request:**
```json
{
  "sql": "SELECT * FROM transactions LIMIT 5"
}
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "account_id": "account-alice",
      "owner_id": "account-alice", 
      "owner_name": "Alice Demo",
      "transaction_type": "account_created",
      "amount": "1000.00",
      "description": "Account created with initial deposit",
      "transaction_timestamp": "2025-08-26T04:14:35.180657Z",
      "event_version": 1,
      "created_at": "2025-08-26T04:20:37.702225Z"
    }
  ],
  "count": 1
}
```

### GET /examples
Get predefined example queries.

### GET /health
Health check endpoint.

### GET /
Interactive web interface for executing queries.

## Security

- Only SELECT statements are allowed
- Queries must reference the 'transactions' table
- SQL injection protection through input validation
- CORS headers enabled for browser access

## Performance Considerations

- The projector processes events in batches (default: 100 events)
- Indexed columns: account_id, owner_id, transaction_type, transaction_timestamp, amount
- Unique constraint prevents duplicate projections
- Projector tracks last processed event ID for efficient resumption

## Testing

Run the projection demo:
```bash
./scripts/test-projection.sh
```

This script:
1. Creates test transactions via BankAccount actors
2. Waits for projection to process events
3. Executes various example queries
4. Displays results in JSON format

## Integration with Existing System

The projection system:
- ✅ Does not modify existing BankAccount actor implementation
- ✅ Uses the same PostgreSQL database for consistency
- ✅ Processes events asynchronously without affecting actor performance
- ✅ Provides additional query capabilities while maintaining event sourcing benefits
- ✅ Can be deployed alongside existing services via Docker Compose