// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount




// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Description of the withdrawal
	Description string `json:"description"`
	// Amount to withdraw
	Amount float64 `json:"amount"`
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
	// Description of the deposit
	Description string `json:"description"`
	// Amount to deposit
	Amount float64 `json:"amount"`
}


