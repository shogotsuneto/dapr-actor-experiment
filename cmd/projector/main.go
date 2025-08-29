package main

import (
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/projection"
)

func main() {
	log.Println("Starting BankAccount Events Projector using cursor-based consumer...")

	// Configure postgres connection using environment variables with defaults
	connectionString := getEnvWithDefault("POSTGRES_CONNECTION_STRING", "postgres://postgres:postgres@postgres:5432/eventstore?sslmode=disable")
	eventsTableName := getEnvWithDefault("EVENTS_TABLE_NAME", "bankaccount_events")
	intervalSecondsStr := getEnvWithDefault("PROJECTION_INTERVAL_SECONDS", "10")

	intervalSeconds, err := strconv.Atoi(intervalSecondsStr)
	if err != nil {
		log.Fatalf("Invalid PROJECTION_INTERVAL_SECONDS: %v", err)
	}

	pollingInterval := time.Duration(intervalSeconds) * time.Second

	log.Printf("Configuration:")
	log.Printf("  - Database: %s", connectionString)
	log.Printf("  - Events Table: %s", eventsTableName)
	log.Printf("  - Polling Interval: %v", pollingInterval)

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

	// Create projector
	projector, err := projection.NewProjector(db, eventsTableName, connectionString, pollingInterval)
	if err != nil {
		log.Fatalf("Failed to create projector: %v", err)
	}
	defer func() {
		if err := projector.Close(); err != nil {
			log.Printf("Error closing projector: %v", err)
		}
	}()

	// Initialize schema
	if err := projector.InitSchema(); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Start projection (this will run in background)
	if err := projector.StartProjection(); err != nil {
		log.Fatalf("Failed to start projection: %v", err)
	}

	log.Println("Projector started successfully. Listening for events...")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping projector...")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}