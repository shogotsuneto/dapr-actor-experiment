// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated


// CounterState Current state of the counter actor
type CounterState struct {
	// The current counter value
	Value int32 `json:"value"`
}

// Error Error response format
type Error struct {
	// Additional error details
	Details map[string]interface{} `json:"details,omitempty"`
	// Human-readable error message
	Error string `json:"error"`
	// Machine-readable error code
	Code string `json:"code,omitempty"`
}

// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}


// BadRequest defines model for BadRequest.
type BadRequest = Error

// ServerError defines model for ServerError.
type ServerError = Error
