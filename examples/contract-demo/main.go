package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/actor"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	
	// Import the generated OpenAPI contract types
	generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

// ContractDemoActor demonstrates contract-first development
// All method signatures must match the generated OpenAPI contract
type ContractDemoActor struct {
	actor.ServerImplBaseCtx
}

// Type returns the actor type name - must match OpenAPI specification
func (c *ContractDemoActor) Type() string {
	return "CounterActor"
}

// Increment method using generated contract types
func (c *ContractDemoActor) Increment(ctx context.Context) (*generated.CounterState, error) {
	log.Printf("[CONTRACT] Increment called for actor ID: %s", c.ID())
	
	state, err := c.getContractState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value++
	
	if err := c.setContractState(ctx, state); err != nil {
		return nil, err
	}
	
	log.Printf("[CONTRACT] Incremented to: %d", state.Value)
	return state, nil
}

// Decrement method using generated contract types
func (c *ContractDemoActor) Decrement(ctx context.Context) (*generated.CounterState, error) {
	log.Printf("[CONTRACT] Decrement called for actor ID: %s", c.ID())
	
	state, err := c.getContractState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value--
	
	if err := c.setContractState(ctx, state); err != nil {
		return nil, err
	}
	
	log.Printf("[CONTRACT] Decremented to: %d", state.Value)
	return state, nil
}

// Get method using generated contract types
func (c *ContractDemoActor) Get(ctx context.Context) (*generated.CounterState, error) {
	log.Printf("[CONTRACT] Get called for actor ID: %s", c.ID())
	
	state, err := c.getContractState(ctx)
	if err != nil {
		return nil, err
	}
	
	log.Printf("[CONTRACT] Current value: %d", state.Value)
	return state, nil
}

// Set method using generated contract types
func (c *ContractDemoActor) Set(ctx context.Context, request generated.SetValueRequest) (*generated.CounterState, error) {
	log.Printf("[CONTRACT] Set called with value %d for actor ID: %s", request.Value, c.ID())
	
	// Create new state with the requested value
	state := &generated.CounterState{Value: request.Value}
	
	if err := c.setContractState(ctx, state); err != nil {
		return nil, err
	}
	
	log.Printf("[CONTRACT] Set to: %d", state.Value)
	return state, nil
}

// getContractState retrieves state using generated contract types
func (c *ContractDemoActor) getContractState(ctx context.Context) (*generated.CounterState, error) {
	stateKey := "counter"
	var state generated.CounterState
	
	ok, err := c.GetStateManager().Contains(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		// Return default state as defined in contract (value: 0)
		return &generated.CounterState{Value: 0}, nil
	}
	
	err = c.GetStateManager().Get(ctx, stateKey, &state)
	if err != nil {
		return nil, err
	}
	
	return &state, nil
}

// setContractState saves state using generated contract types
func (c *ContractDemoActor) setContractState(ctx context.Context, state *generated.CounterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
}

// contractInfoHandler shows information about the contract
func contractInfoHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	info := map[string]interface{}{
		"service":     "contract-demo",
		"actor_type":  "CounterActor",
		"contract":    "OpenAPI 3.0",
		"schema_file": "api-generation/schemas/openapi/counter-actor.yaml",
		"generated_types": []string{
			"generated.CounterState",
			"generated.SetValueRequest",
			"generated.Error",
		},
		"features": []string{
			"Type-safe method signatures",
			"Contract-compliant error handling",
			"Generated request/response types",
			"Automatic JSON serialization",
		},
		"methods": map[string]interface{}{
			"get": map[string]string{
				"http_method": "GET",
				"returns":     "CounterState",
			},
			"increment": map[string]string{
				"http_method": "POST",
				"returns":     "CounterState",
			},
			"decrement": map[string]string{
				"http_method": "POST",
				"returns":     "CounterState",
			},
			"set": map[string]string{
				"http_method": "POST",
				"accepts":     "SetValueRequest",
				"returns":     "CounterState",
			},
		},
	}
	
	data, _ := json.MarshalIndent(info, "", "  ")
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

// schemaHandler returns the OpenAPI schema
func schemaHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	// In a real implementation, you might embed the schema file
	// or read it from the filesystem
	schema := map[string]string{
		"message": "OpenAPI schema available at api-generation/schemas/openapi/counter-actor.yaml",
		"info":    "This endpoint would typically serve the actual OpenAPI specification",
		"note":    "Generated types ensure implementation matches the schema",
	}
	
	data, _ := json.MarshalIndent(schema, "", "  ")
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

func main() {
	fmt.Println("=== Contract-First Development Demo ===")
	fmt.Println("This server demonstrates API-contract-first development")
	fmt.Println("using generated OpenAPI types for type safety and contract compliance.")
	fmt.Println("")
	
	// Create Dapr service
	s := daprd.NewService(":8080")
	
	// Register the contract-based actor
	s.RegisterActorImplFactoryContext(func() actor.ServerContext {
		return &ContractDemoActor{}
	})
	
	// Add contract information endpoints
	s.AddServiceInvocationHandler("/contract-info", contractInfoHandler)
	s.AddServiceInvocationHandler("/schema", schemaHandler)
	
	// Add health endpoint
	s.AddServiceInvocationHandler("/health", func(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
		out = &common.Content{
			Data:        []byte(`{"status": "healthy", "type": "contract-demo"}`),
			ContentType: "application/json",
		}
		return
	})
	
	fmt.Println("Contract Demo Server starting on port 8080...")
	fmt.Println("")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET  /health        - Health check")
	fmt.Println("  GET  /contract-info - Contract information")
	fmt.Println("  GET  /schema        - OpenAPI schema info")
	fmt.Println("")
	fmt.Println("Actor API (via Dapr sidecar on port 3500):")
	fmt.Println("  GET  /v1.0/actors/CounterActor/{id}/method/get")
	fmt.Println("  POST /v1.0/actors/CounterActor/{id}/method/increment")
	fmt.Println("  POST /v1.0/actors/CounterActor/{id}/method/decrement") 
	fmt.Println("  POST /v1.0/actors/CounterActor/{id}/method/set")
	fmt.Println("")
	fmt.Println("Example usage:")
	fmt.Println("  curl http://localhost:3500/v1.0/actors/CounterActor/demo-1/method/get")
	fmt.Println("  curl -X POST http://localhost:3500/v1.0/actors/CounterActor/demo-1/method/increment")
	fmt.Println("")
	
	// Start the service
	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Error starting contract demo service: %v", err)
	}
}