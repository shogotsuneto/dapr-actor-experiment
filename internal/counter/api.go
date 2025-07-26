// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter

import (
	"context"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeCounterActor is the Dapr actor type identifier for CounterActor
const ActorTypeCounterActor = "CounterActor"

// CounterActorAPI defines the interface that must be implemented to satisfy the OpenAPI schema for CounterActor.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type CounterActorAPI interface {
	actor.ServerContext
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*CounterState, error)
	// Decrement counter by 1
	Decrement(ctx context.Context) (*CounterState, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*CounterState, error)
	// Get current counter value
	Get(ctx context.Context) (*CounterState, error)
}