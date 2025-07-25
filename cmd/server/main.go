package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccountactor"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counteractor"
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
		"actor_types": []string{counteractor.ActorTypeCounterActor, bankaccountactor.ActorTypeBankAccountActor},
		"description": "Multi-actor service demonstrating state-based and event-sourced patterns",
		"patterns": map[string]string{
			counteractor.ActorTypeCounterActor:     "State-based - stores current value only",
			bankaccountactor.ActorTypeBankAccountActor: "Event-sourced - stores events and computes state",
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
	
	// Register CounterActor using generated factory with contract enforcement
	log.Printf("Registering %s with state-based pattern", counteractor.ActorTypeCounterActor)
	s.RegisterActorImplFactoryContext(counteractor.NewActorFactory())
	
	// Register BankAccountActor using generated factory with contract enforcement
	log.Printf("Registering %s with event sourcing pattern", bankaccountactor.ActorTypeBankAccountActor)
	s.RegisterActorImplFactoryContext(bankaccountactor.NewActorFactory())
	
	// Add health and status endpoints
	s.AddServiceInvocationHandler("/health", healthHandler)
	s.AddServiceInvocationHandler("/status", statusHandler)
	
	log.Println("Starting Multi-Actor Dapr Service on port 8080...")
	log.Printf("Actors registered:")
	log.Printf("  - %s: State-based counter operations", counteractor.ActorTypeCounterActor)
	log.Printf("  - %s: Event-sourced bank account with full audit trail", bankaccountactor.ActorTypeBankAccountActor)
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}