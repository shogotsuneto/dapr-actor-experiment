package counteractor

import (
	"context"
	"errors"
	
	"github.com/dapr/go-sdk/actor"
)



// CounterActor demonstrates schema-first development using generated OpenAPI types.
// It implements the generated CounterActorAPI interface to ensure compile-time schema compliance.
//
// Note: Dapr actors return errors as strings through the HTTP layer, so custom error types
// with structured data cannot be returned directly. Use standard Go errors for actor methods.
type CounterActor struct {
	actor.ServerImplBaseCtx
}

func (c *CounterActor) Type() string {
	return ActorTypeCounterActor
}

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

func (c *CounterActor) Get(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *CounterActor) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	if err := c.validateSetRequest(request); err != nil {
		return nil, err
	}
	
	state := &CounterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *CounterActor) getState(ctx context.Context) (*CounterState, error) {
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

func (c *CounterActor) setState(ctx context.Context, state *CounterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
}

func (c *CounterActor) validateSetRequest(request SetValueRequest) error {
	const (
		minInt32 = -2147483648
		maxInt32 = 2147483647
	)
	
	if request.Value < minInt32 || request.Value > maxInt32 {
		return errors.New("value out of range for int32")
	}
	
	return nil
}