package counter

import (
	"context"
	"errors"
	"log"
	
	"github.com/dapr/go-sdk/actor"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
)



// Counter demonstrates schema-first development using generated OpenAPI types.
// It implements the generated CounterAPI interface to ensure compile-time schema compliance.
//
// Note: Dapr actors return errors as strings through the HTTP layer, so custom error types
// with structured data cannot be returned directly. Use standard Go errors for actor methods.
type Counter struct {
	actor.ServerImplBaseCtx
}

func (c *Counter) Type() string {
	return ActorTypeCounter
}

func (c *Counter) Increment(ctx context.Context) (*CounterState, error) {
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

func (c *Counter) Decrement(ctx context.Context) (*CounterState, error) {
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

func (c *Counter) Get(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	// Example: Simple check - only authenticated users can set values
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		log.Printf("Counter %s: Unauthenticated user attempted Set operation", c.ID())
		return nil, errors.New("authentication required for set operations")
	}
	
	if err := c.validateSetRequest(request); err != nil {
		return nil, err
	}
	
	state := &CounterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	log.Printf("Counter %s: Set operation completed by user %s", c.ID(), userID)
	return state, nil
}

func (c *Counter) getState(ctx context.Context) (*CounterState, error) {
	stateKey := "counter"
	var state CounterState
	
	ok, err := c.GetStateManager().Contains(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		return &CounterState{Value: 0}, nil
	}
	
	err = c.GetStateManager().Get(ctx, stateKey, &state)
	if err != nil {
		return nil, err
	}
	
	return &state, nil
}

func (c *Counter) setState(ctx context.Context, state *CounterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
}

func (c *Counter) validateSetRequest(request SetValueRequest) error {
	const (
		minInt32 = -2147483648
		maxInt32 = 2147483647
	)
	
	if request.Value < minInt32 || request.Value > maxInt32 {
		return errors.New("value out of range for int32")
	}
	
	return nil
}

