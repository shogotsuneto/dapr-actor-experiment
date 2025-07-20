// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import "context"

// CounterActorContract defines the interface that must be implemented
// to satisfy the OpenAPI contract for CounterActor.
// This interface enforces compile-time contract compliance.
type CounterActorContract interface {
	// Set counter to specific value
	Set(ctx context.Context, request SetValueRequest) (*CounterState, error)
	// Decrement counter by 1
	Decrement(ctx context.Context) (*CounterState, error)
	// Get current counter value
	Get(ctx context.Context) (*CounterState, error)
	// Increment counter by 1
	Increment(ctx context.Context) (*CounterState, error)
}
