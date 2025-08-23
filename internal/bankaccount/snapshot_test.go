package bankaccount

import (
	"context"
	"testing"
	"time"

	"github.com/shogotsuneto/go-simple-eventstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEventStore provides a simple in-memory event store for testing
type MockEventStore struct {
	streams map[string][]eventstore.Event
	version int64
}

func NewMockEventStore() *MockEventStore {
	return &MockEventStore{
		streams: make(map[string][]eventstore.Event),
		version: 0,
	}
}

func (m *MockEventStore) Append(streamID string, events []eventstore.Event, expectedVersion int) (int64, error) {
	if _, exists := m.streams[streamID]; !exists {
		m.streams[streamID] = make([]eventstore.Event, 0)
	}
	
	for i := range events {
		m.version++
		events[i].Version = m.version
		m.streams[streamID] = append(m.streams[streamID], events[i])
	}
	
	return m.version, nil
}

func (m *MockEventStore) Load(streamID string, options eventstore.LoadOptions) ([]eventstore.Event, error) {
	events, exists := m.streams[streamID]
	if !exists {
		return []eventstore.Event{}, nil
	}
	
	// Filter events based on LoadOptions
	var result []eventstore.Event
	
	if options.Desc {
		// Reverse order
		for i := len(events) - 1; i >= 0; i-- {
			event := events[i]
			if options.ExclusiveStartVersion == 0 || event.Version < options.ExclusiveStartVersion {
				result = append(result, event)
				if options.Limit > 0 && len(result) >= options.Limit {
					break
				}
			}
		}
	} else {
		// Forward order
		for _, event := range events {
			if event.Version > options.ExclusiveStartVersion {
				result = append(result, event)
				if options.Limit > 0 && len(result) >= options.Limit {
					break
				}
			}
		}
	}
	
	return result, nil
}

func TestSnapshotCreationAndReplay(t *testing.T) {
	ctx := context.Background()
	mockStore := NewMockEventStore()
	
	// Create actor with low snapshot frequency for testing
	actor := NewBankAccount(mockStore)
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
	events, err := mockStore.Load("bankaccount-test-account", eventstore.LoadOptions{
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
	mockStore := NewMockEventStore()
	
	// Create test actor
	actor := NewBankAccount(mockStore)
	actor.SetID("test-account")
	
	// Simulate some events followed by a snapshot
	// Event 1: Account created
	_, err := actor.appendEvent(ctx, AccountEventEventTypeAccountCreated, AccountCreatedEventData{
		OwnerName:      "Test User",
		OwnerId:        "test-user",
		InitialDeposit: 100.0,
		CreatedAt:      time.Now(),
	})
	require.NoError(t, err)
	
	// Event 2: Money deposited
	_, err = actor.appendEvent(ctx, AccountEventEventTypeMoneyDeposited, MoneyDepositedEventData{
		Amount:      50.0,
		Description: "Test deposit",
		Timestamp:   time.Now(),
	})
	require.NoError(t, err)
	
	// Manually create a snapshot at this point (balance should be 150.0)
	actor.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: "test-account",
			OwnerName: "Test User", 
			OwnerId:   "test-user",
			Balance:   150.0,
			IsActive:  true,
			CreatedAt: time.Now().Format(time.RFC3339),
			Version:   2,
		},
	}
	err = actor.createSnapshot(ctx)
	require.NoError(t, err)
	
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
	actor.accountExists = false
	
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
	mockStore := NewMockEventStore()
	
	actor := NewBankAccount(mockStore)
	actor.SetID("test-account")
	
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
			AccountId: "test-account",
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