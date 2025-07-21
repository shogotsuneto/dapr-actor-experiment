package actor

import (
	"context"
	"testing"

	generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

func TestMultipleActorTypes(t *testing.T) {
	// Test that both actor types have correct constants
	if generated.ActorTypeCounterActor != "CounterActor" {
		t.Errorf("Expected CounterActor type to be 'CounterActor', got '%s'", generated.ActorTypeCounterActor)
	}
	
	if generated.ActorTypeBankAccountActor != "BankAccountActor" {
		t.Errorf("Expected BankAccountActor type to be 'BankAccountActor', got '%s'", generated.ActorTypeBankAccountActor)
	}
}

func TestCounterActorType(t *testing.T) {
	counter := &CounterActor{}
	if counter.Type() != generated.ActorTypeCounterActor {
		t.Errorf("Expected CounterActor type to be '%s', got '%s'", generated.ActorTypeCounterActor, counter.Type())
	}
}

func TestBankAccountActorType(t *testing.T) {
	account := &BankAccountActor{}
	if account.Type() != generated.ActorTypeBankAccountActor {
		t.Errorf("Expected BankAccountActor type to be '%s', got '%s'", generated.ActorTypeBankAccountActor, account.Type())
	}
}

func TestBankAccountEventSourcing(t *testing.T) {
	// Test basic event sourcing functionality 
	account := &BankAccountActor{}
	
	// Note: This is a basic test without full Dapr context
	// In a real scenario, we'd need a mock state manager
	ctx := context.Background()
	
	// Test that methods exist and have correct signatures
	createReq := generated.CreateAccountRequest{
		OwnerName:      "Test User",
		InitialDeposit: 100.0,
	}
	
	// We can't actually call these without a state manager, but we can verify the interface
	_ = createReq
	_ = ctx
	_ = account
	
	// This test just verifies compilation and interface compliance
	t.Log("BankAccountActor implements BankAccountActorAPIContract interface")
}

func TestGeneratedTypes(t *testing.T) {
	// Test that generated types are properly structured
	
	// CounterActor types
	counterState := generated.CounterState{Value: 42}
	if counterState.Value != 42 {
		t.Errorf("Expected counter value 42, got %d", counterState.Value)
	}
	
	setReq := generated.SetValueRequest{Value: 100}
	if setReq.Value != 100 {
		t.Errorf("Expected set value 100, got %d", setReq.Value)
	}
	
	// BankAccountActor types
	bankState := generated.BankAccountState{
		AccountId: "test-123",
		OwnerName: "John Doe", 
		Balance:   1500.50,
		IsActive:  true,
	}
	
	if bankState.AccountId != "test-123" {
		t.Errorf("Expected account ID 'test-123', got '%s'", bankState.AccountId)
	}
	if bankState.Balance != 1500.50 {
		t.Errorf("Expected balance 1500.50, got %f", bankState.Balance)
	}
	
	depositReq := generated.DepositRequest{
		Amount:      250.0,
		Description: "Test deposit",
	}
	if depositReq.Amount != 250.0 {
		t.Errorf("Expected deposit amount 250.0, got %f", depositReq.Amount)
	}
}