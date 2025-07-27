// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter


import (
	"github.com/shogotsuneto/dapr-actor-experiment/internal/shared"
)



// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}

// GetResponse Response from get operation
type GetResponse struct {
	shared.CounterState
}

// SetResponse Response from set operation
type SetResponse struct {
	shared.CounterState
}

// DecrementResponse Response from decrement operation
type DecrementResponse struct {
	shared.CounterState
}

// IncrementResponse Response from increment operation
type IncrementResponse struct {
	shared.CounterState
}


