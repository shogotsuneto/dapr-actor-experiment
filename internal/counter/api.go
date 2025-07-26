// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter

import (
	"context"
	"github.com/dapr/go-sdk/actor"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/shared"
)

// ActorTypeCounter is the Dapr actor type identifier for Counter
const ActorTypeCounter = "Counter"

// CounterAPI defines the interface that must be implemented to satisfy the OpenAPI schema for Counter.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type CounterAPI interface {
	actor.ServerContext
	// Decrement counter by 1
	Decrement(ctx context.Context) (*shared.CounterState, error)
	// Get current counter value
	Get(ctx context.Context) (*shared.CounterState, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*shared.CounterState, error)
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*shared.CounterState, error)
}