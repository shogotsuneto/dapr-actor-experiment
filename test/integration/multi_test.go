package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiActorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient("http://localhost:3500")

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running. Start with: docker compose -f test/integration/docker-compose.test.yml up -d")

	t.Run("TestMultipleActorTypes", func(t *testing.T) {
		testMultipleActorTypes(t, daprClient)
	})

	t.Run("TestActorTypesIsolation", func(t *testing.T) {
		testActorTypesIsolation(t, daprClient)
	})

	t.Run("TestConcurrentActorOperations", func(t *testing.T) {
		testConcurrentActorOperations(t, daprClient)
	})
}

func testMultipleActorTypes(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test that both actor types can operate simultaneously
	// This replicates the test-multi-actors.sh functionality

	// CounterActor operations
	counterActorID := "multi-test-counter"
	var counterState CounterState

	// Initialize counter
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   counterActorID,
		Method:    "Set",
		Data:      SetValueRequest{Value: 5},
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 5, counterState.Value)

	// BankAccountActor operations
	bankActorID := "multi-test-account"
	
	// Create account
	var createResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   bankActorID,
		Method:    "CreateAccount",
		Data: CreateAccountRequest{
			OwnerName:      "Multi Test User",
			InitialDeposit: 2000.0,
		},
	}, &createResult)
	require.NoError(t, err)

	// Perform operations on both actors interleaved
	// Increment counter
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   counterActorID,
		Method:    "Increment",
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 6, counterState.Value)

	// Deposit to bank account
	var depositResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   bankActorID,
		Method:    "Deposit",
		Data: DepositRequest{
			Amount:      500.0,
			Description: "Multi-actor test deposit",
		},
	}, &depositResult)
	require.NoError(t, err)

	// Decrement counter
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   counterActorID,
		Method:    "Decrement",
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 5, counterState.Value)

	// Withdraw from bank account
	var withdrawResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   bankActorID,
		Method:    "Withdraw",
		Data: WithdrawRequest{
			Amount:      300.0,
			Description: "Multi-actor test withdrawal",
		},
	}, &withdrawResult)
	require.NoError(t, err)

	// Verify final states
	// Counter should be 5
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   counterActorID,
		Method:    "Get",
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 5, counterState.Value, "Counter should maintain its state")

	// Bank account should be 2200.0 (2000 + 500 - 300)
	var balance BankAccountBalance
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   bankActorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	assert.Equal(t, 2200.0, balance.Balance, "Bank account should maintain its state")
}

func testActorTypesIsolation(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test that different actor types with same ID don't interfere
	actorID := "isolation-test"

	// Create CounterActor with ID "isolation-test"
	var counterState CounterState
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Set",
		Data:      SetValueRequest{Value: 100},
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 100, counterState.Value)

	// Create BankAccountActor with same ID "isolation-test"
	var createResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: CreateAccountRequest{
			OwnerName:      "Isolation Test",
			InitialDeposit: 1000.0,
		},
	}, &createResult)
	require.NoError(t, err)

	// Verify both actors maintain separate state despite same ID
	// Check counter
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Get",
	}, &counterState)
	require.NoError(t, err)
	assert.Equal(t, 100, counterState.Value, "CounterActor should maintain its state")

	// Check bank account
	var balance BankAccountBalance
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, balance.Balance, "BankAccountActor should maintain its state")
}

func testConcurrentActorOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test concurrent operations on multiple instances of both actor types
	// This simulates the comprehensive scenario from test-multi-actors.sh

	// Setup multiple counter actors
	counterActors := []string{"concurrent-counter-1", "concurrent-counter-2", "concurrent-counter-3"}
	counterValues := []int{10, 20, 30}

	// Setup multiple bank account actors
	bankActors := []struct {
		id      string
		owner   string
		initial float64
	}{
		{"concurrent-account-1", "User One", 1000.0},
		{"concurrent-account-2", "User Two", 2000.0},
		{"concurrent-account-3", "User Three", 3000.0},
	}

	// Initialize all actors
	for i, actorID := range counterActors {
		var state CounterState
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Set",
			Data:      SetValueRequest{Value: counterValues[i]},
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, counterValues[i], state.Value)
	}

	for _, account := range bankActors {
		var createResult interface{}
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   account.id,
			Method:    "CreateAccount",
			Data: CreateAccountRequest{
				OwnerName:      account.owner,
				InitialDeposit: account.initial,
			},
		}, &createResult)
		require.NoError(t, err)
	}

	// Perform operations on all actors
	// Increment all counters
	for i, actorID := range counterActors {
		var state CounterState
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Increment",
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, counterValues[i]+1, state.Value)
		counterValues[i]++ // Update expected value
	}

	// Deposit to all bank accounts
	depositAmount := 500.0
	for _, account := range bankActors {
		var depositResult interface{}
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   account.id,
			Method:    "Deposit",
			Data: DepositRequest{
				Amount:      depositAmount,
				Description: "Concurrent test deposit",
			},
		}, &depositResult)
		require.NoError(t, err)
	}

	// Verify all states are maintained correctly
	for i, actorID := range counterActors {
		var state CounterState
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Get",
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, counterValues[i], state.Value, "Counter %s should maintain correct state", actorID)
	}

	for _, account := range bankActors {
		var balance BankAccountBalance
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   account.id,
			Method:    "GetBalance",
		}, &balance)
		require.NoError(t, err)
		expectedBalance := account.initial + depositAmount
		assert.Equal(t, expectedBalance, balance.Balance, "Account %s should have correct balance", account.id)
		assert.Equal(t, account.owner, balance.OwnerName, "Account %s should have correct owner", account.id)
	}

	// Test transaction history for one of the bank accounts
	var history BankAccountHistory
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   bankActors[0].id,
		Method:    "GetHistory",
	}, &history)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(history.Transactions), 2, "Should have at least account creation and deposit transactions")
}