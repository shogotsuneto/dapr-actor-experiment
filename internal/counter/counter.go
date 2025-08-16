package counter

import (
	"context"
	"fmt"
	"log"
	
	"github.com/dapr/go-sdk/actor"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
)



// Counter demonstrates schema-first development using generated OpenAPI types.
// It implements the generated CounterAPI interface to ensure compile-time schema compliance.
//
// Note: With the updated schema, all operations return structured responses within HTTP 200,
// with error information included in the response body instead of HTTP error codes.
type Counter struct {
	actor.ServerImplBaseCtx
}

func (c *Counter) Type() string {
	return ActorTypeCounter
}

func (c *Counter) Increment(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to get counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	state.Value++
	
	if err := c.setState(ctx, state); err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to save counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return c.successResponse(state.Value), nil
}

func (c *Counter) Decrement(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to get counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	state.Value--
	
	if err := c.setState(ctx, state); err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to save counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return c.successResponse(state.Value), nil
}

func (c *Counter) Get(ctx context.Context) (*CounterState, error) {
	state, err := c.getState(ctx)
	if err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to get counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return c.successResponse(state.Value), nil
}

func (c *Counter) Set(ctx context.Context, request SetValueRequest) (*CounterState, error) {
	// Example: Simple check - only authenticated users can set values
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		log.Printf("Counter %s: Unauthenticated user attempted Set operation", c.ID())
		return c.errorResponse(ErrorCodeAuthenticationError, "Authentication required for set operations", nil), nil
	}
	
	if err := c.validateSetRequest(request); err != nil {
		return c.errorResponse(ErrorCodeValidationError, err.Error(), map[string]interface{}{
			"requestedValue": request.Value,
		}), nil
	}
	
	state := &counterState{Value: request.Value}
	
	if err := c.setState(ctx, state); err != nil {
		return c.errorResponse(ErrorCodeInternalError, "Failed to save counter state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	log.Printf("Counter %s: Set operation completed by user %s", c.ID(), userID)
	return c.successResponse(state.Value), nil
}

// Helper methods

// counterState is the internal state representation
type counterState struct {
	Value int32 `json:"value"`
}

func (c *Counter) successResponse(value int32) *CounterState {
	// Return successful response with data nested under Data field
	return &CounterState{
		Success: true,
		Data: &CounterStateData{
			Value: value,
		},
		// Don't set Error field - omitempty will exclude it from JSON
	}
}

func (c *Counter) errorResponse(code ErrorCode, message string, details map[string]interface{}) *CounterState {
	return &CounterState{
		Success: false,
		Error: Error{
			Code:    code,
			Message: message,
			Details: details,
		},
		// Don't set Data field - omitempty will exclude it from JSON
	}
}

func (c *Counter) getState(ctx context.Context) (*counterState, error) {
	stateKey := "counter"
	var state counterState
	
	ok, err := c.GetStateManager().Contains(ctx, stateKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		return &counterState{Value: 0}, nil
	}
	
	err = c.GetStateManager().Get(ctx, stateKey, &state)
	if err != nil {
		return nil, err
	}
	
	return &state, nil
}

func (c *Counter) setState(ctx context.Context, state *counterState) error {
	stateKey := "counter"
	return c.GetStateManager().Set(ctx, stateKey, state)
}

func (c *Counter) validateSetRequest(request SetValueRequest) error {
	const (
		minInt32 = -2147483648
		maxInt32 = 2147483647
	)
	
	if request.Value < minInt32 || request.Value > maxInt32 {
		return fmt.Errorf("value %d out of range for int32 [%d, %d]", request.Value, minInt32, maxInt32)
	}
	
	return nil
}

