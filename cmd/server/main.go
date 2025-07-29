package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

var jwtMiddleware *auth.JWTMiddleware

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
		"authentication": "JWT validation disabled",
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

// jwtStatusHandler provides status information about the actor service with JWT validation info
func jwtStatusHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	response := map[string]interface{}{
		"status":      "running",
		"service":     "dapr-actor-demo",
		"actor_types": []string{counter.ActorTypeCounter, bankaccount.ActorTypeBankAccount},
		"description": "Multi-actor service demonstrating state-based and event-sourced patterns with JWT authentication",
		"patterns": map[string]string{
			counter.ActorTypeCounter:     "State-based - stores current value only",
			bankaccount.ActorTypeBankAccount: "Event-sourced - stores events and computes state",
		},
		"authentication": "JWT validation enabled",
		"jwks_url": jwtMiddleware.GetJWKSUrl(),
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

func main() {
	// JWT configuration from environment variables
	jwksUrl := os.Getenv("JWKS_URL")
	if jwksUrl == "" {
		jwksUrl = "http://jwks-server:3000/.well-known/jwks.json" // Default for Docker Compose
	}
	
	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "http://localhost:3000" // Default issuer from JWKS mock server
	}
	
	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		jwtAudience = "dev-api" // Default audience from JWKS mock server
	}

	// Initialize JWT middleware
	jwtConfig := &auth.JWTConfig{
		JWKSUrl:  jwksUrl,
		Issuer:   jwtIssuer,
		Audience: jwtAudience,
	}
	
	var err error
	jwtMiddleware, err = auth.NewJWTMiddleware(jwtConfig)
	if err != nil {
		log.Printf("Warning: Failed to initialize JWT middleware: %v", err)
		log.Println("Continuing without JWT validation. Set JWKS_URL environment variable to enable JWT validation.")
		jwtMiddleware = nil
	} else {
		log.Printf("JWT validation enabled with JWKS URL: %s", jwksUrl)
		defer jwtMiddleware.Close()
	}

	// Create Dapr service
	s := daprd.NewService(":8080")
	
	// Register Counter using generated factory with contract enforcement
	log.Printf("Registering %s with state-based pattern", counter.ActorTypeCounter)
	s.RegisterActorImplFactoryContext(counter.NewActorFactory())
	
	// Register BankAccount using generated factory with contract enforcement
	log.Printf("Registering %s with event sourcing pattern", bankaccount.ActorTypeBankAccount)
	s.RegisterActorImplFactoryContext(bankaccount.NewActorFactory())
	
	// Add health and status endpoints (no JWT required)
	s.AddServiceInvocationHandler("/health", healthHandler)
	
	if jwtMiddleware != nil {
		s.AddServiceInvocationHandler("/status", jwtStatusHandler)
	} else {
		s.AddServiceInvocationHandler("/status", statusHandler)
	}
	
	log.Println("Starting Multi-Actor Dapr Service on port 8080...")
	log.Printf("Actors registered:")
	log.Printf("  - %s: State-based counter operations", counter.ActorTypeCounter)
	log.Printf("  - %s: Event-sourced bank account with full audit trail", bankaccount.ActorTypeBankAccount)
	
	if jwtMiddleware != nil {
		log.Println("JWT validation is ENABLED for actor endpoints (handled by sidecar)")
		log.Printf("JWKS URL: %s", jwksUrl)
		log.Printf("Expected Issuer: %s", jwtIssuer)
		log.Printf("Expected Audience: %s", jwtAudience)
		log.Println("NOTE: JWT validation applies to actor endpoints via Dapr sidecar configuration")
	} else {
		log.Println("JWT validation is DISABLED")
	}
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}