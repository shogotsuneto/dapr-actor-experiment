// Package bankaccountactor provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccountactor

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

// NewBankAccountActorFactoryContext creates a factory function for BankAccountActor with schema validation.
// The implementation parameter must implement BankAccountActorAPI interface.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// The generated factory ensures the actor Type() method returns the correct actor type.
func NewBankAccountActorFactoryContext(implementation func() BankAccountActorAPI) func() actor.ServerContext {
	return func() actor.ServerContext {
		// Compile-time check ensures the implementation satisfies the schema
		impl := implementation()
		
		// The implementation must also implement actor.ServerContext
		if serverCtx, ok := impl.(actor.ServerContext); ok {
			// Verify the actor type matches the schema
			if serverCtx.Type() != ActorTypeBankAccountActor {
				panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", serverCtx.Type(), ActorTypeBankAccountActor))
			}
			return serverCtx
		}
		
		// This should never happen if the actor is properly implemented
		panic("actor implementation must embed actor.ServerImplBaseCtx")
	}
}

// NewActorFactory creates a factory function for BankAccountActor with a cleaner API.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// Usage: s.RegisterActorImplFactoryContext(bankaccountactor.NewActorFactory())
func NewActorFactory() func() actor.ServerContext {
	return func() actor.ServerContext {
		// Create a new BankAccountActor instance
		impl := &BankAccountActor{}
		
		// Compile-time check ensures the implementation satisfies the schema
		var _ BankAccountActorAPI = impl
		
		// The implementation must also implement actor.ServerContext
		if serverCtx, ok := interface{}(impl).(actor.ServerContext); ok {
			// Verify the actor type matches the schema
			if serverCtx.Type() != ActorTypeBankAccountActor {
				panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", serverCtx.Type(), ActorTypeBankAccountActor))
			}
			return serverCtx
		}
		
		panic("actor implementation must embed actor.ServerImplBaseCtx and implement actor.ServerContext")
	}
}