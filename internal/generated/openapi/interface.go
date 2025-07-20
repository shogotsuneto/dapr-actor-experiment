// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import "context"

// CounterActorAPIContract defines the interface that must be implemented to satisfy the OpenAPI contract for CounterActor API.
// This interface enforces compile-time contract compliance.
type CounterActorAPIContract interface {
	// Set counter to specific value
	SetCounterValue(ctx context.Context, request SetValueRequest) (*CounterState, error)
	// Decrement counter by 1
	DecrementCounter(ctx context.Context) (*CounterState, error)
	// Get current counter value
	GetCounterValue(ctx context.Context) (*CounterState, error)
	// Increment counter by 1
	IncrementCounter(ctx context.Context) (*CounterState, error)
}