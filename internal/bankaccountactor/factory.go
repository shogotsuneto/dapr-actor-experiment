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
		// Get the implementation (which already implements both BankAccountActorAPI and actor.ServerContext)
		impl := implementation()
		
		// Verify the actor type matches the schema
		if impl.Type() != ActorTypeBankAccountActor {
			panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", impl.Type(), ActorTypeBankAccountActor))
		}
		
		return impl
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
		
		// Verify the actor type matches the schema
		if impl.Type() != ActorTypeBankAccountActor {
			panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", impl.Type(), ActorTypeBankAccountActor))
		}
		
		return impl
	}
}