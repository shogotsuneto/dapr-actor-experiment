// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount


import (
	"github.com/shogotsuneto/dapr-actor-experiment/internal/shared"
)



// WithdrawRequest Request to withdraw money
type WithdrawRequest struct {
	// Amount to withdraw
	Amount float64 `json:"amount"`
	// Description of the withdrawal
	Description string `json:"description"`
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

// CreateAccountResponse Response from createaccount operation
type CreateAccountResponse struct {
	shared.BankAccountState
}

// GetBalanceResponse Response from getbalance operation
type GetBalanceResponse struct {
	shared.BankAccountState
}

// DepositResponse Response from deposit operation
type DepositResponse struct {
	shared.BankAccountState
}

// GetHistoryResponse Response from gethistory operation
type GetHistoryResponse struct {
	shared.TransactionHistory
}

// WithdrawResponse Response from withdraw operation
type WithdrawResponse struct {
	shared.BankAccountState
}


