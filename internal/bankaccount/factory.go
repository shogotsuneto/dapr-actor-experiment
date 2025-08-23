// Package bankaccount provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package bankaccount

import (
	"fmt"
	"github.com/dapr/go-sdk/actor"
	"github.com/shogotsuneto/go-simple-eventstore"
)

// NewActorFactory creates a factory function for BankAccount with external dependencies using Closure pattern.
// Returns a factory function compatible with Dapr's RegisterActorImplFactoryContext.
// Usage: s.RegisterActorImplFactoryContext(bankaccount.NewActorFactory(eventStore))
func NewActorFactory(eventStore eventstore.EventStore) func() actor.ServerContext {
	return func() actor.ServerContext {
		// Create a new BankAccount instance with external dependencies
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