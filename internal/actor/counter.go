package actor

import (
	"context"
	"errors"
	"fmt"
	
	"github.com/dapr/go-sdk/actor"
	generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

// ContractError implements Go's error interface for contract compliance
type ContractError struct {
	Message string
	Code    string
}

func (e *ContractError) Error() string {
	return e.Message
}

// ToGenerated converts to the generated Error type for responses
func (e *ContractError) ToGenerated() *generated.Error {
	details := map[string]interface{}{}
	return &generated.Error{
		Error:   e.Message,
		Code:    &e.Code,
		Details: &details,
	}
}

// NewContractError creates a contract-compliant error
func NewContractError(message string, code string) *ContractError {
	return &ContractError{
		Message: message,
		Code:    code,
	}
}

// CounterActor demonstrates API-contract-first development using generated OpenAPI types
type CounterActor struct {
	actor.ServerImplBaseCtx
}

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
		return NewContractError("Value out of range for int32", "INVALID_INPUT")
	}
	
	return nil
}

// ValidateContract ensures this implementation satisfies the OpenAPI contract
func ValidateContract() error {
	// This function demonstrates compile-time contract validation
	var actor CounterActor
	
	// Verify method signatures match the expected contract
	var _ func(context.Context) (*generated.CounterState, error) = actor.Increment
	var _ func(context.Context) (*generated.CounterState, error) = actor.Decrement
	var _ func(context.Context) (*generated.CounterState, error) = actor.Get
	var _ func(context.Context, generated.SetValueRequest) (*generated.CounterState, error) = actor.Set
	
	return nil
}

// ExampleUsage demonstrates type-safe usage of generated types
func ExampleUsage() error {
	// Example of using generated types for type-safe operations
	request := generated.SetValueRequest{Value: 42}
	
	// Type safety prevents invalid operations
	if request.Value < 0 {
		return errors.New("negative values not allowed in this example")
	}
	
	// Response must be the contract-defined type
	response := &generated.CounterState{Value: request.Value}
	
	fmt.Printf("Request: %+v, Response: %+v\n", request, response)
	return nil
}