// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter


// CounterState Current state of the counter actor (state-based)
type CounterState struct {
	// The current counter value
	Value int32 `json:"value"`
}

// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}


