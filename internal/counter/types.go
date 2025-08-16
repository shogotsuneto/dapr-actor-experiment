// Package counter provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counter


// CounterState Current state of the counter actor (state-based)
type CounterState struct {
	// Counter data (only present if success=true)
	Data *CounterStateData `json:"data,omitempty"`
	// Error information returned within 200 responses
	Error Error `json:"error,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
}

// CounterStateData contains the counter data when success=true
type CounterStateData struct {
	// The current counter value
	Value int32 `json:"value"`
}

// Error Error information returned within 200 responses
type Error struct {
	// Error code identifying the type of error
	Code ErrorCode `json:"code"`
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





// ErrorCode defines valid values for Error.code
type ErrorCode string

// ErrorCode constants
const (
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"
	ErrorCodeAuthenticationError ErrorCode = "AUTHENTICATION_ERROR"
	ErrorCodeAuthorizationError ErrorCode = "AUTHORIZATION_ERROR"
	ErrorCodeInsufficientFunds ErrorCode = "INSUFFICIENT_FUNDS"
	ErrorCodeAccountNotFound ErrorCode = "ACCOUNT_NOT_FOUND"
	ErrorCodeAccountAlreadyExists ErrorCode = "ACCOUNT_ALREADY_EXISTS"
	ErrorCodeValueOutOfRange ErrorCode = "VALUE_OUT_OF_RANGE"
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)
