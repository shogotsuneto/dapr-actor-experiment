package actor

import (
	"context"
	"errors"
	
	"github.com/dapr/go-sdk/actor"
	generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

// ActorError implements Go's error interface for actor operations
type ActorError struct {
	Message string
	Code    string
}

func (e *ActorError) Error() string {
	return e.Message
}

// ToGenerated converts to the generated Error type for responses
func (e *ActorError) ToGenerated() *generated.Error {
	details := map[string]interface{}{}
	return &generated.Error{
		Error:   e.Message,
		Code:    e.Code,
		Details: details,
	}
}

// NewActorError creates an actor error
func NewActorError(message string, code string) *ActorError {
	return &ActorError{
		Message: message,
		Code:    code,
	}
}

// CounterActor demonstrates API-contract-first development using generated OpenAPI types.
// It implements the generated CounterActorContract interface to ensure compile-time contract compliance.
type CounterActor struct {
	actor.ServerImplBaseCtx
}

// Compile-time check to ensure CounterActor implements the generated contract interface
var _ generated.CounterActorAPIContract = (*CounterActor)(nil)

func (c *CounterActor) Type() string {
	return "CounterActor"
}

func (c *CounterActor) Increment(ctx context.Context) (*generated.CounterState, error) {
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

func (c *CounterActor) Decrement(ctx context.Context) (*generated.CounterState, error) {
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

func (c *CounterActor) Get(ctx context.Context) (*generated.CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *CounterActor) Set(ctx context.Context, request generated.SetValueRequest) (*generated.CounterState, error) {
	if err := c.validateSetRequest(request); err != nil {
		return nil, err
	}
	
	state := &generated.CounterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *CounterActor) getState(ctx context.Context) (*generated.CounterState, error) {
	stateKey := "counter"
	var state generated.CounterState
	
	ok, err := c.GetStateManager().Contains(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		return &generated.CounterState{Value: 0}, nil
	}
	
	err = c.GetStateManager().Get(ctx, stateKey, &state)
	if err != nil {
		return nil, err
	}
	
	return &state, nil
}

func (c *CounterActor) setState(ctx context.Context, state *generated.CounterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
}

func (c *CounterActor) validateSetRequest(request generated.SetValueRequest) error {
	const (
		minInt32 = -2147483648
		maxInt32 = 2147483647
	)
	
	if request.Value < minInt32 || request.Value > maxInt32 {
		return NewActorError("Value out of range for int32", "INVALID_INPUT")
	}
	
	return nil
}

// ValidateContract ensures this implementation satisfies the OpenAPI contract.
// This function provides runtime validation that the implementation follows
// the contract defined in the OpenAPI specification.
func ValidateContract() error {
	// The compile-time check above ensures interface compliance
	// This function can be extended for runtime validation if needed
	
	// Example: Validate that all required methods are implemented
	var actor CounterActor
	var contract generated.CounterActorAPIContract = &actor
	
	if contract == nil {
		return errors.New("CounterActor does not implement CounterActorAPIContract")
	}
	
	return nil
}