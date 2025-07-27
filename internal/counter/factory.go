// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

// NewActorFactory creates a factory function for Counter with a cleaner API.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// Usage: s.RegisterActorImplFactoryContext(counter.NewActorFactory())
func NewActorFactory() func() actor.ServerContext {
	return func() actor.ServerContext {
		// Create a new Counter instance
		impl := &Counter{}
		
		// Compile-time check ensures the implementation satisfies the schema
		var _ CounterAPI = impl
		
		// Verify the actor type matches the schema
		if impl.Type() != ActorTypeCounter {
			panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", impl.Type(), ActorTypeCounter))
		}
		
		return impl
	}
}