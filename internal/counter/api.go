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
	// Get current counter value
	Get(ctx context.Context) (*GetResponse, error)
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*SetResponse, error)
	// Decrement counter by 1
	Decrement(ctx context.Context) (*DecrementResponse, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*IncrementResponse, error)
}