// Package bankaccountactor provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccountactor

import (
	"context"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeBankAccountActor is the Dapr actor type identifier for BankAccountActor
const ActorTypeBankAccountActor = "BankAccountActor"

// BankAccountActorAPI defines the interface that must be implemented to satisfy the OpenAPI schema for BankAccountActor.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type BankAccountActorAPI interface {
	actor.ServerContext
	// Get transaction history
	GetHistory(ctx context.Context) (*TransactionHistory, error)
	// Withdraw money from account
	Withdraw(ctx context.Context, request WithdrawRequest) (*BankAccountState, error)
	// Deposit money to account
	Deposit(ctx context.Context, request DepositRequest) (*BankAccountState, error)
	// Get current account balance
	GetBalance(ctx context.Context) (*BankAccountState, error)
	// Create new bank account
	CreateAccount(ctx context.Context, request CreateAccountRequest) (*BankAccountState, error)
}