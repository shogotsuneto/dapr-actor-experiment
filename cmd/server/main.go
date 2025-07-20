package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

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
	actorType := "basic"
	if os.Getenv("USE_CONTRACT_ACTOR") == "true" {
		actorType = "contract"
	}
	
	response := map[string]string{
		"status":     "running",
		"service":    "dapr-actor-demo",
		"actor_type": "CounterActor",
		"mode":       actorType,
		"description": getActorDescription(actorType),
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

// getActorDescription returns a description of the current actor mode
func getActorDescription(actorType string) string {
	switch actorType {
	case "contract":
		return "Using OpenAPI contract-generated types for type safety and contract compliance"
	default:
		return "Using basic implementation with manually defined types"
	}
}

func main() {
	// Create Dapr service
	s := daprd.NewService(":8080")
	
	// Register the appropriate actor based on environment variable
	useContractActor := os.Getenv("USE_CONTRACT_ACTOR") == "true"
	
	if useContractActor {
		log.Println("Using Contract-based CounterActor (OpenAPI generated types)")
		// Register the contract-based actor using generated OpenAPI types
		s.RegisterActorImplFactoryContext(func() actor.ServerContext {
			return &counteractor.ContractCounterActor{}
		})
	} else {
		log.Println("Using Basic CounterActor (manual types)")
		// Register the basic actor using manually defined types
		s.RegisterActorImplFactoryContext(func() actor.ServerContext {
			return &counteractor.CounterActor{}
		})
	}
	
	// Add health and status endpoints
	s.AddServiceInvocationHandler("/health", healthHandler)
	s.AddServiceInvocationHandler("/status", statusHandler)
	
	log.Printf("Starting Dapr Actor Service on port 8080 (mode: %s)...", getActorMode())
	log.Println("Environment variables:")
	log.Printf("  USE_CONTRACT_ACTOR=%s", os.Getenv("USE_CONTRACT_ACTOR"))
	log.Println("")
	log.Println("To use contract-based actor: USE_CONTRACT_ACTOR=true ./bin/server")
	log.Println("To use basic actor: ./bin/server")
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting service: %v", err)
	}
}

// getActorMode returns the current actor mode for logging
func getActorMode() string {
	if os.Getenv("USE_CONTRACT_ACTOR") == "true" {
		return "contract"
	}
	return "basic"
}