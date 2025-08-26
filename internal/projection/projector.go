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
	db                   *sql.DB
	eventsTableName      string
	processedEvents      map[string]bool // Track processed event IDs to ensure idempotency
	lastProcessedTime    time.Time
	pollingInterval      time.Duration
}

// NewProjector creates a new projector instance
func NewProjector(db *sql.DB, eventsTableName string, pollingInterval time.Duration) (*Projector, error) {
	return &Projector{
		db:               db,
		eventsTableName:  eventsTableName,
		processedEvents:  make(map[string]bool),
		lastProcessedTime: time.Time{}, // Start from beginning
		pollingInterval:  pollingInterval,
	}, nil
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

// StartProjection starts the event projection process
func (p *Projector) StartProjection() error {
	log.Println("Starting event projection...")

	// Load already processed events to avoid reprocessing
	if err := p.loadProcessedEventIDs(); err != nil {
		log.Printf("Warning: Could not load processed events, starting fresh: %v", err)
	}

	// Load last processed timestamp for resumption
	if err := p.loadLastProcessedTime(); err != nil {
		log.Printf("Warning: Could not load last processed time, starting from beginning: %v", err)
	}

	log.Printf("Starting projection from timestamp: %v", p.lastProcessedTime)

	// Start processing events in a goroutine with polling
	go p.processEventsPeriodically()

	log.Println("Event projection started successfully")
	return nil
}

// processEventsPeriodically processes events periodically using timestamp-based querying
func (p *Projector) processEventsPeriodically() {
	log.Println("Started periodic event processing...")
	
	// Process immediately, then periodically
	if err := p.processNewEvents(); err != nil {
		log.Printf("Error in initial event processing: %v", err)
	}

	ticker := time.NewTicker(p.pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.processNewEvents(); err != nil {
				log.Printf("Error processing events: %v", err)
			}
		}
	}
}

// processNewEvents processes new events from the timestamp where we left off
func (p *Projector) processNewEvents() error {
	// Query for events newer than our last processed time
	// This mimics the approach used by PostgresEventConsumer but includes stream_id
	query := fmt.Sprintf(`
		SELECT event_id, event_type, event_data, metadata, timestamp, version, stream_id
		FROM %s 
		WHERE timestamp >= $1 
		ORDER BY timestamp ASC, id ASC 
		LIMIT 100`, p.eventsTableName)

	rows, err := p.db.Query(query, p.lastProcessedTime)
	if err != nil {
		return fmt.Errorf("failed to query events: %v", err)
	}
	defer rows.Close()

	eventsProcessed := 0
	var latestTimestamp time.Time

	for rows.Next() {
		var eventID, eventType, streamID string
		var eventData []byte
		var metadataJSON []byte
		var timestamp time.Time
		var version int64

		err := rows.Scan(&eventID, &eventType, &eventData, &metadataJSON, &timestamp, &version, &streamID)
		if err != nil {
			return fmt.Errorf("failed to scan event row: %v", err)
		}

		// Check if we've already processed this event (idempotency)
		if p.processedEvents[eventID] {
			log.Printf("Event %s already processed, skipping", eventID)
			continue
		}

		// Parse metadata
		var metadata map[string]string
		if metadataJSON != nil {
			if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
				log.Printf("Warning: Failed to parse metadata for event %s: %v", eventID, err)
				continue
			}
		}

		// Extract account ID from stream ID (format: bankaccount-<accountId>)
		accountID := extractAccountIDFromStreamID(streamID)
		if accountID == "" {
			log.Printf("Warning: Could not extract account ID from stream %s", streamID)
			continue
		}

		// Project the event based on type
		transaction, err := p.projectEvent(accountID, version, eventType, eventData, timestamp)
		if err != nil {
			log.Printf("Warning: Failed to project event %s: %v", eventID, err)
			continue
		}

		// Insert transaction into projection table if not nil
		if transaction != nil {
			err = p.insertTransaction(transaction)
			if err != nil {
				// Check if it's a duplicate key error (event already processed)
				if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
					log.Printf("Transaction already exists for account_id=%s, version=%d", transaction.AccountID, transaction.EventVersion)
				} else {
					return fmt.Errorf("failed to insert transaction: %v", err)
				}
			}
		}

		// Mark event as processed
		p.processedEvents[eventID] = true
		if err := p.recordProcessedEventID(eventID); err != nil {
			log.Printf("Warning: Failed to record processed event ID %s: %v", eventID, err)
		}

		eventsProcessed++
		latestTimestamp = timestamp
		
		log.Printf("Successfully processed event: ID=%s, Type=%s, Account=%s", eventID, eventType, accountID)
	}

	// Update last processed timestamp
	if eventsProcessed > 0 {
		p.lastProcessedTime = latestTimestamp
		if err := p.saveLastProcessedTime(); err != nil {
			log.Printf("Warning: Failed to save last processed time: %v", err)
		}
		log.Printf("Processed %d events, last timestamp: %v", eventsProcessed, latestTimestamp)
	}

	return rows.Err()
}

// extractAccountIDFromStreamID extracts account ID from stream ID like "bankaccount-<accountId>"
func extractAccountIDFromStreamID(streamID string) string {
	const prefix = "bankaccount-"
	if len(streamID) > len(prefix) && streamID[:len(prefix)] == prefix {
		accountID := streamID[len(prefix):]
		// Ensure it's not a snapshot stream
		if len(accountID) > 0 && accountID[len(accountID)-10:] != "-snapshots" {
			return accountID
		}
	}
	return ""
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

// Close closes the projector and cleans up resources
func (p *Projector) Close() error {
	log.Println("Closing projector...")
	log.Println("Projector closed successfully")
	return nil
}

// loadLastProcessedTime loads the last processed timestamp for resumption
func (p *Projector) loadLastProcessedTime() error {
	// Create table to track last processed time if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS projection_checkpoint (
			id INTEGER PRIMARY KEY DEFAULT 1,
			last_processed_time TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT single_row CHECK (id = 1)
		);
	`
	
	if _, err := p.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create projection_checkpoint table: %v", err)
	}

	// Insert default row if it doesn't exist
	insertDefaultQuery := `
		INSERT INTO projection_checkpoint (id, last_processed_time) 
		VALUES (1, '1970-01-01T00:00:00Z') 
		ON CONFLICT (id) DO NOTHING
	`
	
	if _, err := p.db.Exec(insertDefaultQuery); err != nil {
		return fmt.Errorf("failed to insert default checkpoint: %v", err)
	}

	// Load the last processed time
	var lastTime time.Time
	err := p.db.QueryRow("SELECT last_processed_time FROM projection_checkpoint WHERE id = 1").Scan(&lastTime)
	if err != nil {
		return fmt.Errorf("failed to query last processed time: %v", err)
	}

	p.lastProcessedTime = lastTime
	log.Printf("Loaded last processed time: %v", lastTime)
	return nil
}

// saveLastProcessedTime saves the last processed timestamp
func (p *Projector) saveLastProcessedTime() error {
	query := `
		UPDATE projection_checkpoint 
		SET last_processed_time = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1
	`
	_, err := p.db.Exec(query, p.lastProcessedTime)
	return err
}

// loadProcessedEventIDs loads previously processed event IDs from the database for idempotency
func (p *Projector) loadProcessedEventIDs() error {
	// Create a table to track processed events if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS processed_events (
			event_id VARCHAR(255) PRIMARY KEY,
			processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`
	
	if _, err := p.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create processed_events table: %v", err)
	}

	// Load recent processed event IDs (last 24 hours to keep memory usage reasonable)
	// We rely on timestamp-based resumption as the primary mechanism
	cutoffTime := time.Now().Add(-24 * time.Hour)
	rows, err := p.db.Query("SELECT event_id FROM processed_events WHERE processed_at > $1", cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to query recent processed events: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var eventID string
		if err := rows.Scan(&eventID); err != nil {
			return fmt.Errorf("failed to scan processed event ID: %v", err)
		}
		p.processedEvents[eventID] = true
		count++
	}

	log.Printf("Loaded %d recent processed event IDs for idempotency", count)
	return rows.Err()
}

// recordProcessedEventID records an event ID as processed for idempotency
func (p *Projector) recordProcessedEventID(eventID string) error {
	query := "INSERT INTO processed_events (event_id) VALUES ($1) ON CONFLICT (event_id) DO NOTHING"
	_, err := p.db.Exec(query, eventID)
	return err
}