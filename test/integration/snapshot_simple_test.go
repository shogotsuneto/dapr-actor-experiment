package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSnapshotFunctionalityAssumeRunning demonstrates snapshot testing when services are already running
func TestSnapshotFunctionalityAssumeRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test snapshot functionality with already running services
	daprClient := NewDaprClient("http://localhost:3500")

	WithSnapshotTesting(t, func(t *testing.T, snapshotter *SnapshotTester) {
		ctx := context.Background()
		actorID := "snapshot-demo"

		// Test counter snapshots
		resp, err := daprClient.InvokeActorMethod(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Get",
		})
		require.NoError(t, err)
		snapshotter.MatchJSONSnapshot(t, "counter_get_initial", resp.Body)

		// Increment and snapshot
		resp, err = daprClient.InvokeActorMethod(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Increment",
		})
		require.NoError(t, err)
		snapshotter.MatchJSONSnapshot(t, "counter_increment_demo", resp.Body)

		// Set value and snapshot
		resp, err = daprClient.InvokeActorMethod(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Set",
			Data:      SetValueRequest{Value: 123},
		})
		require.NoError(t, err)
		snapshotter.MatchJSONSnapshot(t, "counter_set_123", resp.Body)

		// Test bank account snapshots
		bankActorID := "snapshot-demo-bank"
		
		resp, err = daprClient.InvokeActorMethod(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   bankActorID,
			Method:    "CreateAccount",
			Data: CreateAccountRequest{
				OwnerName:      "Snapshot Demo User",
				InitialDeposit: 1000.0,
			},
		})
		require.NoError(t, err)
		snapshotter.MatchJSONSnapshot(t, "bank_create_demo", resp.Body)

		resp, err = daprClient.InvokeActorMethod(ctx, ActorMethodRequest{
			ActorType: "BankAccountActor",
			ActorID:   bankActorID,
			Method:    "GetBalance",
		})
		require.NoError(t, err)
		snapshotter.MatchJSONSnapshot(t, "bank_balance_demo", resp.Body)
	})
}