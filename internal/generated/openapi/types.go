// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated


// Error Error response format
type Error struct {
	// Machine-readable error code
	Code string `json:"code,omitempty"`
	// Additional error details
	Details map[string]interface{} `json:"details,omitempty"`
	// Human-readable error message
	Error string `json:"error"`
}

// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}

// CounterState Current state of the counter actor
type CounterState struct {
	// The current counter value
	Value int32 `json:"value"`
}


// BadRequest defines model for BadRequest.
type BadRequest = Error

// ServerError defines model for ServerError.
type ServerError = Error
