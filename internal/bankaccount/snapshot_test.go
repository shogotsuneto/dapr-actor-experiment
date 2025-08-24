package bankaccount

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	eventstore "github.com/shogotsuneto/go-simple-eventstore"
	"github.com/shogotsuneto/go-simple-eventstore/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LoadCall represents a tracked eventstore.Load call
type LoadCall struct {
	StreamID string
	Options  eventstore.LoadOptions
	Events   []eventstore.Event
	Error    error
}

// TrackedEventStore wraps an eventstore and tracks all Load calls
type TrackedEventStore struct {
	eventstore.EventStore
	LoadCalls []LoadCall
}

// NewTrackedEventStore creates a new tracked event store wrapper
func NewTrackedEventStore(store eventstore.EventStore) *TrackedEventStore {
	return &TrackedEventStore{
		EventStore: store,
		LoadCalls:  make([]LoadCall, 0),
	}
}

// Load wraps the underlying Load call and tracks it
func (t *TrackedEventStore) Load(streamID string, options eventstore.LoadOptions) ([]eventstore.Event, error) {
	events, err := t.EventStore.Load(streamID, options)

	// Track this call
	call := LoadCall{
		StreamID: streamID,
		Options:  options,
		Events:   events,
		Error:    err,
	}
	t.LoadCalls = append(t.LoadCalls, call)

	return events, err
}

// GetLoadCalls returns all tracked Load calls
func (t *TrackedEventStore) GetLoadCalls() []LoadCall {
	return t.LoadCalls
}

// ClearLoadCalls resets the tracked calls
func (t *TrackedEventStore) ClearLoadCalls() {
	t.LoadCalls = make([]LoadCall, 0)
}

func TestSnapshotCreationAndReplay(t *testing.T) {
	ctx := context.Background()
	memoryStore := memory.NewInMemoryEventStore()
	trackedStore := NewTrackedEventStore(memoryStore)

	// Create actor with low snapshot frequency for testing
	actor := NewBankAccount(trackedStore)
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

	// Verify snapshot was stored by directly loading it (snapshot creation itself doesn't use Load)
	// This verifies the snapshot was correctly stored in the event store
	events, err := trackedStore.Load("bankaccount-test-account", eventstore.LoadOptions{
		ExclusiveStartVersion: 0,
		Limit:                 0,
		Desc:                  false,
	})
	require.NoError(t, err)
	require.Len(t, events, 1, "Should have one snapshot event")
	assert.Equal(t, string(AccountEventEventTypeStateSnapshot), events[0].Type)
}

func TestSnapshotBasedReplay(t *testing.T) {
	ctx := context.Background()
	memoryStore := memory.NewInMemoryEventStore()
	trackedStore := NewTrackedEventStore(memoryStore)

	// Create test actor
	actor := NewBankAccount(trackedStore)
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

	// Clear load calls and reset state to simulate actor reactivation
	trackedStore.ClearLoadCalls()
	actor.state = nil
	actor.stateLoaded = false

	// Now test replay from snapshot - this should trigger Load calls
	state, err := actor.computeStateFromEvents(ctx)
	require.NoError(t, err)
	require.NotNil(t, state)
	require.NotNil(t, state.Data)

	// Verify the Load calls made during state computation
	loadCalls := trackedStore.GetLoadCalls()
	require.Len(t, loadCalls, 2, "Should have made exactly two Load calls: one for snapshot discovery, one for events after snapshot")

	// First call should be for finding the latest snapshot (reverse order)
	snapshotCall := loadCalls[0]
	assert.Equal(t, "bankaccount-test-account-replay", snapshotCall.StreamID)
	assert.Equal(t, int64(0), snapshotCall.Options.ExclusiveStartVersion)
	assert.Equal(t, 100, snapshotCall.Options.Limit) // Limited search for snapshots
	assert.True(t, snapshotCall.Options.Desc, "Should load in reverse order to find latest snapshot")
	require.NoError(t, snapshotCall.Error)

	// Should find the snapshot among the returned events
	snapshotFound := false
	snapshotVersion := int64(0)
	for _, event := range snapshotCall.Events {
		if event.Type == string(AccountEventEventTypeStateSnapshot) {
			snapshotFound = true
			snapshotVersion = event.Version
			break
		}
	}
	assert.True(t, snapshotFound, "Should have found the snapshot event in the Load call")
	assert.Equal(t, int64(3), snapshotVersion, "Snapshot should be at version 3")

	// Second call should be for loading events after the snapshot
	eventsCall := loadCalls[1]
	assert.Equal(t, "bankaccount-test-account-replay", eventsCall.StreamID)
	// Should start from the snapshot data version (2), not the snapshot event version (3)
	assert.Equal(t, int64(2), eventsCall.Options.ExclusiveStartVersion, "Should start loading from snapshot data version")
	assert.Equal(t, 0, eventsCall.Options.Limit) // No limit for event replay
	assert.False(t, eventsCall.Options.Desc, "Should load events in chronological order")
	require.NoError(t, eventsCall.Error)

	// Should find the withdrawal event (version 4) after the snapshot
	withdrawalFound := false
	for _, event := range eventsCall.Events {
		if event.Type == string(AccountEventEventTypeMoneyWithdrawn) {
			withdrawalFound = true
			assert.Equal(t, int64(4), event.Version, "Withdrawal should be at version 4")
			break
		}
	}
	assert.True(t, withdrawalFound, "Should have found the withdrawal event after snapshot")

	// Verify final state is correct
	// Balance should be 150.0 (from snapshot) - 25.0 (withdrawal) = 125.0
	assert.Equal(t, 125.0, state.Data.Balance, "Balance should reflect snapshot + events after snapshot")
	assert.Equal(t, "Test User", state.Data.OwnerName, "Owner name should be restored from snapshot")
	assert.Equal(t, "test-user", state.Data.OwnerId, "Owner ID should be restored from snapshot")
	assert.True(t, state.Data.IsActive, "Account should be active")
	assert.Equal(t, int64(4), state.Data.Version, "Version should reflect all events including snapshot")
}

func TestFindLatestSnapshot(t *testing.T) {
	ctx := context.Background()
	memoryStore := memory.NewInMemoryEventStore()
	trackedStore := NewTrackedEventStore(memoryStore)

	actor := NewBankAccount(trackedStore)
	actor.SetID("test-account-find")

	// Initially no snapshot should be found
	snapshot, err := actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	assert.Nil(t, snapshot, "Should not find snapshot when none exists")

	// Verify the Load call for finding snapshot when none exists
	loadCalls := trackedStore.GetLoadCalls()
	require.Len(t, loadCalls, 1, "Should have made one Load call to search for snapshots")

	call := loadCalls[0]
	assert.Equal(t, "bankaccount-test-account-find", call.StreamID)
	assert.Equal(t, int64(0), call.Options.ExclusiveStartVersion)
	assert.Equal(t, 100, call.Options.Limit)
	assert.True(t, call.Options.Desc, "Should search in reverse order")
	require.NoError(t, call.Error)
	assert.Empty(t, call.Events, "Should find no events when stream is empty")

	// Clear calls for next test
	trackedStore.ClearLoadCalls()

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

	// Clear calls and create a snapshot
	trackedStore.ClearLoadCalls()

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

	// Update state version to reflect snapshot creation (snapshot event added to stream)
	actor.state.Data.Version = 2

	// Clear calls and now should find the snapshot
	trackedStore.ClearLoadCalls()

	snapshot, err = actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, snapshot, "Should find the snapshot")
	assert.Equal(t, string(AccountEventEventTypeStateSnapshot), snapshot.Type)

	// Test multiple snapshots - should return the latest one
	trackedStore.ClearLoadCalls()

	// Create second snapshot with different state
	// The stream now has: 1 event + 1 snapshot = version 2
	// We need to update the state to reflect some changes first
	_, err = actor.appendEvent(ctx, AccountEventEventTypeMoneyDeposited, MoneyDepositedEventData{
		Amount:      50.0,
		Description: "Another deposit",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	actor.state.Data.Balance = 150.0
	actor.state.Data.Version = 3 // After the new deposit event
	err = actor.createSnapshot(ctx)
	require.NoError(t, err, "Should create second snapshot")

	// Update state version after second snapshot creation
	actor.state.Data.Version = 4

	// Add more events  
	_, err = actor.appendEvent(ctx, AccountEventEventTypeMoneyDeposited, MoneyDepositedEventData{
		Amount:      50.0,
		Description: "Third deposit",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)

	// Create third snapshot
	actor.state.Data.Balance = 200.0
	actor.state.Data.Version = 5 // After the third deposit event
	err = actor.createSnapshot(ctx)
	require.NoError(t, err, "Should create third snapshot")

	// Clear calls and find latest snapshot
	trackedStore.ClearLoadCalls()

	latestSnapshot, err := actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot, "Should find the latest snapshot")
	assert.Equal(t, string(AccountEventEventTypeStateSnapshot), latestSnapshot.Type)

	// Verify it's the latest snapshot by checking the version
	// The latest snapshot should have the highest version number
	// Parse the snapshot data to verify it contains the latest state
	var snapshotData StateSnapshotEventData
	err = json.Unmarshal(latestSnapshot.Data, &snapshotData)
	require.NoError(t, err, "Should parse snapshot data")
	assert.Equal(t, int64(5), snapshotData.Version, "Should return snapshot with highest version (latest)")
	assert.Equal(t, 200.0, snapshotData.Balance, "Should return snapshot with latest balance")

	// Verify the Load call was made correctly
	loadCalls = trackedStore.GetLoadCalls()
	require.Len(t, loadCalls, 1, "Should have made one Load call for finding latest snapshot")

	call = loadCalls[0]
	assert.Equal(t, "bankaccount-test-account-find", call.StreamID)
	assert.Equal(t, int64(0), call.Options.ExclusiveStartVersion)
	assert.Equal(t, 100, call.Options.Limit)
	assert.True(t, call.Options.Desc, "Should search in reverse order to find latest first")
	require.NoError(t, call.Error)

	// Verify that multiple snapshots were returned but the method picked the latest
	snapshotCount := 0
	var foundVersions []int64
	for _, event := range call.Events {
		if event.Type == string(AccountEventEventTypeStateSnapshot) {
			snapshotCount++
			// Parse each snapshot to track versions
			var data StateSnapshotEventData
			if err := json.Unmarshal(event.Data, &data); err == nil {
				foundVersions = append(foundVersions, data.Version)
			}
		}
	}
	assert.GreaterOrEqual(t, snapshotCount, 2, "Should have found multiple snapshots in the stream")
	assert.Contains(t, foundVersions, int64(5), "Should have found the latest snapshot version")
}
