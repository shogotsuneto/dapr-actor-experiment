// Package counteractor provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counteractor

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

// NewActorFactory creates a factory function for CounterActor with a cleaner API.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// Usage: s.RegisterActorImplFactoryContext(counteractor.NewActorFactory())
func NewActorFactory() func() actor.ServerContext {
	return func() actor.ServerContext {
		// Create a new CounterActor instance
		impl := &CounterActor{}
		
		// Compile-time check ensures the implementation satisfies the schema
		var _ CounterActorAPI = impl
		
		// Verify the actor type matches the schema
		if impl.Type() != ActorTypeCounterActor {
			panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", impl.Type(), ActorTypeCounterActor))
		}
		
		return impl
	}
}