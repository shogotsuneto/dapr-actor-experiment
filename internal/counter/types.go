// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter


// CounterState Current state of the counter actor (state-based)
type CounterState struct {
	// Error information returned within 200 responses
	Error Error `json:"error,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
	// The current counter value (only present if success=true)
	Value int32 `json:"value,omitempty"`
}

// Error Error information returned within 200 responses
type Error struct {
	// Error code identifying the type of error
	Code string `json:"code"`
	// Additional error-specific details
	Details map[string]interface{} `json:"details,omitempty"`
	// Human-readable error message
	Message string `json:"message"`
}

// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}


