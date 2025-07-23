package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestActorServicesAssumeRunning tests the integration without managing Docker services
// This test assumes services are already running (via ./scripts/run-docker.sh)
func TestActorServicesAssumeRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Simple test that assumes services are already running
	daprClient := NewDaprClient("http://localhost:3500")
	ctx := context.Background()

	t.Run("TestBasicCounterOperations", func(t *testing.T) {
		actorID := "quick-test-counter"

		// Test get initial value
		var state CounterState
		err := daprClient.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Get",
		}, &state)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, state.Value, 0, "Counter value should be non-negative")

		// Test increment
		err = daprClient.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Increment",
		}, &state)
		require.NoError(t, err)
		initialValue := state.Value

		// Test increment again and verify it increased
		err = daprClient.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Increment",
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, initialValue+1, state.Value, "Counter should increment by 1")
	})

	t.Run("TestBasicBankAccountOperations", func(t *testing.T) {
		actorID := "quick-test-account-" + fmt.Sprintf("%d", time.Now().UnixNano())

		// Test create account
		var createResult interface{}
		err := daprClient.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   actorID,
			Method:    "CreateAccount",
			Data: CreateAccountRequest{
				OwnerName:      "Quick Test User",
				InitialDeposit: 500.0,
			},
		}, &createResult)
		require.NoError(t, err)

		// Test get balance
		var balance BankAccountBalance
		err = daprClient.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   actorID,
			Method:    "GetBalance",
		}, &balance)
		require.NoError(t, err)
		assert.Equal(t, 500.0, balance.Balance, "Initial balance should be 500.0")
		assert.Equal(t, "Quick Test User", balance.OwnerName, "Owner name should match")
	})
}