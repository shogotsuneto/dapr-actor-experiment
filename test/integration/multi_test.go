package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

// Helper functions for multi-test
func assertCounterSuccessMulti(t *testing.T, state counter.CounterState, expectedValue int32, message string) {
	require.True(t, state.Success, "Counter operation should succeed: %s", message)
	require.NotNil(t, state.Data, "Counter data should not be nil when success=true")
	assert.Equal(t, expectedValue, state.Data.Value, message)
}

func assertBankAccountSuccessMulti(t *testing.T, state bankaccount.BankAccountState, expectedBalance float64, message string) {
	require.True(t, state.Success, "BankAccount operation should succeed: %s", message)
	require.NotNil(t, state.Data, "BankAccount data should not be nil when success=true")
	assert.Equal(t, expectedBalance, state.Data.Balance, message)
}



func TestMultiActorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient(GetDaprEndpoint())

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
	// This replicates the quick-demo.sh and individual test functionality

	// Counter operations
	counterActorID := fmt.Sprintf("multi-test-counter-%d", time.Now().UnixNano()%10000)
	var counterState counter.CounterState

	// Generate JWT token for Counter operations (use generic test user)
	counterToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for Counter operations")

	// Initialize counter
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   counterActorID,
		Method:    "Set",
		Data:      counter.SetValueRequest{Value: int32(5)},
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(5), "Counter should be set to 5")

	// BankAccount operations
	bankActorID := fmt.Sprintf("multi-test-account-%d", time.Now().UnixNano()%10000)
	
	// Generate JWT token for BankAccount operations (use actorID as userID for ownership)
	bankToken, err := generateTestToken(bankActorID, "Multi Test User", "multitest@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for BankAccount operations")
	
	// Create account
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   bankActorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			InitialDeposit: 2000.0,
		},
	}, bankToken, &createResult)
	require.NoError(t, err)

	// Perform operations on both actors interleaved
	// Increment counter
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   counterActorID,
		Method:    "Increment",
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(6), "Counter should be 6 after increment")

	// Deposit to bank account
	var depositResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   bankActorID,
		Method:    "Deposit",
		Data: bankaccount.DepositRequest{
			Amount:      500.0,
			Description: "Multi-actor test deposit",
		},
	}, bankToken, &depositResult)
	require.NoError(t, err)

	// Decrement counter
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   counterActorID,
		Method:    "Decrement",
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(5), "Counter should be set to 5")

	// Withdraw from bank account
	var withdrawResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   bankActorID,
		Method:    "Withdraw",
		Data: bankaccount.WithdrawRequest{
			Amount:      300.0,
			Description: "Multi-actor test withdrawal",
		},
	}, bankToken, &withdrawResult)
	require.NoError(t, err)

	// Verify final states
	// Counter should be 5
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   counterActorID,
		Method:    "Get",
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(5), "Counter should maintain its state")

	// Bank account should be 2200.0 (2000 + 500 - 300)
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   bankActorID,
		Method:    "GetBalance",
	}, bankToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccessMulti(t, balance, 2200.0, "Bank account should maintain its state")
}

func testActorTypesIsolation(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test that different actor types with same ID don't interfere
	actorID := fmt.Sprintf("isolation-test-%d", time.Now().UnixNano()%10000)

	// Generate JWT tokens
	counterToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for Counter operations")

	bankToken, err := generateTestToken(actorID, "Isolation Test User", "isolation@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for BankAccount operations")

	// Create Counter with ID "isolation-test"
	var counterState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Set",
		Data:      counter.SetValueRequest{Value: int32(100)},
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(100), "Counter should be 100")

	// Create BankAccount with same ID "isolation-test"
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			InitialDeposit: 1000.0,
		},
	}, bankToken, &createResult)
	require.NoError(t, err)

	// Verify both actors maintain separate state despite same ID
	// Check counter
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Get",
	}, counterToken, &counterState)
	require.NoError(t, err)
	assertCounterSuccessMulti(t, counterState, int32(100), "Counter should maintain its state")

	// Check bank account
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, bankToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccessMulti(t, balance, 1000.0, "BankAccount should maintain its state")
}

func testConcurrentActorOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test concurrent operations on multiple instances of both actor types
	// This simulates the comprehensive scenario from quick-demo.sh and individual test scripts

	// Setup multiple counter actors with unique IDs
	timestamp := time.Now().UnixNano() % 10000
	counterActors := []string{
		fmt.Sprintf("concurrent-counter-1-%d", timestamp),
		fmt.Sprintf("concurrent-counter-2-%d", timestamp),
		fmt.Sprintf("concurrent-counter-3-%d", timestamp),
	}
	counterValues := []int32{10, 20, 30}

	// Setup multiple bank account actors with unique IDs
	bankActors := []struct {
		id      string
		initial float64
	}{
		{fmt.Sprintf("concurrent-account-1-%d", timestamp), 1000.0},
		{fmt.Sprintf("concurrent-account-2-%d", timestamp), 2000.0},
		{fmt.Sprintf("concurrent-account-3-%d", timestamp), 3000.0},
	}

	// Generate JWT tokens
	counterToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for Counter operations")

	// Initialize all actors
	for i, actorID := range counterActors {
		var state counter.CounterState
		_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   actorID,
			Method:    "Set",
			Data:      counter.SetValueRequest{Value: counterValues[i]},
		}, counterToken, &state)
		require.NoError(t, err)
		assertCounterSuccessMulti(t, state, counterValues[i], fmt.Sprintf("Counter %s should have correct value", actorID))
	}

	for _, account := range bankActors {
		// Generate JWT token for each bank account (use actorID as userID for ownership)
		bankToken, err := generateTestToken(account.id, "user", fmt.Sprintf("%s@example.com", account.id), []string{"user"}, 1*time.Hour)
		require.NoError(t, err, "Failed to generate JWT token for BankAccount %s", account.id)

		var createResult interface{}
		_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   account.id,
			Method:    "CreateAccount",
			Data: bankaccount.CreateAccountRequest{
				InitialDeposit: account.initial,
			},
		}, bankToken, &createResult)
		require.NoError(t, err)
	}

	// Perform operations on all actors
	// Increment all counters
	for i, actorID := range counterActors {
		var state counter.CounterState
		_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   actorID,
			Method:    "Increment",
		}, counterToken, &state)
		require.NoError(t, err)
		assertCounterSuccessMulti(t, state, counterValues[i]+1, fmt.Sprintf("Counter %s should be incremented", actorID))
		counterValues[i]++ // Update expected value
	}

	// Deposit to all bank accounts
	depositAmount := 500.0
	for _, account := range bankActors {
		// Generate JWT token for this specific account
		bankToken, err := generateTestToken(account.id, "user", fmt.Sprintf("%s@example.com", account.id), []string{"user"}, 1*time.Hour)
		require.NoError(t, err, "Failed to generate JWT token for BankAccount %s", account.id)

		var depositResult interface{}
		_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   account.id,
			Method:    "Deposit",
			Data: bankaccount.DepositRequest{
				Amount:      depositAmount,
				Description: "Concurrent test deposit",
			},
		}, bankToken, &depositResult)
		require.NoError(t, err)
	}

	// Verify all states are maintained correctly
	for i, actorID := range counterActors {
		var state counter.CounterState
		_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   actorID,
			Method:    "Get",
		}, counterToken, &state)
		require.NoError(t, err)
		assertCounterSuccessMulti(t, state, counterValues[i], fmt.Sprintf("Counter %s should maintain correct state", actorID))
	}

	for _, account := range bankActors {
		// Generate JWT token for this specific account
		bankToken, err := generateTestToken(account.id, "user", fmt.Sprintf("%s@example.com", account.id), []string{"user"}, 1*time.Hour)
		require.NoError(t, err, "Failed to generate JWT token for BankAccount %s", account.id)

		var balance bankaccount.BankAccountState
		_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   account.id,
			Method:    "GetBalance",
		}, bankToken, &balance)
		require.NoError(t, err)
		expectedBalance := account.initial + depositAmount
		require.True(t, balance.Success, "Balance operation should succeed for account %s", account.id)
		require.NotNil(t, balance.Data, "Balance data should not be nil when success=true")
		assert.Equal(t, expectedBalance, balance.Data.Balance, "Account %s should have correct balance", account.id)
	}
}