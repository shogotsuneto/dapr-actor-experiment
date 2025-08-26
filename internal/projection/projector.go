package projection

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/lib/pq"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
)

// TransactionProjection represents a projected transaction record
type TransactionProjection struct {
	ID                     int       `json:"id" db:"id"`
	AccountID              string    `json:"accountId" db:"account_id"`
	OwnerID                string    `json:"ownerId" db:"owner_id"`
	OwnerName              string    `json:"ownerName" db:"owner_name"`
	TransactionType        string    `json:"transactionType" db:"transaction_type"`
	Amount                 float64   `json:"amount" db:"amount"`
	Description            string    `json:"description" db:"description"`
	TransactionTimestamp   time.Time `json:"transactionTimestamp" db:"transaction_timestamp"`
	EventVersion           int64     `json:"eventVersion" db:"event_version"`
	CreatedAt              time.Time `json:"createdAt" db:"created_at"`
}

// Projector handles event projection from bankaccount events to transactions table
type Projector struct {
	db               *sql.DB
	eventsTableName  string
	lastProcessedID  int64
	batchSize        int
}

// NewProjector creates a new projector instance
func NewProjector(db *sql.DB, eventsTableName string) *Projector {
	return &Projector{
		db:               db,
		eventsTableName:  eventsTableName,
		lastProcessedID:  0,
		batchSize:        100,
	}
}

// InitSchema initializes the transactions table schema
func (p *Projector) InitSchema() error {
	schema := `
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
CREATE INDEX IF NOT EXISTS idx_transactions_amount ON transactions(amount);`

	_, err := p.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize transactions table schema: %v", err)
	}

	log.Println("Transactions table schema initialized successfully")
	return nil
}

// ProcessEvents processes new events from the event store and projects them to transactions table
func (p *Projector) ProcessEvents() error {
	// Query for new events since last processed
	query := fmt.Sprintf(`
		SELECT id, stream_id, version, event_type, event_data, metadata, timestamp 
		FROM %s 
		WHERE id > $1 
		ORDER BY id 
		LIMIT $2`, p.eventsTableName)

	rows, err := p.db.Query(query, p.lastProcessedID, p.batchSize)
	if err != nil {
		return fmt.Errorf("failed to query events: %v", err)
	}
	defer rows.Close()

	eventsProcessed := 0
	var maxProcessedID int64

	for rows.Next() {
		var eventID int64
		var streamID string
		var version int64
		var eventType string
		var eventData []byte
		var metadata []byte
		var timestamp time.Time

		err := rows.Scan(&eventID, &streamID, &version, &eventType, &eventData, &metadata, &timestamp)
		if err != nil {
			return fmt.Errorf("failed to scan event row: %v", err)
		}

		// Parse metadata to get account info
		var meta map[string]interface{}
		if err := json.Unmarshal(metadata, &meta); err != nil {
			log.Printf("Warning: Failed to parse metadata for event %d: %v", eventID, err)
			continue
		}

		actorID, ok := meta["actorId"].(string)
		if !ok {
			log.Printf("Warning: No actorId found in metadata for event %d", eventID)
			continue
		}

		// Project the event based on type
		transaction, err := p.projectEvent(actorID, version, eventType, eventData, timestamp)
		if err != nil {
			log.Printf("Warning: Failed to project event %d: %v", eventID, err)
			continue
		}

		// Insert transaction into projection table
		if transaction != nil {
			err = p.insertTransaction(transaction)
			if err != nil {
				// Check if it's a duplicate key error (event already processed)
				if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
					log.Printf("Event already projected: account_id=%s, version=%d", transaction.AccountID, transaction.EventVersion)
				} else {
					return fmt.Errorf("failed to insert transaction: %v", err)
				}
			}
		}

		eventsProcessed++
		maxProcessedID = eventID
	}

	// Update last processed ID
	if eventsProcessed > 0 {
		p.lastProcessedID = maxProcessedID
		log.Printf("Processed %d events, last ID: %d", eventsProcessed, maxProcessedID)
	}

	return rows.Err()
}

// projectEvent converts an event store event into a transaction projection
func (p *Projector) projectEvent(accountID string, version int64, eventType string, eventData []byte, timestamp time.Time) (*TransactionProjection, error) {
	switch bankaccount.EventTypeV1(eventType) {
	case bankaccount.EventTypeAccountCreatedV1:
		var event bankaccount.AccountCreatedEventV1
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal AccountCreatedV1: %v", err)
		}

		return &TransactionProjection{
			AccountID:            accountID,
			OwnerID:              event.OwnerId,
			OwnerName:            event.OwnerName,
			TransactionType:      "account_created",
			Amount:               event.InitialDeposit,
			Description:          fmt.Sprintf("Account created with initial deposit"),
			TransactionTimestamp: event.CreatedAt,
			EventVersion:         version,
		}, nil

	case bankaccount.EventTypeMoneyDepositedV1:
		var event bankaccount.MoneyDepositedEventV1
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MoneyDepositedV1: %v", err)
		}

		// Need to get owner info from previous events or cache
		ownerID, ownerName, err := p.getAccountOwnerInfo(accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get owner info for account %s: %v", accountID, err)
		}

		return &TransactionProjection{
			AccountID:            accountID,
			OwnerID:              ownerID,
			OwnerName:            ownerName,
			TransactionType:      "deposit",
			Amount:               event.Amount,
			Description:          event.Description,
			TransactionTimestamp: event.Timestamp,
			EventVersion:         version,
		}, nil

	case bankaccount.EventTypeMoneyWithdrawnV1:
		var event bankaccount.MoneyWithdrawnEventV1
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MoneyWithdrawnV1: %v", err)
		}

		// Need to get owner info from previous events or cache
		ownerID, ownerName, err := p.getAccountOwnerInfo(accountID)
		if err != nil {
			return nil, fmt.Errorf("failed to get owner info for account %s: %v", accountID, err)
		}

		return &TransactionProjection{
			AccountID:            accountID,
			OwnerID:              ownerID,
			OwnerName:            ownerName,
			TransactionType:      "withdrawal",
			Amount:               event.Amount,
			Description:          event.Description,
			TransactionTimestamp: event.Timestamp,
			EventVersion:         version,
		}, nil

	case bankaccount.EventTypeStateSnapshotV1:
		// Skip snapshot events as they don't represent transactions
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}
}

// getAccountOwnerInfo retrieves owner information for an account from existing transactions
func (p *Projector) getAccountOwnerInfo(accountID string) (string, string, error) {
	var ownerID, ownerName string
	query := `SELECT owner_id, owner_name FROM transactions WHERE account_id = $1 LIMIT 1`
	
	err := p.db.QueryRow(query, accountID).Scan(&ownerID, &ownerName)
	if err != nil {
		return "", "", fmt.Errorf("owner info not found for account %s: %v", accountID, err)
	}

	return ownerID, ownerName, nil
}

// insertTransaction inserts a transaction projection into the database
func (p *Projector) insertTransaction(tx *TransactionProjection) error {
	query := `
		INSERT INTO transactions (account_id, owner_id, owner_name, transaction_type, amount, description, transaction_timestamp, event_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := p.db.Exec(query,
		tx.AccountID,
		tx.OwnerID,
		tx.OwnerName,
		tx.TransactionType,
		tx.Amount,
		tx.Description,
		tx.TransactionTimestamp,
		tx.EventVersion,
	)

	return err
}

// GetLastProcessedEventID returns the last processed event ID for resumption
func (p *Projector) GetLastProcessedEventID() (int64, error) {
	// Get the highest event ID that corresponds to events already projected
	query := `
		SELECT COALESCE(MAX(e.id), 0) 
		FROM ` + p.eventsTableName + ` e
		INNER JOIN transactions t ON e.stream_id = t.account_id AND e.version = t.event_version
	`
	
	var lastID int64
	err := p.db.QueryRow(query).Scan(&lastID)
	if err != nil {
		return 0, fmt.Errorf("failed to get last processed event ID: %v", err)
	}

	return lastID, nil
}

// SetLastProcessedEventID sets the last processed event ID
func (p *Projector) SetLastProcessedEventID(id int64) {
	p.lastProcessedID = id
}