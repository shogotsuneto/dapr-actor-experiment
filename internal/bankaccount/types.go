// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount


// BankAccountState Current state of bank account (computed from events)
type BankAccountState struct {
	// Account data (only present if success=true)
	Data *BankAccountStateData `json:"data,omitempty"`
	// Error information returned within 200 responses
	Error *Error `json:"error,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
}

// BankAccountStateData Account data (only present if success=true)
type BankAccountStateData struct {
	// Unique account identifier
	AccountId string `json:"accountId"`
	// Current account balance (computed from events)
	Balance float64 `json:"balance"`
	// Account creation timestamp
	CreatedAt string `json:"createdAt"`
	// Whether account is active
	IsActive bool `json:"isActive"`
	// Account owner ID (for authorization)
	OwnerId string `json:"ownerId"`
	// Account owner name
	OwnerName string `json:"ownerName"`
}

// CreateAccountRequest Request to create a new bank account
type CreateAccountRequest struct {
	// Initial deposit amount
	InitialDeposit float64 `json:"initialDeposit"`
	// Name of the account owner
	OwnerName string `json:"ownerName"`
}

// DepositRequest Request to deposit money
type DepositRequest struct {
	// Amount to deposit
	Amount float64 `json:"amount"`
	// Description of the deposit
	Description string `json:"description"`
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

// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Amount to withdraw
	Amount float64 `json:"amount"`
	// Description of the withdrawal
	Description string `json:"description"`
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
