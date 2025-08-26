package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/projection"
)

func main() {
	log.Println("Starting BankAccount Events Projector...")

	// Configure postgres connection using environment variables with defaults
	connectionString := getEnvWithDefault("POSTGRES_CONNECTION_STRING", "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable")
	eventsTableName := getEnvWithDefault("EVENTS_TABLE_NAME", "bankaccount_events")
	intervalSecondsStr := getEnvWithDefault("PROJECTION_INTERVAL_SECONDS", "30")

	intervalSeconds, err := strconv.Atoi(intervalSecondsStr)
	if err != nil {
		log.Fatalf("Invalid PROJECTION_INTERVAL_SECONDS: %v", err)
	}

	log.Printf("Configuration:")
	log.Printf("  - Database: %s", connectionString)
	log.Printf("  - Events Table: %s", eventsTableName)
	log.Printf("  - Processing Interval: %d seconds", intervalSeconds)

	// Connect to database
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

	// Create projector
	projector := projection.NewProjector(db, eventsTableName)

	// Initialize schema
	if err := projector.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Get last processed event ID for resumption
	lastProcessedID, err := projector.GetLastProcessedEventID()
	if err != nil {
		log.Printf("Warning: Could not determine last processed event ID, starting from 0: %v", err)
		lastProcessedID = 0
	}
	projector.SetLastProcessedEventID(lastProcessedID)
	log.Printf("Starting projection from event ID: %d", lastProcessedID)

	// Process events in a loop
	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Println("Projector started. Processing events...")

	// Process once immediately
	if err := projector.ProcessEvents(); err != nil {
		log.Printf("Error processing events: %v", err)
	}

	// Then process periodically
	for {
		select {
		case <-ticker.C:
			if err := projector.ProcessEvents(); err != nil {
				log.Printf("Error processing events: %v", err)
			}
		}
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}