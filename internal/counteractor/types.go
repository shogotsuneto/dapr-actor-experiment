// Package counteractor provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package counteractor


// CounterState Current state of the counter actor (state-based)
type CounterState struct {
	// The current counter value
	Value int32 `json:"value"`
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

// SetValueRequest Request to set the counter to a specific value
type SetValueRequest struct {
	// The value to set the counter to
	Value int32 `json:"value"`
}

// TransactionHistory Complete transaction history (event sourcing benefit)
type TransactionHistory struct {
	// List of all events in chronological order
	Events []interface{} `json:"events"`
	// Account identifier
	AccountId string `json:"accountId"`
}

// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Amount to withdraw
	Amount float64 `json:"amount"`
	// Description of the withdrawal
	Description string `json:"description"`
}

// AccountEvent A single account event
type AccountEvent struct {
	// Unique event identifier
	EventId string `json:"eventId"`
	// Type of event
	EventType string `json:"eventType"`
	// When the event occurred
	Timestamp string `json:"timestamp"`
	// Event-specific data
	Data map[string]interface{} `json:"data"`
}

// BankAccountState Current state of bank account (computed from events)
type BankAccountState struct {
	// Account owner name
	OwnerName string `json:"ownerName"`
	// Unique account identifier
	AccountId string `json:"accountId"`
	// Current account balance (computed from events)
	Balance float64 `json:"balance"`
	// Account creation timestamp
	CreatedAt string `json:"createdAt,omitempty"`
	// Whether account is active
	IsActive bool `json:"isActive"`
}

