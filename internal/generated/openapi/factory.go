// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import (
	"github.com/dapr/go-sdk/actor"
)

// CounterActorAPIContractFactory provides a type-safe factory for creating actor implementations
// that comply with the CounterActorAPIContract interface.
type CounterActorAPIContractFactory[T interface {
	CounterActorAPIContract
	actor.ServerContext
}] struct{}

// NewCounterActorAPIContractFactory creates a new factory for the given implementation type.
// The implementation type T must satisfy both the CounterActorAPIContract interface and actor.ServerContext.
func NewCounterActorAPIContractFactory[T interface {
	CounterActorAPIContract
	actor.ServerContext
}]() *CounterActorAPIContractFactory[T] {
	return &CounterActorAPIContractFactory[T]{}
}

// CreateActorImplFactory returns a factory function that can be used with
// Dapr's RegisterActorImplFactoryContext method. This ensures compile-time
// type safety and contract compliance.
func (f *CounterActorAPIContractFactory[T]) CreateActorImplFactory(newImpl func() T) func() actor.ServerContext {
	return func() actor.ServerContext {
		impl := newImpl()
		// Compile-time check to ensure T implements both the contract and actor.ServerContext
		var _ CounterActorAPIContract = impl
		var _ actor.ServerContext = impl
		return impl
	}
}