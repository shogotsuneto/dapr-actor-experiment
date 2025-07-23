package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestActorSnapshotIntegration demonstrates snapshot testing for fast execution
func TestActorSnapshotIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	composeFile := filepath.Join("..", "..", "docker-compose.yml")
	dockerManager := NewDockerComposeManager(composeFile)
	daprClient := NewDaprClient("http://localhost:3500")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start services
	require.NoError(t, dockerManager.StartServices(ctx), "Failed to start Docker services")
	defer func() {
		if err := dockerManager.StopServices(context.Background()); err != nil {
			t.Logf("Failed to stop services: %v", err)
		}
	}()

	WithSnapshotTesting(t, func(t *testing.T, snapshotter *SnapshotTester) {
		testCounterActorSnapshots(t, daprClient, snapshotter)
		testBankAccountActorSnapshots(t, daprClient, snapshotter)
	})
}

func testCounterActorSnapshots(t *testing.T, client *DaprClient, snapshotter *SnapshotTester) {
	ctx := context.Background()
	actorID := "snapshot-counter"

	// Test initial state
	resp, err := client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Get",
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "counter_initial_get", resp.Body)

	// Test increment
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Increment",
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "counter_increment", resp.Body)

	// Test set value
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Set",
		Data:      SetValueRequest{Value: 42},
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "counter_set_42", resp.Body)

	// Test decrement
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Decrement",
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "counter_decrement", resp.Body)
}

func testBankAccountActorSnapshots(t *testing.T, client *DaprClient, snapshotter *SnapshotTester) {
	ctx := context.Background()
	actorID := "snapshot-account"

	// Test create account
	resp, err := client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "CreateAccount",
		Data: CreateAccountRequest{
			OwnerName:      "Snapshot Test User",
			InitialDeposit: 1500.0,
		},
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "bank_create_account", resp.Body)

	// Test get balance
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "bank_initial_balance", resp.Body)

	// Test deposit
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "Deposit",
		Data: DepositRequest{
			Amount:      750.0,
			Description: "Snapshot test deposit",
		},
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "bank_deposit", resp.Body)

	// Test withdraw
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "Withdraw",
		Data: WithdrawRequest{
			Amount:      250.0,
			Description: "Snapshot test withdrawal",
		},
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "bank_withdraw", resp.Body)

	// Test final balance
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetBalance",
	})
	require.NoError(t, err)
	snapshotter.MatchJSONSnapshot(t, "bank_final_balance", resp.Body)

	// Test transaction history - note: this will contain timestamps so may need special handling
	resp, err = client.InvokeActorMethod(ctx, ActorMethodRequest{
		ActorType: "BankAccountActor",
		ActorID:   actorID,
		Method:    "GetHistory",
	})
	require.NoError(t, err)
	// For history, we'll create a snapshot but note it contains timestamps
	// In real-world usage, you might want to normalize timestamps for snapshot testing
	snapshotter.MatchJSONSnapshot(t, "bank_transaction_history", resp.Body)
}