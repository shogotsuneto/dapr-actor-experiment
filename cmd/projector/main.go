package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/go-simple-es-projector"
	es "github.com/shogotsuneto/go-simple-eventstore"
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

func main() {
	log.Println("Starting BankAccount Events Projector using go-simple-es-projector...")

	// Configure postgres connection using environment variables with defaults
	connectionString := getEnvWithDefault("POSTGRES_CONNECTION_STRING", "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable")
	eventsTableName := getEnvWithDefault("EVENTS_TABLE_NAME", "bankaccount_events")
	intervalSecondsStr := getEnvWithDefault("PROJECTION_INTERVAL_SECONDS", "10")

	intervalSeconds, err := strconv.Atoi(intervalSecondsStr)
	if err != nil {
		log.Fatalf("Invalid PROJECTION_INTERVAL_SECONDS: %v", err)
	}

	idleSleep := time.Duration(intervalSeconds) * time.Second

	log.Printf("Configuration:")
	log.Printf("  - Database: %s", connectionString)
	log.Printf("  - Events Table: %s", eventsTableName)
	log.Printf("  - Idle Sleep: %v", idleSleep)

	// Connect to database for transactions table
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Test the connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")

	// Initialize schema
	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create event source consumer
	consumer, err := postgres.NewPostgresEventConsumer(postgres.Config{
		ConnectionString: connectionString,
		TableName:        eventsTableName,
	})
	if err != nil {
		log.Fatalf("Failed to create event consumer: %v", err)
	}

	// Load starting cursor from checkpoint
	ctx := context.Background()
	cursor, err := loadCursor(ctx, db)
	if err != nil {
		log.Fatalf("Failed to load cursor: %v", err)
	}

	log.Printf("Starting projection from cursor (%d bytes)", len(cursor))

	// Create and configure the worker
	worker := &projector.Worker{
		Source:    consumer,
		Start:     cursor,
		BatchSize: 100,
		IdleSleep: idleSleep,
		Apply:     createApplyFunc(db),
		Logger: func(msg string, kv ...any) {
			log.Printf("[PROJECTOR] %s %v", msg, kv)
		},
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("Projector started successfully. Listening for events...")
		errChan <- worker.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Shutdown signal received, stopping projector...")
		cancel()
		<-errChan // Wait for worker to stop
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			log.Fatalf("Projector error: %v", err)
		}
	}

	log.Println("Projector stopped successfully")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// initSchema initializes the transactions table and checkpoint tables
func initSchema(db *sql.DB) error {
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
CREATE INDEX IF NOT EXISTS idx_transactions_amount ON transactions(amount);

-- Checkpoint table for cursor persistence
CREATE TABLE IF NOT EXISTS projection_checkpoint (
    id INTEGER PRIMARY KEY DEFAULT 1,
    last_cursor BYTEA,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert default row if it doesn't exist
INSERT INTO projection_checkpoint (id, last_cursor) 
VALUES (1, NULL) 
ON CONFLICT (id) DO NOTHING;

`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %v", err)
	}

	log.Println("Database schema initialized successfully")
	return nil
}

// loadCursor loads the last cursor position from the checkpoint table
func loadCursor(ctx context.Context, db *sql.DB) (es.Cursor, error) {
	var cursor []byte
	err := db.QueryRowContext(ctx, "SELECT last_cursor FROM projection_checkpoint WHERE id = 1").Scan(&cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to load cursor: %v", err)
	}

	return es.Cursor(cursor), nil
}

// saveCursorTx saves the cursor within a transaction
func saveCursorTx(ctx context.Context, tx *sql.Tx, cursor es.Cursor) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE projection_checkpoint 
		SET last_cursor = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = 1`,
		[]byte(cursor))
	return err
}

// createApplyFunc creates the Apply function for the projector using optimistic inserts for idempotency
func createApplyFunc(db *sql.DB) projector.ApplyFunc {
	return func(ctx context.Context, batch []es.Envelope, next es.Cursor) error {
		// Begin transaction for atomic projection + checkpoint
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %v", err)
		}

		// Ensure transaction is cleaned up
		defer func() {
			if err != nil {
				_ = tx.Rollback()
			}
		}()

		eventsProcessed := 0

		// Process each event in the batch
		for _, envelope := range batch {
			// Extract account ID from stream ID (format: bankaccount-<accountId>)
			accountID := extractAccountIDFromStreamID(envelope.StreamID)
			if accountID == "" {
				log.Printf("Warning: Could not extract account ID from stream %s", envelope.StreamID)
				continue
			}

			// Parse event version from envelope offset
			version := int64(1) // Default version
			if envelope.Offset != "" {
				if parsedVersion, err := strconv.ParseInt(envelope.Offset, 10, 64); err == nil {
					version = parsedVersion
				} else {
					log.Printf("Debug: Failed to parse version from offset '%s': %v", envelope.Offset, err)
				}
			}

			log.Printf("Processing event: Stream=%s, Type=%s, Account=%s, Version=%d", envelope.StreamID, envelope.Type, accountID, version)

			transaction, err := projectEvent(accountID, version, envelope.Type, envelope.Data, envelope.CommitTime)
			if err != nil {
				log.Printf("Warning: Failed to project event %s: %v", envelope.EventID, err)
				continue
			}

			// Insert transaction into projection table if not nil
			if transaction != nil {
				// Handle owner name lookup for deposit/withdrawal events (simplified)
				if transaction.OwnerName == "" && transaction.OwnerID != "" {
					ownerName, err := lookupOwnerName(ctx, tx, accountID)
					if err != nil {
						log.Printf("Warning: Failed to lookup owner name for account %s: %v", accountID, err)
						// Continue with empty owner name rather than failing
					} else {
						transaction.OwnerName = ownerName
					}
				}

				// Insert transaction (database unique constraint provides idempotency)
				err = upsertTransactionTx(tx, transaction)
				if err != nil {
					return fmt.Errorf("failed to upsert transaction: %v", err)
				}
			}

			eventsProcessed++
			log.Printf("Successfully processed event: Stream=%s, Type=%s, Account=%s, Version=%d", envelope.StreamID, envelope.Type, accountID, version)
		}

		// Save the cursor to mark progress
		if err := saveCursorTx(ctx, tx, next); err != nil {
			return fmt.Errorf("failed to save cursor: %v", err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}

		if eventsProcessed > 0 {
			log.Printf("Successfully projected %d events, cursor advanced (%d bytes)", eventsProcessed, len(next))
		}

		return nil
	}
}

// extractAccountIDFromStreamID extracts account ID from stream ID like "bankaccount-<accountId>"
func extractAccountIDFromStreamID(streamID string) string {
	const prefix = "bankaccount-"
	if len(streamID) > len(prefix) && strings.HasPrefix(streamID, prefix) {
		accountID := streamID[len(prefix):]
		// Ensure it's not a snapshot stream
		if len(accountID) > 0 && !strings.HasSuffix(accountID, "-snapshots") {
			return accountID
		}
	}
	return ""
}

// projectEvent converts an event store event into a transaction projection
func projectEvent(accountID string, version int64, eventType string, eventData []byte, timestamp time.Time) (*TransactionProjection, error) {
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
			Description:          "Account created with initial deposit",
			TransactionTimestamp: event.CreatedAt,
			EventVersion:         version,
		}, nil

	case bankaccount.EventTypeMoneyDepositedV1:
		var event bankaccount.MoneyDepositedEventV1
		if err := json.Unmarshal(eventData, &event); err != nil {
			return nil, fmt.Errorf("failed to unmarshal MoneyDepositedV1: %v", err)
		}

		return &TransactionProjection{
			AccountID:            accountID,
			OwnerID:              event.OwnerId,
			OwnerName:            "", // Will be populated from account state lookup if needed
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

		return &TransactionProjection{
			AccountID:            accountID,
			OwnerID:              event.OwnerId,
			OwnerName:            "", // Will be populated from account state lookup if needed
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

// upsertTransactionTx optimistically inserts a transaction, relying on database unique constraint for idempotency
func upsertTransactionTx(tx *sql.Tx, transaction *TransactionProjection) error {
	query := `
		INSERT INTO transactions (account_id, owner_id, owner_name, transaction_type, amount, description, transaction_timestamp, event_version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (account_id, event_version) DO NOTHING
	`

	result, err := tx.Exec(query,
		transaction.AccountID,
		transaction.OwnerID,
		transaction.OwnerName,
		transaction.TransactionType,
		transaction.Amount,
		transaction.Description,
		transaction.TransactionTimestamp,
		transaction.EventVersion,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		log.Printf("Transaction already exists for account_id=%s, version=%d (idempotent)", transaction.AccountID, transaction.EventVersion)
	}

	return nil
}

// lookupOwnerName looks up the owner name from existing transactions for the given account
func lookupOwnerName(ctx context.Context, tx *sql.Tx, accountID string) (string, error) {
	var ownerName string
	err := tx.QueryRowContext(ctx, 
		"SELECT owner_name FROM transactions WHERE account_id = $1 AND owner_name != '' LIMIT 1", 
		accountID).Scan(&ownerName)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No owner name found, not an error
		}
		return "", err
	}
	return ownerName, nil
}

