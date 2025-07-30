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
	// Log JWT information for demonstration
	c.logJWTInfo(ctx, "Increment")
	
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
	// Log JWT information for demonstration
	c.logJWTInfo(ctx, "Decrement")
	
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
	// Log JWT information for demonstration
	c.logJWTInfo(ctx, "Get")
	
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	return state, nil
}

func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	// Log JWT information for demonstration
	c.logJWTInfo(ctx, "Set")
	
	// Example: Check if user has admin role for set operations
	if !auth.HasRole(ctx, "admin") && !auth.HasRole(ctx, "counter_admin") {
		userID := auth.GetUserIdentifier(ctx)
		log.Printf("Counter %s: User %s attempted Set operation without admin role", c.ID(), userID)
		return nil, errors.New("insufficient permissions: admin role required for set operations")
	}
	
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

// logJWTInfo logs JWT information for demonstration purposes
func (c *Counter) logJWTInfo(ctx context.Context, operation string) {
	claims, ok := auth.GetJWTClaims(ctx)
	if !ok {
		log.Printf("Counter %s: %s operation - No JWT claims found", c.ID(), operation)
		return
	}
	
	userID := auth.GetUserIdentifier(ctx)
	log.Printf("Counter %s: %s operation by user %s (subject: %s, username: %s, roles: %v)", 
		c.ID(), operation, userID, claims.Subject, claims.Username, claims.Roles)
}