package counter

import (
	"context"
	"errors"
	"log"
	
	"github.com/dapr/go-sdk/actor"
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
	log.Printf("🔐 Counter.Increment() called - JWT info not directly accessible in actor methods")
	
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

// IncrementWithUser demonstrates passing JWT info to actor methods
func (c *Counter) IncrementWithUser(ctx context.Context, userID string) (*CounterState, error) {
	// Log the JWT subject (userId) within the actor method
	log.Printf("🔐 JWT Subject (userId) in Counter.IncrementWithUser(): %s", userID)
	
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	state.Value++
	log.Printf("📈 Counter incremented to %d by user: %s", state.Value, userID)
	
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
	// When called directly through actor invocation, JWT info is not directly accessible
	// However, you can create JWT-aware versions of methods that accept user context
	log.Printf("🔐 Counter.Get() called - JWT info not directly accessible in actor methods")
	log.Printf("💡 Use GetWithUser() method or service-level handlers for JWT access")
	
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

// GetWithUser demonstrates how to pass JWT information to actor methods
// This method can be called from service handlers that extract JWT info
func (c *Counter) GetWithUser(ctx context.Context, userID string) (*CounterState, error) {
	// Log the JWT subject (userId) as requested - this is what the user wanted to see
	log.Printf("🔐 JWT Subject (userId) accessed within Counter actor: %s", userID)
	
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	// You could add user-specific logic here, like:
	// - Check if user owns this counter instance
	// - Log user actions for audit trail
	// - Apply user-specific business rules
	
	log.Printf("📊 Counter value %d accessed by user: %s", state.Value, userID)
	
	return state, nil
}

func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	if err := c.validateSetRequest(request); err != nil {
		return nil, err
	}
	
	state := &CounterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
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