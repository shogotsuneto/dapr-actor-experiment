// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount


// BankAccountState Current state of bank account (computed from events)
type BankAccountState struct {
	// Unique account identifier
	AccountId string `json:"accountId"`
	// Current account balance (computed from events)
	Balance float64 `json:"balance"`
	// Account creation timestamp
	CreatedAt string `json:"createdAt,omitempty"`
	// Whether account is active
	IsActive bool `json:"isActive"`
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

// TransactionHistory Complete transaction history (event sourcing benefit)
type TransactionHistory struct {
	// Account identifier
	AccountId string `json:"accountId"`
	// List of all events in chronological order
	Events []interface{} `json:"events"`
}

// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Amount to withdraw
	Amount float64 `json:"amount"`
	// Description of the withdrawal
	Description string `json:"description"`
}


