// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount

import (
	"context"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeBankAccount is the Dapr actor type identifier for BankAccount
const ActorTypeBankAccount = "BankAccount"

// BankAccountAPI defines the interface that must be implemented to satisfy the OpenAPI schema for BankAccount.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type BankAccountAPI interface {
	actor.ServerContext
	// Create new bank account
	CreateAccount(ctx context.Context, request CreateAccountRequest) (*BankAccountState, error)
	// Deposit money to account
	Deposit(ctx context.Context, request DepositRequest) (*BankAccountState, error)
	// Get current account balance
	GetBalance(ctx context.Context) (*BankAccountState, error)
	// Get transaction history
	GetHistory(ctx context.Context) (*TransactionHistory, error)
	// Withdraw money from account
	Withdraw(ctx context.Context, request WithdrawRequest) (*BankAccountState, error)
}