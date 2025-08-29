package projection

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/go-simple-eventstore"
	"github.com/shogotsuneto/go-simple-eventstore/postgres"
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

// Projector handles event projection from bankaccount events to transactions table using cursor-based consumer
type Projector struct {
	db                   *sql.DB
	consumer             *postgres.PostgresEventConsumer
	processedEvents      map[string]bool // Track processed event IDs to ensure idempotency
	currentCursor        eventstore.Cursor // Current cursor position
	pollingInterval      time.Duration
	ctx                  context.Context
	cancel               context.CancelFunc
}

// NewProjector creates a new projector instance with cursor-based consumer
func NewProjector(db *sql.DB, eventsTableName, connectionString string, pollingInterval time.Duration) (*Projector, error) {
	// Create PostgresEventConsumer
	config := postgres.Config{
		ConnectionString: connectionString,
		TableName:        eventsTableName,
	}

	consumer, err := postgres.NewPostgresEventConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create event consumer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Projector{
		db:              db,
		consumer:        consumer,
		processedEvents: make(map[string]bool),
		currentCursor:   nil, // Start from beginning
		pollingInterval: pollingInterval,
		ctx:             ctx,
		cancel:          cancel,
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

// StartProjection starts the event projection process using cursor-based consumer
func (p *Projector) StartProjection() error {
	log.Println("Starting cursor-based event projection...")

	// Load already processed events to avoid reprocessing
	if err := p.loadProcessedEventIDs(); err != nil {
		log.Printf("Warning: Could not load processed events, starting fresh: %v", err)
	}

	// Load last cursor position for resumption
	if err := p.loadLastCursor(); err != nil {
		log.Printf("Warning: Could not load last cursor, starting from beginning: %v", err)
	}

	log.Printf("Starting projection from cursor position (length: %d bytes)", len(p.currentCursor))

	// Start processing events in a goroutine with polling
	go p.processEventsPeriodically()

	log.Println("Cursor-based event projection started successfully")
	return nil
}

// processEventsPeriodically processes events periodically using cursor-based fetching
func (p *Projector) processEventsPeriodically() {
	log.Println("Started periodic cursor-based event processing...")
	
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
		case <-p.ctx.Done():
			log.Println("Context cancelled, stopping event processing")
			return
		}
	}
}

// processNewEvents processes new events using cursor-based consumer
func (p *Projector) processNewEvents() error {
	const batchSize = 100

	// Fetch events from current cursor position
	batch, nextCursor, err := p.consumer.Fetch(p.ctx, p.currentCursor, batchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch events: %v", err)
	}

	if len(batch) == 0 {
		// No new events
		return nil
	}

	eventsProcessed := 0

	for _, envelope := range batch {
		// Check if we've already processed this event (idempotency)
		if envelope.EventID != "" && p.processedEvents[envelope.EventID] {
			log.Printf("Event %s already processed, skipping", envelope.EventID)
			continue
		}

		// Extract account ID from stream ID (format: bankaccount-<accountId>)
		accountID := extractAccountIDFromStreamID(envelope.StreamID)
		if accountID == "" {
			log.Printf("Warning: Could not extract account ID from stream %s", envelope.StreamID)
			continue
		}

		// Parse metadata (if present) - handle gracefully if not valid JSON
		var metadata map[string]string
		if envelope.Metadata != nil && len(envelope.Metadata) > 0 {
			if err := json.Unmarshal(envelope.Metadata, &metadata); err != nil {
				// Not all metadata may be JSON, log and continue
				log.Printf("Debug: Metadata not JSON for event %s: %v", envelope.EventID, string(envelope.Metadata))
			}
		}

		// Project the event based on type - extract version from metadata if available
		version := int64(1) // Default version
		if metadata != nil {
			if versionStr, exists := metadata["version"]; exists {
				if parsedVersion, err := strconv.ParseInt(versionStr, 10, 64); err == nil {
					version = parsedVersion
				}
			}
		}
		
		log.Printf("Debug: Processing event ID=%s, Type=%s, Account=%s, Version=%d", envelope.EventID, envelope.Type, accountID, version)
		
		transaction, err := p.projectEvent(accountID, version, envelope.Type, envelope.Data, envelope.CommitTime)
		if err != nil {
			log.Printf("Warning: Failed to project event %s: %v", envelope.EventID, err)
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
		if envelope.EventID != "" {
			p.processedEvents[envelope.EventID] = true
			if err := p.recordProcessedEventID(envelope.EventID); err != nil {
				log.Printf("Warning: Failed to record processed event ID %s: %v", envelope.EventID, err)
			}
		}

		eventsProcessed++
		
		log.Printf("Successfully processed event: ID=%s, Type=%s, Account=%s", envelope.EventID, envelope.Type, accountID)
	}

	// Update cursor position and commit
	if eventsProcessed > 0 {
		p.currentCursor = nextCursor
		if err := p.saveLastCursor(); err != nil {
			log.Printf("Warning: Failed to save last cursor: %v", err)
		}
		
		// Commit the cursor position
		if err := p.consumer.Commit(p.ctx, p.currentCursor); err != nil {
			log.Printf("Warning: Failed to commit cursor: %v", err)
		}
		
		log.Printf("Processed %d events, cursor advanced (%d bytes)", eventsProcessed, len(p.currentCursor))
	}

	return nil
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
	p.cancel() // Cancel the context to stop the background goroutine
	log.Println("Projector closed successfully")
	return nil
}

// loadLastCursor loads the last cursor position for resumption
func (p *Projector) loadLastCursor() error {
	// Create table to track last cursor position if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS projection_checkpoint (
			id INTEGER PRIMARY KEY DEFAULT 1,
			last_cursor BYTEA,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT single_row CHECK (id = 1)
		);
	`
	
	if _, err := p.db.Exec(createTableQuery); err != nil {
		return fmt.Errorf("failed to create projection_checkpoint table: %v", err)
	}

	// Insert default row if it doesn't exist
	insertDefaultQuery := `
		INSERT INTO projection_checkpoint (id, last_cursor) 
		VALUES (1, NULL) 
		ON CONFLICT (id) DO NOTHING
	`
	
	if _, err := p.db.Exec(insertDefaultQuery); err != nil {
		return fmt.Errorf("failed to insert default checkpoint: %v", err)
	}

	// Load the last cursor position
	var cursor []byte
	err := p.db.QueryRow("SELECT last_cursor FROM projection_checkpoint WHERE id = 1").Scan(&cursor)
	if err != nil {
		return fmt.Errorf("failed to query last cursor: %v", err)
	}

	p.currentCursor = eventstore.Cursor(cursor)
	log.Printf("Loaded last cursor (%d bytes)", len(p.currentCursor))
	return nil
}

// saveLastCursor saves the last cursor position
func (p *Projector) saveLastCursor() error {
	query := `
		UPDATE projection_checkpoint 
		SET last_cursor = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1
	`
	_, err := p.db.Exec(query, []byte(p.currentCursor))
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