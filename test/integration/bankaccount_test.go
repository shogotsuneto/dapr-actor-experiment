package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccountactor"
)

func TestBankAccountActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient(GetDaprEndpoint())

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running. Start with: docker compose -f test/integration/docker-compose.test.yml up -d")

	t.Run("TestBankAccountActorBasicOperations", func(t *testing.T) {
		testBankAccountActorBasicOperations(t, daprClient)
	})

	t.Run("TestBankAccountActorStateIsolation", func(t *testing.T) {
		testBankAccountActorStateIsolation(t, daprClient)
	})

	t.Run("TestBankAccountActorEventSourcing", func(t *testing.T) {
		testBankAccountActorEventSourcing(t, daprClient)
	})
}

func testBankAccountActorBasicOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := "account-test-basic"

	// Test 1: Create account
	var createResult interface{}
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccountactor.CreateAccountRequest{
			OwnerName:      "Test User",
			InitialDeposit: 1000.0,
		},
	}, &createResult)
	require.NoError(t, err)

	// Test 2: Get initial balance
	var balance bankaccountactor.BankAccountState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	assert.Equal(t, 1000.0, balance.Balance, "Initial balance should be 1000.0")
	assert.Equal(t, "Test User", balance.OwnerName, "Owner name should match")

	// Test 3: Deposit money
	var depositResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "Deposit",
		Data: bankaccountactor.DepositRequest{
			Amount:      500.0,
			Description: "Test deposit",
		},
	}, &depositResult)
	require.NoError(t, err)

	// Test 4: Check balance after deposit
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	assert.Equal(t, 1500.0, balance.Balance, "Balance should be 1500.0 after deposit")

	// Test 5: Withdraw money
	var withdrawResult interface{}
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "Withdraw",
		Data: bankaccountactor.WithdrawRequest{
			Amount:      200.0,
			Description: "Test withdrawal",
		},
	}, &withdrawResult)
	require.NoError(t, err)

	// Test 6: Check final balance
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	assert.Equal(t, 1300.0, balance.Balance, "Final balance should be 1300.0")
}

func testBankAccountActorStateIsolation(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test scenario similar to the shell script test-bank-account-actor.sh
	testAccounts := []struct {
		actorID        string
		ownerName      string
		initialDeposit float64
		operations     []Operation
		expectedBalance float64
	}{
		{
			actorID:        "account-alice",
			ownerName:      "Alice Johnson",
			initialDeposit: 1500.0,
			operations: []Operation{
				{Type: "deposit", Amount: 3000.0, Description: "Monthly salary"},
				{Type: "withdraw", Amount: 1200.0, Description: "Rent payment"},
				{Type: "withdraw", Amount: 150.0, Description: "Grocery shopping"},
			},
			expectedBalance: 3150.0, // 1500 + 3000 - 1200 - 150
		},
		{
			actorID:        "account-bob",
			ownerName:      "Bob Smith",
			initialDeposit: 500.0,
			operations: []Operation{
				{Type: "deposit", Amount: 800.0, Description: "Freelance project payment"},
				{Type: "deposit", Amount: 200.0, Description: "Performance bonus"},
				{Type: "withdraw", Amount: 350.0, Description: "Car loan payment"},
			},
			expectedBalance: 1150.0, // 500 + 800 + 200 - 350
		},
		{
			actorID:        "account-charlie",
			ownerName:      "Charlie Brown",
			initialDeposit: 2000.0,
			operations: []Operation{
				{Type: "withdraw", Amount: 50.0, Description: "Coffee shop"},
				{Type: "withdraw", Amount: 25.0, Description: "Parking fee"},
				{Type: "withdraw", Amount: 100.0, Description: "Gas station"},
				{Type: "deposit", Amount: 5000.0, Description: "Investment return"},
			},
			expectedBalance: 6825.0, // 2000 - 50 - 25 - 100 + 5000
		},
	}

	for _, account := range testAccounts {
		t.Run("Account_"+account.actorID, func(t *testing.T) {
			// Create account
			var createResult interface{}
			err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
				ActorType: "BankAccountActor",
				ActorID:   account.actorID,
				Method:    "CreateAccount",
				Data: bankaccountactor.CreateAccountRequest{
					OwnerName:      account.ownerName,
					InitialDeposit: account.initialDeposit,
				},
			}, &createResult)
			require.NoError(t, err)

			// Execute operations
			for _, op := range account.operations {
				var result interface{}
				if op.Type == "deposit" {
					err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "BankAccountActor",
						ActorID:   account.actorID,
						Method:    "Deposit",
						Data: bankaccountactor.DepositRequest{
							Amount:      op.Amount,
							Description: op.Description,
						},
					}, &result)
				} else if op.Type == "withdraw" {
					err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "BankAccountActor",
						ActorID:   account.actorID,
						Method:    "Withdraw",
						Data: bankaccountactor.WithdrawRequest{
							Amount:      op.Amount,
							Description: op.Description,
						},
					}, &result)
				}
				require.NoError(t, err)
			}

			// Verify final balance
			var balance bankaccountactor.BankAccountState
			err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
				ActorType: "BankAccountActor",
				ActorID:   account.actorID,
				Method:    "GetBalance",
			}, &balance)
			require.NoError(t, err)
			assert.Equal(t, account.expectedBalance, balance.Balance, "Final balance for %s should be %.2f", account.actorID, account.expectedBalance)
			assert.Equal(t, account.ownerName, balance.OwnerName, "Owner name should match for %s", account.actorID)
		})
	}
}

func testBankAccountActorEventSourcing(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := "account-event-sourcing-test"

	// Create account
	var createResult interface{}
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccountactor.CreateAccountRequest{
			OwnerName:      "Event Sourcing Test",
			InitialDeposit: 1000.0,
		},
	}, &createResult)
	require.NoError(t, err)

	// Perform multiple operations
	operations := []Operation{
		{Type: "deposit", Amount: 500.0, Description: "First deposit"},
		{Type: "withdraw", Amount: 200.0, Description: "First withdrawal"},
		{Type: "deposit", Amount: 300.0, Description: "Second deposit"},
		{Type: "withdraw", Amount: 100.0, Description: "Second withdrawal"},
	}

	for _, op := range operations {
		var result interface{}
		if op.Type == "deposit" {
			err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
				ActorType: "BankAccountActor",
				ActorID:   actorID,
				Method:    "Deposit",
				Data: bankaccountactor.DepositRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, &result)
		} else if op.Type == "withdraw" {
			err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
				ActorType: "BankAccountActor",
				ActorID:   actorID,
				Method:    "Withdraw",
				Data: bankaccountactor.WithdrawRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, &result)
		}
		require.NoError(t, err)
	}

	// Get transaction history to verify event sourcing
	var history bankaccountactor.TransactionHistory
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetHistory",
	}, &history)
	require.NoError(t, err)

	// Verify transaction history contains all operations (including account creation)
	// Should have: 1 account creation + 4 operations = 5 events
	assert.GreaterOrEqual(t, len(history.Events), 5, "Should have at least 5 events including account creation")

	// Verify event types
	foundDeposits := 0
	foundWithdrawals := 0
	foundAccountCreated := 0
	for _, eventInterface := range history.Events {
		// Convert interface{} to map[string]interface{} (JSON unmarshaling result)
		eventMap, ok := eventInterface.(map[string]interface{})
		if !ok {
			continue
		}
		eventType, ok := eventMap["eventType"].(string)
		if !ok {
			continue
		}
		switch eventType {
		case "AccountCreated":
			foundAccountCreated++
		case "MoneyDeposited":
			foundDeposits++
		case "MoneyWithdrawn":
			foundWithdrawals++
		}
	}
	assert.GreaterOrEqual(t, foundAccountCreated, 1, "Should have account creation event")
	assert.GreaterOrEqual(t, foundDeposits, 2, "Should have at least 2 deposit events")
	assert.GreaterOrEqual(t, foundWithdrawals, 2, "Should have at least 2 withdrawal events")

	// Verify final balance matches expected calculation
	var balance bankaccountactor.BankAccountState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, &balance)
	require.NoError(t, err)
	expectedBalance := 1000.0 + 500.0 - 200.0 + 300.0 - 100.0 // 1500.0
	assert.Equal(t, expectedBalance, balance.Balance, "Final balance should match event sourcing calculation")
}

// Operation represents a bank account operation
type Operation struct {
	Type        string
	Amount      float64
	Description string
}