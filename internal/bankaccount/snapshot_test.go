package bankaccount

import (
	"context"
	"testing"
	"time"

	"github.com/shogotsuneto/go-simple-eventstore"
	"github.com/shogotsuneto/go-simple-eventstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotCreationAndReplay(t *testing.T) {
	ctx := context.Background()
	eventStore := memory.NewInMemoryEventStore()
	
	// Create actor with low snapshot frequency for testing
	actor := NewBankAccount(eventStore)
	actor.SetID("test-account")
	actor.snapshotFrequency = 3 // Create snapshot every 3 events
	actor.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: "test-account",
			OwnerName: "Test User",
			OwnerId:   "test-user",
			Balance:   100.0,
			IsActive:  true,
			CreatedAt: time.Now().Format(time.RFC3339),
			Version:   0,
		},
	}
	
	// Test creating a snapshot
	err := actor.createSnapshot(ctx)
	require.NoError(t, err, "Should be able to create snapshot")
	
	// Verify snapshot was stored
	events, err := eventStore.Load("bankaccount-test-account", eventstore.LoadOptions{
		ExclusiveStartVersion: 0,
		Limit: 0,
		Desc:  false,
	})
	require.NoError(t, err)
	require.Len(t, events, 1, "Should have one snapshot event")
	assert.Equal(t, string(AccountEventEventTypeStateSnapshot), events[0].Type)
}

func TestSnapshotBasedReplay(t *testing.T) {
	ctx := context.Background()
	eventStore := memory.NewInMemoryEventStore()
	
	// Create test actor
	actor := NewBankAccount(eventStore)
	actor.SetID("test-account-replay")
	
	// Simulate some events followed by a snapshot
	// Event 1: Account created
	_, err := actor.appendEvent(ctx, AccountEventEventTypeAccountCreated, AccountCreatedEventData{
		OwnerName:      "Test User",
		OwnerId:        "test-user",
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
	})
	require.NoError(t, err)
	
	// Apply the first event to the state so the version is tracked properly
	actor.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: "test-account-replay",
			OwnerName: "Test User",
			OwnerId:   "test-user",
			Balance:   100.0,
			IsActive:  true,
			CreatedAt: time.Now().Format(time.RFC3339),
			Version:   1,
		},
	}
	actor.stateLoaded = true
	
	// Event 2: Money deposited
	_, err = actor.appendEvent(ctx, AccountEventEventTypeMoneyDeposited, MoneyDepositedEventData{
		Amount:      50.0,
		Description: "Test deposit",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)
	
	// Update state to reflect the deposit
	actor.state.Data.Balance = 150.0
	actor.state.Data.Version = 2
	
	// Create a snapshot manually (balance should be 150.0)
	err = actor.createSnapshot(ctx)
	require.NoError(t, err)
	
	// Update state version to reflect snapshot creation
	actor.state.Data.Version = 3
	
	// Event 3: Money withdrawn (after snapshot)
	_, err = actor.appendEvent(ctx, AccountEventEventTypeMoneyWithdrawn, MoneyWithdrawnEventData{
		Amount:      25.0,
		Description: "Test withdrawal",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)
	
	// Now test replay from snapshot
	actor.state = nil
	actor.stateLoaded = false
	
	state, err := actor.computeStateFromEvents(ctx)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.Data)
	
	// Balance should be 150.0 (from snapshot) - 25.0 (withdrawal) = 125.0
	assert.Equal(t, 125.0, state.Data.Balance, "Balance should reflect snapshot + events after snapshot")
	assert.Equal(t, "Test User", state.Data.OwnerName, "Owner name should be restored from snapshot")
	assert.Equal(t, "test-user", state.Data.OwnerId, "Owner ID should be restored from snapshot")
	assert.True(t, state.Data.IsActive, "Account should be active")
	assert.Equal(t, int64(4), state.Data.Version, "Version should reflect all events including snapshot")
}

func TestFindLatestSnapshot(t *testing.T) {
	ctx := context.Background()
	eventStore := memory.NewInMemoryEventStore()
	
	actor := NewBankAccount(eventStore)
	actor.SetID("test-account-find")
	
	// Initially no snapshot should be found
	snapshot, err := actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	assert.Nil(t, snapshot, "Should not find snapshot when none exists")
	
	// Add some regular events
	_, err = actor.appendEvent(ctx, AccountEventEventTypeAccountCreated, AccountCreatedEventData{
		OwnerName:      "Test User",
		OwnerId:        "test-user", 
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
	})
	require.NoError(t, err)
	
	// Still no snapshot
	snapshot, err = actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	assert.Nil(t, snapshot, "Should not find snapshot among regular events")
	
	// Create a snapshot
	actor.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: "test-account-find",
			Balance:   100.0,
			Version:   1,
		},
	}
	err = actor.createSnapshot(ctx)
	require.NoError(t, err)
	
	// Now should find the snapshot
	snapshot, err = actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, snapshot, "Should find the snapshot")
	assert.Equal(t, string(AccountEventEventTypeStateSnapshot), snapshot.Type)
}