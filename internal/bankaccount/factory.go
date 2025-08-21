// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// WARNING: This file will be overwritten by code generation. 
// Developers should restore the implementation after regeneration.
package bankaccount

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
	"github.com/shogotsuneto/go-simple-eventstore"
)

// NewActorFactory creates a factory function for BankAccount with a cleaner API.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// Usage: s.RegisterActorImplFactoryContext(bankaccount.NewActorFactory(eventStore))
func NewActorFactory(eventStore eventstore.EventStore) func() actor.ServerContext {
	return func() actor.ServerContext {
		// Create a new BankAccount instance using the constructor with closure-captured eventStore
		impl := NewBankAccount(eventStore)
		
		// Compile-time check ensures the implementation satisfies the schema
		var _ BankAccountAPI = impl
		
		// Verify the actor type matches the schema
		if impl.Type() != ActorTypeBankAccount {
			panic(fmt.Sprintf("actor implementation Type() returns '%s', expected '%s'", impl.Type(), ActorTypeBankAccount))
		}
		
		return impl
	}
}