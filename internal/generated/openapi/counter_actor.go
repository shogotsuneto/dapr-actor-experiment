// Package generated provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import (
	"context"
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeCounterActor is the Dapr actor type identifier for CounterActor
const ActorTypeCounterActor = "CounterActor"

// CounterActorAPIContract defines the interface that must be implemented to satisfy the OpenAPI schema for CounterActor.
// This interface enforces compile-time schema compliance.
type CounterActorAPIContract interface {
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*CounterState, error)
	// Get current counter value
	Get(ctx context.Context) (*CounterState, error)
	// Decrement counter by 1
	Decrement(ctx context.Context) (*CounterState, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*CounterState, error)
}

// NewCounterActorFactoryContext creates a factory function for CounterActor with schema validation.
// The implementation parameter must implement CounterActorAPIContract interface.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// The generated factory ensures the actor Type() method returns the correct actor type.
func NewCounterActorFactoryContext(implementation func() CounterActorAPIContract) func() actor.ServerContext {
	return func() actor.ServerContext {
		// Compile-time check ensures the implementation satisfies the schema
		impl := implementation()
		
		// The implementation must also implement actor.ServerContext
		if serverCtx, ok := impl.(actor.ServerContext); ok {
			// Verify the actor type matches the schema
			if serverCtx.Type() != ActorTypeCounterActor {
				panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", serverCtx.Type(), ActorTypeCounterActor))
			}
			return serverCtx
		}
		
		// This should never happen if the actor is properly implemented
		panic("actor implementation must embed actor.ServerImplBaseCtx")
	}
}