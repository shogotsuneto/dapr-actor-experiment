package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

// healthHandler provides a simple health check endpoint
func healthHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	out = &common.Content{
		Data:        []byte("OK"),
		ContentType: "text/plain",
	}
	return
}

// statusHandler provides status information about the actor service
func statusHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	response := map[string]interface{}{
		"status":      "running",
		"service":     "dapr-actor-demo",
		"actor_types": []string{counter.ActorTypeCounter, bankaccount.ActorTypeBankAccount},
		"description": "Multi-actor service demonstrating state-based and event-sourced patterns",
		"patterns": map[string]string{
			counter.ActorTypeCounter:     "State-based - stores current value only",
			bankaccount.ActorTypeBankAccount: "Event-sourced - stores events and computes state",
		},
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

func main() {
	// Create Chi router with middleware
	r := chi.NewRouter()
	
	// Add basic middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	
	// Configure JWT middleware
	jwtConfig := auth.JWTMiddlewareConfig{
		SecretKey: []byte(getJWTSecret()),
		RequiredIssuer: getJWTIssuer(),
		SkipPaths: []string{
			"/health",
			"/status",
			"/v1.0/healthz", // Dapr health check
		},
		AllowInsecure: isInsecureMode(),
	}
	
	// Add JWT middleware for all routes except skip paths
	r.Use(auth.JWTMiddleware(jwtConfig))
	
	// Create Dapr service with custom router
	s := daprd.NewServiceWithMux(":8080", r)
	
	// Register Counter using generated factory with contract enforcement
	log.Printf("Registering %s with state-based pattern", counter.ActorTypeCounter)
	s.RegisterActorImplFactoryContext(counter.NewActorFactory())
	
	// Register BankAccount using generated factory with contract enforcement
	log.Printf("Registering %s with event sourcing pattern", bankaccount.ActorTypeBankAccount)
	s.RegisterActorImplFactoryContext(bankaccount.NewActorFactory())
	
	// Add health and status endpoints
	s.AddServiceInvocationHandler("/health", healthHandler)
	s.AddServiceInvocationHandler("/status", statusHandler)
	
	log.Println("Starting JWT-aware Multi-Actor Dapr Service on port 8080...")
	log.Printf("JWT Configuration:")
	log.Printf("  - Secret: %s", maskSecret(getJWTSecret()))
	log.Printf("  - Issuer: %s", getJWTIssuer())
	log.Printf("  - Insecure mode: %t", isInsecureMode())
	log.Printf("Actors registered:")
	log.Printf("  - %s: State-based counter operations (JWT-aware)", counter.ActorTypeCounter)
	log.Printf("  - %s: Event-sourced bank account with full audit trail (JWT-aware)", bankaccount.ActorTypeBankAccount)
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}

// getJWTSecret returns the JWT secret key from environment or default
func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	// Use default test secret (never use in production)
	return auth.DefaultTestSecret
}

// getJWTIssuer returns the JWT issuer from environment or default
func getJWTIssuer() string {
	if issuer := os.Getenv("JWT_ISSUER"); issuer != "" {
		return issuer
	}
	return auth.DefaultTestIssuer
}

// isInsecureMode returns true if insecure mode is enabled (for testing)
func isInsecureMode() bool {
	return os.Getenv("JWT_INSECURE_MODE") == "true"
}

// maskSecret masks a secret for logging
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}