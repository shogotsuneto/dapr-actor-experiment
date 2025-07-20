package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/actor"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	
	counteractor "github.com/shogotsuneto/dapr-actor-experiment/internal/actor"
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
	response := map[string]string{
		"status":     "running",
		"service":    "dapr-actor-demo",
		"actor_type": "CounterActor",
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
	
	// Register the actor using FactoryContext
	s.RegisterActorImplFactoryContext(func() actor.ServerContext {
		return &counteractor.CounterActor{}
	})
	
	// Add health and status endpoints
	s.AddServiceInvocationHandler("/health", healthHandler)
	s.AddServiceInvocationHandler("/status", statusHandler)
	
	log.Println("Starting Dapr Actor Service on port 8080...")
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}