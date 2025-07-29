// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter

import (
	"context"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeCounter is the Dapr actor type identifier for Counter
const ActorTypeCounter = "Counter"

// CounterAPI defines the interface that must be implemented to satisfy the OpenAPI schema for Counter.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type CounterAPI interface {
	actor.ServerContext
	// Decrement counter by 1
	Decrement(ctx context.Context) (*CounterState, error)
	// Get current counter value
	Get(ctx context.Context) (*CounterState, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*CounterState, error)
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*CounterState, error)
	
	// JWT-aware methods for demonstrating user access
	// These methods accept userID to show how JWT subject can be passed to actors
	GetWithUser(ctx context.Context, userID string) (*CounterState, error)
	IncrementWithUser(ctx context.Context, userID string) (*CounterState, error)
}