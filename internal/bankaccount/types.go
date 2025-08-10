// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount


// BankAccountState Current state of bank account (computed from events)
type BankAccountState struct {
	// Unique account identifier (only present if success=true)
	AccountId string `json:"accountId,omitempty"`
	// Current account balance (computed from events, only present if success=true)
	Balance float64 `json:"balance,omitempty"`
	// Account creation timestamp (only present if success=true)
	CreatedAt string `json:"createdAt,omitempty"`
	// Error information returned within 200 responses
	Error Error `json:"error,omitempty"`
	// Whether account is active (only present if success=true)
	IsActive bool `json:"isActive,omitempty"`
	// Account owner ID (for authorization, only present if success=true)
	OwnerId string `json:"ownerId,omitempty"`
	// Account owner name (only present if success=true)
	OwnerName string `json:"ownerName,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
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
	Code string `json:"code"`
	// Additional error-specific details
	Details map[string]interface{} `json:"details,omitempty"`
	// Human-readable error message
	Message string `json:"message"`
}

// TransactionHistory Complete transaction history (event sourcing benefit)
type TransactionHistory struct {
	// Account identifier (only present if success=true)
	AccountId string `json:"accountId,omitempty"`
	// Error information returned within 200 responses
	Error Error `json:"error,omitempty"`
	// List of all events in chronological order (only present if success=true)
	Events []interface{} `json:"events,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
}

// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Amount to withdraw
	Amount float64 `json:"amount"`
	// Description of the withdrawal
	Description string `json:"description"`
}


