// Package generated provides primitives for OpenAPI-based contract validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package generated

import (
	"context"
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeBankAccountActor is the Dapr actor type identifier for BankAccountActor
const ActorTypeBankAccountActor = "BankAccountActor"

// BankAccountActorAPIContract defines the interface that must be implemented to satisfy the OpenAPI contract for BankAccountActor.
// This interface enforces compile-time contract compliance.
type BankAccountActorAPIContract interface {
	// Get transaction history
	GetHistory(ctx context.Context) (*TransactionHistory, error)
	// Withdraw money from account
	Withdraw(ctx context.Context, request WithdrawRequest) (*BankAccountState, error)
	// Create new bank account
	CreateAccount(ctx context.Context, request CreateAccountRequest) (*BankAccountState, error)
	// Get current account balance
	GetBalance(ctx context.Context) (*BankAccountState, error)
	// Deposit money to account
	Deposit(ctx context.Context, request DepositRequest) (*BankAccountState, error)
}

// NewBankAccountActorFactoryContext creates a factory function for BankAccountActor with contract validation.
// The implementation parameter must implement BankAccountActorAPIContract interface.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// The generated factory ensures the actor Type() method returns the correct actor type.
func NewBankAccountActorFactoryContext(implementation func() BankAccountActorAPIContract) func() actor.ServerContext {
	return func() actor.ServerContext {
		// Compile-time check ensures the implementation satisfies the contract
		impl := implementation()
		
		// The implementation must also implement actor.ServerContext
		if serverCtx, ok := impl.(actor.ServerContext); ok {
			// Verify the actor type matches the contract
			if serverCtx.Type() != ActorTypeBankAccountActor {
				panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", serverCtx.Type(), ActorTypeBankAccountActor))
			}
			return serverCtx
		}
		
		// This should never happen if the actor is properly implemented
		panic("actor implementation must embed actor.ServerImplBaseCtx")
	}
}