package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
)

// Helper functions to assert bankaccount response success
func assertBankAccountSuccess(t *testing.T, state bankaccount.BankAccountState, expectedBalance float64, message string) {
	require.True(t, state.Success, "BankAccount operation should succeed: %s", message)
	require.NotNil(t, state.Data, "BankAccount data should not be nil when success=true")
	assert.Equal(t, expectedBalance, state.Data.Balance, message)
}

func assertBankAccountSuccessWithOwner(t *testing.T, state bankaccount.BankAccountState, expectedBalance float64, expectedOwner string, message string) {
	require.True(t, state.Success, "BankAccount operation should succeed: %s", message)
	require.NotNil(t, state.Data, "BankAccount data should not be nil when success=true")
	assert.Equal(t, expectedBalance, state.Data.Balance, message)
	assert.Equal(t, expectedOwner, state.Data.OwnerName, "Owner name should match")
}



func TestBankAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient(GetDaprEndpoint())

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running. Start with: docker compose -f test/integration/docker-compose.test.yml up -d")

	t.Run("TestBankAccountBasicOperations", func(t *testing.T) {
		testBankAccountBasicOperations(t, daprClient)
	})

	t.Run("TestBankAccountStateIsolation", func(t *testing.T) {
		testBankAccountStateIsolation(t, daprClient)
	})

	t.Run("TestBankAccountEventSourcing", func(t *testing.T) {
		testBankAccountEventSourcing(t, daprClient)
	})

	t.Run("TestBankAccountAutomaticSnapshotCreation", func(t *testing.T) {
		testBankAccountAutomaticSnapshotCreation(t, daprClient)
	})

	t.Run("TestBankAccountSnapshotPerformance", func(t *testing.T) {
		testBankAccountSnapshotPerformance(t, daprClient)
	})
}

func testBankAccountBasicOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := "account-test-basic-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000) // Unique ID

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken(actorID, "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test 1: Create account
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			OwnerName:      "Test User",
			InitialDeposit: 1000.0,
		},
	}, userToken, &createResult)
	require.NoError(t, err)

	// Test 2: Get initial balance
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccessWithOwner(t, balance, 1000.0, "Test User", "Initial balance should be 1000.0")

	// Test 3: Deposit money
	var depositResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "Deposit",
		Data: bankaccount.DepositRequest{
			Amount:      500.0,
			Description: "Test deposit",
		},
	}, userToken, &depositResult)
	require.NoError(t, err)

	// Test 4: Check balance after deposit
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccess(t, balance, 1500.0, "Balance should be 1500.0 after deposit")

	// Test 5: Withdraw money
	var withdrawResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "Withdraw",
		Data: bankaccount.WithdrawRequest{
			Amount:      200.0,
			Description: "Test withdrawal",
		},
	}, userToken, &withdrawResult)
	require.NoError(t, err)

	// Test 6: Check final balance
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccess(t, balance, 1300.0, "Final balance should be 1300.0")
}

func testBankAccountStateIsolation(t *testing.T, client *DaprClient) {
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
			actorID:        fmt.Sprintf("account-alice-%d", time.Now().UnixNano()%10000),
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
			actorID:        fmt.Sprintf("account-bob-%d", time.Now().UnixNano()%10000),
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
			actorID:        fmt.Sprintf("account-charlie-%d", time.Now().UnixNano()%10000),
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
			// Generate JWT token for this account
			userToken, err := generateTestToken(account.actorID, account.ownerName, account.ownerName+"@example.com", []string{"user"}, 1*time.Hour)
			require.NoError(t, err, "Failed to generate JWT token for testing")

			// Create account
			var createResult interface{}
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   account.actorID,
				Method:    "CreateAccount",
				Data: bankaccount.CreateAccountRequest{
					OwnerName:      account.ownerName,
					InitialDeposit: account.initialDeposit,
				},
			}, userToken, &createResult)
			require.NoError(t, err)

			// Execute operations
			for _, op := range account.operations {
				var result interface{}
				if op.Type == "deposit" {
					_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "BankAccount",
						ActorID:   account.actorID,
						Method:    "Deposit",
						Data: bankaccount.DepositRequest{
							Amount:      op.Amount,
							Description: op.Description,
						},
					}, userToken, &result)
				} else if op.Type == "withdraw" {
					_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "BankAccount",
						ActorID:   account.actorID,
						Method:    "Withdraw",
						Data: bankaccount.WithdrawRequest{
							Amount:      op.Amount,
							Description: op.Description,
						},
					}, userToken, &result)
				}
				require.NoError(t, err)
			}

			// Verify final balance
			var balance bankaccount.BankAccountState
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   account.actorID,
				Method:    "GetBalance",
			}, userToken, &balance)
			require.NoError(t, err)
			assertBankAccountSuccessWithOwner(t, balance, account.expectedBalance, account.ownerName, fmt.Sprintf("Final balance for %s should be %.2f", account.actorID, account.expectedBalance))
		})
	}
}

func testBankAccountEventSourcing(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := fmt.Sprintf("account-event-sourcing-test-%d", time.Now().UnixNano()%10000)

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken(actorID, "event-sourcing-user", "events@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Create account
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			OwnerName:      "Event Sourcing Test",
			InitialDeposit: 1000.0,
		},
	}, userToken, &createResult)
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
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Deposit",
				Data: bankaccount.DepositRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, userToken, &result)
		} else if op.Type == "withdraw" {
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Withdraw",
				Data: bankaccount.WithdrawRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, userToken, &result)
		}
		require.NoError(t, err)
	}

	// Verify final balance matches expected calculation
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	expectedBalance := 1000.0 + 500.0 - 200.0 + 300.0 - 100.0 // 1500.0
	assertBankAccountSuccess(t, balance, expectedBalance, "Final balance should match event sourcing calculation")
}

func testBankAccountAutomaticSnapshotCreation(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := fmt.Sprintf("account-snapshot-test-%d", time.Now().UnixNano()%10000)

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken(actorID, "snapshot-test-user", "snapshot@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test 1: Create account and perform enough operations to trigger snapshot creation
	// Default snapshot frequency is 10 events, so we'll create 15+ operations to ensure snapshots
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			OwnerName:      "Snapshot Test User",
			InitialDeposit: 1000.0,
		},
	}, userToken, &createResult)
	require.NoError(t, err, "Account creation should succeed")

	// Test 2: Perform multiple operations to trigger automatic snapshot creation
	// We'll do 15 operations total (including create) to exceed the default snapshot frequency of 10
	operations := []Operation{
		{Type: "deposit", Amount: 100.0, Description: "Op 1"},
		{Type: "withdraw", Amount: 50.0, Description: "Op 2"},
		{Type: "deposit", Amount: 200.0, Description: "Op 3"},
		{Type: "withdraw", Amount: 75.0, Description: "Op 4"},
		{Type: "deposit", Amount: 150.0, Description: "Op 5"},
		{Type: "withdraw", Amount: 25.0, Description: "Op 6"},
		{Type: "deposit", Amount: 300.0, Description: "Op 7"},
		{Type: "withdraw", Amount: 100.0, Description: "Op 8"},
		{Type: "deposit", Amount: 250.0, Description: "Op 9"},  // This should trigger first snapshot (version 10)
		{Type: "withdraw", Amount: 50.0, Description: "Op 10"},
		{Type: "deposit", Amount: 100.0, Description: "Op 11"},
		{Type: "withdraw", Amount: 25.0, Description: "Op 12"},
		{Type: "deposit", Amount: 175.0, Description: "Op 13"},
		{Type: "withdraw", Amount: 75.0, Description: "Op 14"}, // This should trigger second snapshot (version 20)
	}

	expectedBalance := 1000.0 // Starting balance
	for _, op := range operations {
		var result interface{}
		if op.Type == "deposit" {
			expectedBalance += op.Amount
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Deposit",
				Data: bankaccount.DepositRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, userToken, &result)
		} else if op.Type == "withdraw" {
			expectedBalance -= op.Amount
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Withdraw",
				Data: bankaccount.WithdrawRequest{
					Amount:      op.Amount,
					Description: op.Description,
				},
			}, userToken, &result)
		}
		require.NoError(t, err, "Operation %s should succeed", op.Description)
	}

	// Test 3: Verify balance is correct after all operations
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccess(t, balance, expectedBalance, "Balance should be correct after all operations with snapshots")

	// Test 4: Verify that operations work correctly after snapshot creation
	// This tests that state is correctly reconstructed from snapshots during replay
	// We don't have a GetHistory endpoint, but we can verify data consistency through balance checks

	// Test 5: Test that subsequent operations work correctly after snapshots exist
	// This tests the replay-from-snapshot functionality
	var additionalResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "Deposit",
		Data: bankaccount.DepositRequest{
			Amount:      500.0,
			Description: "Post-snapshot deposit",
		},
	}, userToken, &additionalResult)
	require.NoError(t, err, "Operations should work correctly after snapshot creation")

	// Test 6: Verify final balance includes the post-snapshot operation
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	finalExpectedBalance := expectedBalance + 500.0
	assertBankAccountSuccess(t, balance, finalExpectedBalance, "Final balance should include post-snapshot operations")

	// Test 7: Additional verification of snapshot-optimized replay consistency
	// Perform one more operation to ensure the state remains consistent
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "Withdraw",
		Data: bankaccount.WithdrawRequest{
			Amount:      100.0,
			Description: "Final consistency test",
		},
	}, userToken, &additionalResult)
	require.NoError(t, err, "Additional operations should work correctly with snapshots")

	// Final balance verification
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	finalBalance := finalExpectedBalance - 100.0
	assertBankAccountSuccess(t, balance, finalBalance, "Final balance should remain consistent with snapshot optimization")
}

func testBankAccountSnapshotPerformance(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := fmt.Sprintf("account-perf-test-%d", time.Now().UnixNano()%10000)

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken(actorID, "perf-test-user", "perf@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test 1: Create account
	var createResult interface{}
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: bankaccount.CreateAccountRequest{
			OwnerName:      "Performance Test User",
			InitialDeposit: 1000.0,
		},
	}, userToken, &createResult)
	require.NoError(t, err, "Account creation should succeed")

	// Test 2: Create exactly enough transactions to trigger multiple snapshots
	// Default snapshot frequency is 10, so we'll create 25 transactions total to get 2 snapshots
	numOperations := 24 // Plus account creation = 25 total events
	expectedBalance := 1000.0

	for i := 0; i < numOperations; i++ {
		var result interface{}
		if i%2 == 0 {
			// Deposit
			amount := 100.0
			expectedBalance += amount
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Deposit",
				Data: bankaccount.DepositRequest{
					Amount:      amount,
					Description: fmt.Sprintf("Perf test deposit %d", i),
				},
			}, userToken, &result)
		} else {
			// Withdraw
			amount := 50.0
			expectedBalance -= amount
			_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "BankAccount",
				ActorID:   actorID,
				Method:    "Withdraw",
				Data: bankaccount.WithdrawRequest{
					Amount:      amount,
					Description: fmt.Sprintf("Perf test withdrawal %d", i),
				},
			}, userToken, &result)
		}
		require.NoError(t, err, "Operation %d should succeed", i)
	}

	// Test 3: Verify balance consistency after many operations with snapshots
	var balance bankaccount.BankAccountState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccess(t, balance, expectedBalance, "Balance should be correct after many operations with multiple snapshots")

	// Test 4: Test rapid sequence of operations to ensure snapshots don't interfere with normal operations
	start := time.Now()
	rapidOps := 5
	for i := 0; i < rapidOps; i++ {
		var result interface{}
		_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   actorID,
			Method:    "Deposit",
			Data: bankaccount.DepositRequest{
				Amount:      10.0,
				Description: fmt.Sprintf("Rapid op %d", i),
			},
		}, userToken, &result)
		require.NoError(t, err, "Rapid operation %d should succeed", i)
		expectedBalance += 10.0
	}
	elapsed := time.Since(start)

	// Verify operations complete in reasonable time (should be fast due to snapshots)
	assert.Less(t, elapsed, 5*time.Second, "Rapid operations should complete quickly with snapshot optimization")

	// Test 5: Final balance verification
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   actorID,
		Method:    "GetBalance",
	}, userToken, &balance)
	require.NoError(t, err)
	assertBankAccountSuccess(t, balance, expectedBalance, "Final balance should be correct after performance test")

	t.Logf("Performance test completed: %d total operations in %.2f seconds with snapshot optimization", 
		numOperations+rapidOps, elapsed.Seconds())
}

// Operation represents a bank account operation
type Operation struct {
	Type        string
	Amount      float64
	Description string
}