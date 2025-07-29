package counter

import (
	"context"
	"errors"
	"fmt"
	
	"github.com/dapr/go-sdk/actor"
)

// JWTAwareCounter demonstrates how JWT information can be used within actors
// This extends the base Counter with JWT-aware operations
type JWTAwareCounter struct {
	Counter
}

// JWTContext contains JWT information extracted from headers
type JWTContext struct {
	Subject string `json:"subject"`
	Role    string `json:"role"`
}

// GetWithOwnership demonstrates how actors can use JWT information for authorization
// Note: In practice, JWT information would need to be passed as method parameters
// since Dapr actors don't have direct access to HTTP headers
func (c *JWTAwareCounter) GetWithOwnership(ctx context.Context, jwtCtx JWTContext) (*CounterStateWithOwner, error) {
	// Get the current state
	state, err := c.getState(ctx)
	if err != nil {
		return nil, err
	}
	
	// Get ownership information from actor state
	ownership, err := c.getOwnership(ctx)
	if err != nil {
		return nil, err
	}
	
	// Check if the user has access to this counter
	if ownership.Owner != "" && ownership.Owner != jwtCtx.Subject {
		return nil, errors.New("access denied: you don't own this counter")
	}
	
	return &CounterStateWithOwner{
		Value:     state.Value,
		Owner:     ownership.Owner,
		CreatedBy: ownership.CreatedBy,
		AccessedBy: jwtCtx.Subject,
	}, nil
}

// SetWithOwnership sets the counter value and establishes ownership
func (c *JWTAwareCounter) SetWithOwnership(ctx context.Context, request SetValueWithOwnerRequest, jwtCtx JWTContext) (*CounterStateWithOwner, error) {
	if err := c.validateSetRequest(SetValueRequest{Value: request.Value}); err != nil {
		return nil, err
	}
	
	// Get current ownership
	ownership, err := c.getOwnership(ctx)
	if err != nil {
		return nil, err
	}
	
	// If counter has no owner, the current user becomes the owner
	if ownership.Owner == "" {
		ownership.Owner = jwtCtx.Subject
		ownership.CreatedBy = jwtCtx.Subject
	} else if ownership.Owner != jwtCtx.Subject {
		// Check if user has permission to modify this counter
		if !c.hasWritePermission(jwtCtx, ownership) {
			return nil, fmt.Errorf("access denied: user %s cannot modify counter owned by %s", jwtCtx.Subject, ownership.Owner)
		}
	}
	
	// Update the counter state
	state := &CounterState{Value: request.Value}
	if err := c.setState(ctx, state); err != nil {
		return nil, err
	}
	
	// Update ownership information
	if err := c.setOwnership(ctx, ownership); err != nil {
		return nil, err
	}
	
	return &CounterStateWithOwner{
		Value:     state.Value,
		Owner:     ownership.Owner,
		CreatedBy: ownership.CreatedBy,
		AccessedBy: jwtCtx.Subject,
	}, nil
}

// TransferOwnership allows transferring counter ownership (admin only)
func (c *JWTAwareCounter) TransferOwnership(ctx context.Context, request TransferOwnershipRequest, jwtCtx JWTContext) error {
	// Only admins can transfer ownership
	if jwtCtx.Role != "admin" {
		return errors.New("access denied: only admins can transfer ownership")
	}
	
	ownership, err := c.getOwnership(ctx)
	if err != nil {
		return err
	}
	
	ownership.Owner = request.NewOwner
	return c.setOwnership(ctx, ownership)
}

// hasWritePermission checks if a user can write to this counter
func (c *JWTAwareCounter) hasWritePermission(jwtCtx JWTContext, ownership *CounterOwnership) bool {
	// Owner always has write permission
	if ownership.Owner == jwtCtx.Subject {
		return true
	}
	
	// Admins have write permission to all counters
	if jwtCtx.Role == "admin" {
		return true
	}
	
	// Other roles don't have write permission by default
	return false
}

// getOwnership retrieves ownership information from actor state
func (c *JWTAwareCounter) getOwnership(ctx context.Context) (*CounterOwnership, error) {
	ownershipKey := "ownership"
	var ownership CounterOwnership
	
	ok, err := c.GetStateManager().Contains(ctx, ownershipKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		return &CounterOwnership{}, nil
	}
	
	err = c.GetStateManager().Get(ctx, ownershipKey, &ownership)
	if err != nil {
		return nil, err
	}
	
	return &ownership, nil
}

// setOwnership stores ownership information in actor state
func (c *JWTAwareCounter) setOwnership(ctx context.Context, ownership *CounterOwnership) error {
	ownershipKey := "ownership"
	return c.GetStateManager().Set(ctx, ownershipKey, ownership)
}

// CounterStateWithOwner extends CounterState with ownership information
type CounterStateWithOwner struct {
	Value      int32  `json:"value"`
	Owner      string `json:"owner"`
	CreatedBy  string `json:"created_by"`
	AccessedBy string `json:"accessed_by"`
}

// CounterOwnership stores ownership information
type CounterOwnership struct {
	Owner     string `json:"owner"`
	CreatedBy string `json:"created_by"`
}

// SetValueWithOwnerRequest includes ownership context
type SetValueWithOwnerRequest struct {
	Value int32 `json:"value"`
}

// TransferOwnershipRequest for transferring counter ownership
type TransferOwnershipRequest struct {
	NewOwner string `json:"new_owner"`
}