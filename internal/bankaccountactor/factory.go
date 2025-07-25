// Package bankaccountactor provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccountactor

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
)

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