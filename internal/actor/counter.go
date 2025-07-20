package actor

import (
	"context"
	"encoding/json"
	"fmt"

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