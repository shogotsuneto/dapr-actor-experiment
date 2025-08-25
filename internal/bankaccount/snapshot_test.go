package bankaccount

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shogotsuneto/go-eventsourced/locked"
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
	
	// Initialize state manager and set up test state through proper event application
	zero := &BankAccountStateV1{
		AccountId: "test-account",
		Balance:   0,
		IsActive:  true,
	}
	actor.stateManager = locked.New(zero)
	
	// Apply an AccountCreated event to set up initial state
	createEvent := AccountCreatedEventV1{
		OwnerName:      "Test User",
		OwnerId:        "test-user",
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
		Version:        1, // Set version for the event
	}
	
	require.NoError(t, actor.stateManager.Apply(createEvent))
	
	// Also append the event to the event store to maintain consistency
	eventBytes, err := json.Marshal(createEvent)
	require.NoError(t, err)
	
	storeEvent := eventstore.Event{
		ID:        "test-event-1",
		Type:      createEvent.Type(),
		Data:      eventBytes,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"actorType": ActorTypeBankAccount,
			"actorId":   "test-account",
		},
	}
	
	_, err = trackedStore.Append("bankaccount-test-account", []eventstore.Event{storeEvent}, 0)
	require.NoError(t, err)
	
	// Update the actor's stream version to match the actual stream
	actor.streamVersion = 1

	// Test creating a snapshot
	err = actor.createSnapshot(ctx)
	require.NoError(t, err, "Should be able to create snapshot")

	// Verify snapshot was stored by directly loading it (snapshot creation itself doesn't use Load)
	// This verifies the snapshot was correctly stored in the event store
	var events []eventstore.Event
	events, err = trackedStore.Load("bankaccount-test-account", eventstore.LoadOptions{
		ExclusiveStartVersion: 0,
		Limit:                 0,
		Desc:                  false,
	})
	require.NoError(t, err)
	
	// Filter to only snapshot events
	var snapshotEvents []eventstore.Event
	for _, event := range events {
		if event.Type == string(EventTypeStateSnapshotV1) {
			snapshotEvents = append(snapshotEvents, event)
		}
	}
	
	require.Len(t, snapshotEvents, 1, "Should have one snapshot event")
	assert.Equal(t, string(EventTypeStateSnapshotV1), snapshotEvents[0].Type)
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
	event1 := AccountCreatedEventV1{
		OwnerName:      "Test User",
		OwnerId:        "test-user",
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
	}
	appendedEvent1, err := actor.appendEvent(ctx, event1)
	require.NoError(t, err)

	// Initialize state manager and apply the first event so the version is tracked properly
	zero := &BankAccountStateV1{
		AccountId: "test-account-replay",
		Balance:   0,
		IsActive:  true,
	}
	actor.stateManager = locked.New(zero)
	event1.Version = appendedEvent1.Version // Use the actual version from eventstore
	err = actor.stateManager.Apply(event1)
	require.NoError(t, err)
	actor.stateLoaded = true

	// Event 2: Money deposited
	event2 := MoneyDepositedEventV1{
		Amount:      50.0,
		Description: "Test deposit",
		Timestamp:   time.Now(),
	}
	appendedEvent2, err := actor.appendEvent(ctx, event2)
	require.NoError(t, err)

	// Update state to reflect the deposit
	event2.Version = appendedEvent2.Version // Use the actual version from eventstore
	err = actor.stateManager.Apply(event2)
	require.NoError(t, err)

	// Create a snapshot manually (balance should be 150.0)
	err = actor.createSnapshot(ctx)
	require.NoError(t, err)

	// Event 3: Money withdrawn (after snapshot)
	event3 := MoneyWithdrawnEventV1{
		Amount:      25.0,
		Description: "Test withdrawal",
		Timestamp:   time.Now(),
	}
	_, err = actor.appendEvent(ctx, event3)
	require.NoError(t, err)

	// Clear load calls and reset state to simulate actor reactivation
	trackedStore.ClearLoadCalls()
	actor.stateManager = nil
	actor.stateLoaded = false

	// Now test replay from snapshot - this should trigger Load calls
	err = actor.computeStateFromEvents(ctx)
	require.NoError(t, err)
	require.NotNil(t, actor.stateManager)

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
		if event.Type == string(EventTypeStateSnapshotV1) {
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
	// Should start from the snapshot data version (2), because snapshots can be taken concurrently
	// and the snapshot event version (3) may be different from the data version it represents
	assert.Equal(t, int64(2), eventsCall.Options.ExclusiveStartVersion, "Should start loading from snapshot data version")
	assert.Equal(t, 0, eventsCall.Options.Limit) // No limit for event replay
	assert.False(t, eventsCall.Options.Desc, "Should load events in chronological order")
	require.NoError(t, eventsCall.Error)

	// Should find the withdrawal event (version 4) after the snapshot
	withdrawalFound := false
	for _, event := range eventsCall.Events {
		if event.Type == string(EventTypeMoneyWithdrawnV1) {
			withdrawalFound = true
			assert.Equal(t, int64(4), event.Version, "Withdrawal should be at version 4")
			break
		}
	}
	assert.True(t, withdrawalFound, "Should have found the withdrawal event after snapshot")

	// Verify final state is correct
	// Balance should be 150.0 (from snapshot) - 25.0 (withdrawal) = 125.0
	finalState := actor.stateManager.GetState()
	assert.Equal(t, 125.0, finalState.Balance, "Balance should reflect snapshot + events after snapshot")
	assert.Equal(t, "Test User", finalState.OwnerName, "Owner name should be restored from snapshot")
	assert.Equal(t, "test-user", finalState.OwnerId, "Owner ID should be restored from snapshot")
	assert.True(t, finalState.IsActive, "Account should be active")
	assert.Equal(t, int64(4), finalState.Version, "Version should reflect all events including snapshot")
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
	event1 := AccountCreatedEventV1{
		OwnerName:      "Test User",
		OwnerId:        "test-user",
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
	}
	_, err = actor.appendEvent(ctx, event1)
	require.NoError(t, err)

	// Still no snapshot
	snapshot, err = actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	assert.Nil(t, snapshot, "Should not find snapshot among regular events")

	// Clear calls and create a snapshot
	trackedStore.ClearLoadCalls()

	zero := &BankAccountStateV1{
		AccountId: "test-account-find",
		Balance:   0,
		IsActive:  true,
	}
	actor.stateManager = locked.New(zero)
	
	// Load the events that were already appended and apply them to sync the state manager
	err = actor.computeStateFromEvents(ctx)
	require.NoError(t, err)
	err = actor.createSnapshot(ctx)
	require.NoError(t, err)

	// Clear calls and now should find the snapshot
	trackedStore.ClearLoadCalls()

	snapshot, err = actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, snapshot, "Should find the snapshot")
	assert.Equal(t, string(EventTypeStateSnapshotV1), snapshot.Type)

	// Test multiple snapshots - should return the latest one
	trackedStore.ClearLoadCalls()

	// Create second snapshot with different state
	// The stream now has: 1 event + 1 snapshot = version 2
	// We need to update the state to reflect some changes first
	event2 := MoneyDepositedEventV1{
		Amount:      50.0,
		Description: "Another deposit",
		Timestamp:   time.Now(),
	}
	appendedEvent2, err := actor.appendEvent(ctx, event2)
	require.NoError(t, err)

	event2.Version = appendedEvent2.Version // Use the actual version from eventstore
	err = actor.stateManager.Apply(event2)
	require.NoError(t, err)
	err = actor.createSnapshot(ctx)
	require.NoError(t, err, "Should create second snapshot")

	// Add more events  
	event3 := MoneyDepositedEventV1{
		Amount:      50.0,
		Description: "Third deposit",
		Timestamp:   time.Now(),
	}
	appendedEvent3, err := actor.appendEvent(ctx, event3)
	require.NoError(t, err)

	// Create third snapshot
	event3.Version = appendedEvent3.Version // Use the actual version from eventstore
	err = actor.stateManager.Apply(event3)
	require.NoError(t, err)
	err = actor.createSnapshot(ctx)
	require.NoError(t, err, "Should create third snapshot")

	// Clear calls and find latest snapshot
	trackedStore.ClearLoadCalls()

	latestSnapshot, err := actor.findLatestSnapshot(ctx)
	require.NoError(t, err)
	require.NotNil(t, latestSnapshot, "Should find the latest snapshot")
	assert.Equal(t, string(EventTypeStateSnapshotV1), latestSnapshot.Type)

	// Verify it's the latest snapshot by checking the version
	// The latest snapshot should have the highest version number
	// Parse the snapshot data to verify it contains the latest state
	var snapshotData StateSnapshotEventV1
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
		if event.Type == string(EventTypeStateSnapshotV1) {
			snapshotCount++
			// Parse each snapshot to track versions
			var data StateSnapshotEventV1
			if err := json.Unmarshal(event.Data, &data); err == nil {
				foundVersions = append(foundVersions, data.Version)
			}
		}
	}
	assert.GreaterOrEqual(t, snapshotCount, 2, "Should have found multiple snapshots in the stream")
	assert.Contains(t, foundVersions, int64(5), "Should have found the latest snapshot version")
}
