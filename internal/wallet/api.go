// Package wallet provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package wallet

import (
	"context"
	"github.com/dapr/go-sdk/actor"
)

// ActorTypeWallet is the Dapr actor type identifier for Wallet
const ActorTypeWallet = "Wallet"

// WalletAPI defines the interface that must be implemented to satisfy the OpenAPI schema for Wallet.
// This interface enforces compile-time schema compliance and includes actor.ServerContext for proper Dapr actor implementation.
type WalletAPI interface {
	actor.ServerContext
	// Add funds to wallet
	AddFunds(ctx context.Context, request AddFundsRequest) (*WalletState, error)
	// Create a new wallet
	CreateWallet(ctx context.Context, request CreateWalletRequest) (*WalletState, error)
	// Get current wallet balance
	GetBalance(ctx context.Context) (*WalletState, error)
	// Get wallet transaction history
	GetTransactions(ctx context.Context) (*WalletTransactionHistory, error)
	// Spend funds from wallet
	SpendFunds(ctx context.Context, request SpendFundsRequest) (*WalletState, error)
}