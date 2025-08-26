-- Transactions projection table
-- This table stores flattened transaction data from the bankaccount events
CREATE TABLE IF NOT EXISTS transactions (
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
    
    -- Add indexes for common query patterns
    UNIQUE(account_id, event_version) -- Ensure each event is projected once
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_owner_id ON transactions(owner_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(transaction_type);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(transaction_timestamp);
CREATE INDEX IF NOT EXISTS idx_transactions_amount ON transactions(amount);