// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount

import "github.com/shogotsuneto/dapr-actor-experiment/internal/shared"

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

// GetBalanceResponse Response from get balance operation
type GetBalanceResponse struct {
	shared.BankAccountState
}

// WithdrawResponse Response from withdraw operation
type WithdrawResponse struct {
	shared.BankAccountState
}

// GetHistoryResponse Response from get history operation
type GetHistoryResponse struct {
	shared.TransactionHistory
}

// CreateAccountResponse Response from create account operation
type CreateAccountResponse struct {
	shared.BankAccountState
}

// DepositResponse Response from deposit operation
type DepositResponse struct {
	shared.BankAccountState
}


