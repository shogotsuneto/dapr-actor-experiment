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
)

// CounterActor represents a counter actor with state
type CounterActor struct {
	actor.ServerImplBaseCtx
}

// CounterState represents the state of the counter
type CounterState struct {
	Value int `json:"value"`
}

// Type returns the actor type name
func (c *CounterActor) Type() string {
	return "CounterActor"
}

// InvokeMethod handles method invocations on the actor
func (c *CounterActor) InvokeMethod(ctx context.Context, methodName string, request []byte) ([]byte, error) {
	switch methodName {
	case "increment":
		return c.increment(ctx)
	case "decrement":
		return c.decrement(ctx)
	case "get":
		return c.get(ctx)
	case "set":
		return c.set(ctx, request)
	default:
		return nil, fmt.Errorf("method %s not found", methodName)
	}
}

// increment increases the counter value by 1
func (c *CounterActor) increment(ctx context.Context) ([]byte, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value++
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return json.Marshal(state)
}

// decrement decreases the counter value by 1
func (c *CounterActor) decrement(ctx context.Context) ([]byte, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value--
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return json.Marshal(state)
}

// get returns the current counter value
func (c *CounterActor) get(ctx context.Context) ([]byte, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return json.Marshal(state)
}

// set sets the counter to a specific value
func (c *CounterActor) set(ctx context.Context, request []byte) ([]byte, error) {
	var setValue struct {
		Value int `json:"value"`
	}
	
	if err := json.Unmarshal(request, &setValue); err != nil {
		return nil, err
	}
	
	state := &CounterState{Value: setValue.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return json.Marshal(state)
}

// getState retrieves the current state from Dapr state store
func (c *CounterActor) getState(ctx context.Context) (*CounterState, error) {
	stateKey := "counter"
	var state CounterState
	
	ok, err := c.GetStateManager().Contains(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		// Initialize with default value if state doesn't exist
		return &CounterState{Value: 0}, nil
	}
	
	err = c.GetStateManager().Get(ctx, stateKey, &state)
	if err != nil {
		return nil, err
	}
	
	return &state, nil
}

// setState saves the current state to Dapr state store
func (c *CounterActor) setState(ctx context.Context, state *CounterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
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
		return &CounterActor{}
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