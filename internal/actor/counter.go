package actor

import (
	"context"

	"github.com/dapr/go-sdk/actor"
)

// CounterActor represents a counter actor with state
type CounterActor struct {
	actor.ServerImplBaseCtx
}

// CounterState represents the state of the counter
type CounterState struct {
	Value int `json:"value"`
}

// SetValueRequest represents the request for setting a specific value
type SetValueRequest struct {
	Value int `json:"value"`
}

// Type returns the actor type name
func (c *CounterActor) Type() string {
	return "CounterActor"
}

// Increment increases the counter value by 1
func (c *CounterActor) Increment(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value++
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return state, nil
}

// Decrement decreases the counter value by 1
func (c *CounterActor) Decrement(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value--
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return state, nil
}

// Get returns the current counter value
func (c *CounterActor) Get(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

// Set sets the counter to a specific value
func (c *CounterActor) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	state := &CounterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return state, nil
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