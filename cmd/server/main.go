package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

// jwtAwareCounterGetHandler demonstrates the pattern for accessing JWT info
// Note: This is a service invocation handler, not an actor method
// In practice, Bearer middleware forwards JWT info differently for service vs actor calls
func jwtAwareCounterGetHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	log.Printf("🔐 Service handler called - this is where JWT headers would be accessible")
	log.Printf("💡 For actor calls, JWT is validated by Bearer middleware but headers are not directly accessible in actors")
	
	// In a real implementation, JWT headers would be available here (not in InvocationEvent.Metadata)
	// You would extract them and then call actor methods with the user information
	
	response := map[string]interface{}{
		"message": "Service handler demonstrates JWT access pattern",
		"note": "JWT validation works for actors, but subject access requires service-mediated calls",
		"pattern": "Service extracts JWT -> Calls actor with user info",
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

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
	// Create Dapr service
	s := daprd.NewService(":8080")
	
	// Register Counter using generated factory with contract enforcement
	log.Printf("Registering %s with state-based pattern", counter.ActorTypeCounter)
	s.RegisterActorImplFactoryContext(counter.NewActorFactory())
	
	// Register BankAccount using generated factory with contract enforcement
	log.Printf("Registering %s with event sourcing pattern", bankaccount.ActorTypeBankAccount)
	s.RegisterActorImplFactoryContext(bankaccount.NewActorFactory())
	
	// Add health and status endpoints
	s.AddServiceInvocationHandler("/health", healthHandler)
	s.AddServiceInvocationHandler("/status", statusHandler)
	
	// Add JWT-aware endpoint to demonstrate accessing JWT subject
	s.AddServiceInvocationHandler("/jwt-demo", jwtAwareCounterGetHandler)
	
	log.Println("Starting Multi-Actor Dapr Service on port 8080...")
	log.Printf("Actors registered:")
	log.Printf("  - %s: State-based counter operations", counter.ActorTypeCounter)
	log.Printf("  - %s: Event-sourced bank account with full audit trail", bankaccount.ActorTypeBankAccount)
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}