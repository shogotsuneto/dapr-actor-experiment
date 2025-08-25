package bankaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dapr/go-sdk/actor"
	"github.com/google/uuid"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
	"github.com/shogotsuneto/go-eventsourced"
	"github.com/shogotsuneto/go-eventsourced/locked"
	eventstore "github.com/shogotsuneto/go-simple-eventstore"
)

// BankAccount demonstrates event sourcing pattern with external postgres event store.
// This actor stores events using go-simple-eventstore/postgres for durability and audit trail,
// while maintaining fast access through ephemeral in-memory state cache as long as the actor is activated.
// 
// State management is now handled by LockedStateManager using the go-eventsourced/locked library for 
// proper separation of concerns and thread-safe operations.
type BankAccount struct {
	actor.ServerImplBaseCtx

	// Reference to shared external event store (singleton)
	eventStore eventstore.EventStore

	// Stream ID for this actor's events
	streamID string

	// Locked state manager using go-eventsourced/locked library
	stateManager *locked.LockedES[*BankAccountStateV1]
	stateLoaded  bool // Track if state has been loaded from events

	// Snapshot configuration
	snapshotFrequency int64 // Create snapshot every N events (default: 10)
}

// getStreamID returns the stream ID for this actor's business events, initializing it if needed
func (b *BankAccount) getStreamID() string {
	if b.streamID == "" {
		b.streamID = fmt.Sprintf("bankaccount-%s", b.ID())
	}
	return b.streamID
}

// getSnapshotStreamID returns the stream ID for this actor's snapshot events
func (b *BankAccount) getSnapshotStreamID() string {
	return fmt.Sprintf("bankaccount-%s-snapshots", b.ID())
}

// getCurrentVersion returns the current stream version from state, or 0 if no state exists
func (b *BankAccount) getCurrentVersion() int64 {
	if b.stateManager != nil {
		return b.stateManager.GetState().Version
	}
	return 0
}

// NewBankAccount creates a new BankAccount actor instance with access to the provided event store.
func NewBankAccount(eventStore eventstore.EventStore) *BankAccount {
	if eventStore == nil {
		log.Printf("WARNING: BankAccount created without event store. External persistence disabled.")
	}

	return &BankAccount{
		eventStore:        eventStore,
		snapshotFrequency: 10, // Default: create snapshot every 10 events
	}
}

func (b *BankAccount) Type() string {
	return ActorTypeBankAccount
}

// ensureStateLoaded loads and caches state from event store if not already loaded.
// This provides fast in-memory access while maintaining event sourcing benefits.
//
// PERFORMANCE: This method implements lazy loading - state is computed from events
// only once when the actor is first accessed, then cached for subsequent operations.
func (b *BankAccount) ensureStateLoaded(ctx context.Context) error {
	// Check if already loaded (simple check without locking)
	if b.stateLoaded {
		return nil // State already loaded and cached - fast path!
	}

	if b.eventStore == nil {
		return fmt.Errorf("event store not configured")
	}

	// Load state from event store for the first time (expensive operation)
	err := b.computeStateFromEvents(ctx)
	if err != nil {
		return err
	}

	b.stateLoaded = true
	return nil
}

// checkOwnership verifies that the user can access this account
func (b *BankAccount) checkOwnership(ctx context.Context) (string, error) {
	userID, ok := auth.GetUserID(ctx)
	if !ok {
		return "", fmt.Errorf("authentication required")
	}

	// For account creation, the actor ID should match the user ID (simplified ownership check)
	// This means users can only create accounts that match their user ID
	if b.stateManager == nil || !b.stateLoaded {
		if userID != b.ID() {
			return "", fmt.Errorf("insufficient permissions: can only create accounts for yourself")
		}
		return userID, nil
	}

	// For existing accounts, check against the stored owner ID
	state := b.stateManager.GetState()
	ownerID := state.OwnerId

	if ownerID != userID {
		return "", fmt.Errorf("insufficient permissions: cannot access this account")
	}

	return userID, nil
}

// Helper methods for structured responses

func (b *BankAccount) successResponse() *BankAccountState {
	if b.stateManager == nil {
		return &BankAccountState{Success: false}
	}

	// Get current state from state manager
	currentState := b.stateManager.GetState()
	return b.successResponseWithStateV1(*currentState)
}

func (b *BankAccount) successResponseWithStateV1(stateData BankAccountStateV1) *BankAccountState {
	return &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: stateData.AccountId,
			OwnerName: stateData.OwnerName,
			OwnerId:   stateData.OwnerId,
			Balance:   stateData.Balance,
			IsActive:  stateData.IsActive,
			CreatedAt: stateData.CreatedAt,
			Version:   stateData.Version,
		},
		// Don't set Error field - omitempty will exclude it from JSON
	}
}

func (b *BankAccount) successResponseWithState(stateData *BankAccountStateData) *BankAccountState {
	return &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: stateData.AccountId,
			OwnerName: stateData.OwnerName,
			OwnerId:   stateData.OwnerId,
			Balance:   stateData.Balance,
			IsActive:  stateData.IsActive,
			CreatedAt: stateData.CreatedAt,
			Version:   stateData.Version,
		},
		// Don't set Error field - omitempty will exclude it from JSON
	}
}

func (b *BankAccount) errorResponse(code ErrorCode, message string, details map[string]interface{}) *BankAccountState {
	return &BankAccountState{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
		// Don't set Data field - omitempty will exclude it from JSON
	}
}

func (b *BankAccount) CreateAccount(ctx context.Context, request CreateAccountRequest) (*BankAccountState, error) {
	// Check ownership and get user ID
	userID, err := b.checkOwnership(ctx)
	if err != nil {
		log.Printf("BankAccount %s: User access denied - %v", b.ID(), err)
		return b.errorResponse(ErrorCodeAuthorizationError, err.Error(), nil), nil
	}

	// Ensure state is loaded
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Check if account already exists (fast in-memory check)
	if b.stateManager != nil && b.stateLoaded {
		return b.errorResponse(ErrorCodeAccountAlreadyExists, "Account already exists", map[string]interface{}{
			"accountId": b.ID(),
		}), nil
	}

	// Validate request
	if request.OwnerName == "" {
		return b.errorResponse(ErrorCodeValidationError, "Owner name is required", nil), nil
	}
	if request.InitialDeposit < 0 {
		return b.errorResponse(ErrorCodeValidationError, "Initial deposit cannot be negative", map[string]interface{}{
			"providedAmount": request.InitialDeposit,
		}), nil
	}

	// Create and store versioned event for durability (include creator info)
	eventData := AccountCreatedEventV1{
		OwnerName:      request.OwnerName,
		OwnerId:        userID,
		InitialDeposit: request.InitialDeposit,
		CreatedAt:      time.Now(),
	}

	event, err := b.appendEvent(ctx, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to create account", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Set version on the event data for Apply
	eventData.Version = event.Version

	// Initialize state manager and apply the event using centralized logic
	if b.stateManager == nil {
		zero := &BankAccountStateV1{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		}
		b.stateManager = locked.New(zero)
	}

	if err := b.stateManager.Apply(eventData); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	log.Printf("BankAccount %s: Account created by user %s for owner %s", b.ID(), userID, request.OwnerName)
	return b.successResponseWithStateV1(*b.stateManager.GetState()), nil
}

func (b *BankAccount) Deposit(ctx context.Context, request DepositRequest) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse(ErrorCodeAuthorizationError, err.Error(), nil), nil
	}

	// Validate request
	if request.Amount <= 0 {
		return b.errorResponse(ErrorCodeValidationError, "Deposit amount must be positive", map[string]interface{}{
			"providedAmount": request.Amount,
		}), nil
	}

	// Ensure account exists
	if b.stateManager == nil || !b.stateLoaded {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}

	// Create and store versioned event for durability
	eventData := MoneyDepositedEventV1{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}

	event, err := b.appendEvent(ctx, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to record deposit", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Set version on the event data for Apply
	eventData.Version = event.Version

	// Update in-memory state using centralized event application
	if err := b.stateManager.Apply(eventData); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	return b.successResponseWithStateV1(*b.stateManager.GetState()), nil
}

func (b *BankAccount) Withdraw(ctx context.Context, request WithdrawRequest) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse(ErrorCodeAuthorizationError, err.Error(), nil), nil
	}

	// Validate request
	if request.Amount <= 0 {
		return b.errorResponse(ErrorCodeValidationError, "Withdrawal amount must be positive", map[string]interface{}{
			"providedAmount": request.Amount,
		}), nil
	}

	// Ensure account exists and check balance
	if b.stateManager == nil || !b.stateLoaded {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}

	// Check sufficient balance using fast in-memory state
	currentState := b.stateManager.GetState()
	currentBalance := currentState.Balance
	
	if currentBalance < request.Amount {
		return b.errorResponse(ErrorCodeInsufficientFunds, fmt.Sprintf("Insufficient funds: balance %.2f, requested %.2f", currentBalance, request.Amount), map[string]interface{}{
			"currentBalance":  currentBalance,
			"requestedAmount": request.Amount,
		}), nil
	}

	// Create and store versioned event for durability
	eventData := MoneyWithdrawnEventV1{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}

	event, err := b.appendEvent(ctx, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to record withdrawal", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Set version on the event data for Apply
	eventData.Version = event.Version

	// Update in-memory state using centralized event application
	if err := b.stateManager.Apply(eventData); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	return b.successResponseWithStateV1(*b.stateManager.GetState()), nil
}

func (b *BankAccount) GetBalance(ctx context.Context) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}

	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse(ErrorCodeAuthorizationError, err.Error(), nil), nil
	}

	// Check if account exists
	if b.stateManager == nil || !b.stateLoaded {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}

	// Return fast in-memory cached state
	return b.successResponse(), nil
}

// Event store implementation details

func (b *BankAccount) appendEvent(ctx context.Context, event eventsourced.Event) (*eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}

	// Convert event data to JSON
	dataBytes, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %v", err)
	}

	// Create event for store
	storeEvent := eventstore.Event{
		ID:        uuid.New().String(),
		Type:      event.Type(),
		Data:      dataBytes,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"actorType": ActorTypeBankAccount,
			"actorId":   b.ID(),
		},
	}

	// Get current stream version for optimistic concurrency control
	currentEvents, err := b.eventStore.Load(b.getStreamID(), eventstore.LoadOptions{
		Limit: 1,
		Desc:  true, // Get the latest event
	})
	
	currentStreamVersion := int64(0)
	if err == nil && len(currentEvents) > 0 {
		currentStreamVersion = currentEvents[0].Version
	}

	// Append to event store using stream ID based on actor ID with version check
	events := []eventstore.Event{storeEvent}
	_, err = b.eventStore.Append(b.getStreamID(), events, int(currentStreamVersion))
	if err != nil {
		return nil, err
	}

	// The event now has its version set by the event store
	appendedEvent := &events[0]

	// Check if we should create a snapshot (only for business events, not snapshots)
	if event.Type() != string(EventTypeStateSnapshotV1) &&
		b.snapshotFrequency > 0 &&
		appendedEvent.Version%b.snapshotFrequency == 0 {

		// Create snapshot asynchronously to avoid affecting the main operation
		go func() {
			// Give a short time for the current operation to complete and update state
			time.Sleep(100 * time.Millisecond)

			if err := b.createSnapshot(context.Background()); err != nil {
				log.Printf("BankAccount %s: Failed to create automatic snapshot at version %d: %v",
					b.ID(), appendedEvent.Version, err)
			}
		}()
	}

	return appendedEvent, nil
}

// appendSnapshotEvent appends a snapshot event to the separate snapshot stream
func (b *BankAccount) appendSnapshotEvent(ctx context.Context, event eventsourced.Event) (*eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}

	// Convert event data to JSON
	dataBytes, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot event data: %v", err)
	}

	// Create event for store
	storeEvent := eventstore.Event{
		ID:        uuid.New().String(),
		Type:      event.Type(),
		Data:      dataBytes,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"actorType": ActorTypeBankAccount,
			"actorId":   b.ID(),
		},
	}

	// Get current snapshot stream version for optimistic concurrency control
	currentEvents, err := b.eventStore.Load(b.getSnapshotStreamID(), eventstore.LoadOptions{
		Limit: 1,
		Desc:  true, // Get the latest snapshot
	})
	
	currentStreamVersion := int64(0)
	if err == nil && len(currentEvents) > 0 {
		currentStreamVersion = currentEvents[0].Version
	}

	// Append to snapshot stream using separate stream ID
	events := []eventstore.Event{storeEvent}
	_, err = b.eventStore.Append(b.getSnapshotStreamID(), events, int(currentStreamVersion))
	if err != nil {
		return nil, err
	}

	// The event now has its version set by the event store
	appendedEvent := &events[0]

	return appendedEvent, nil
}

// createSnapshot creates a snapshot of the current state and stores it as an event
func (b *BankAccount) createSnapshot(ctx context.Context) error {
	if b.stateManager == nil {
		return fmt.Errorf("cannot create snapshot: no state available")
	}

	// Get current state
	currentState := b.stateManager.GetState()

	// Parse the CreatedAt timestamp back to time.Time for the snapshot
	createdAt, err := time.Parse(time.RFC3339, currentState.CreatedAt)
	if err != nil {
		// If parsing fails, use current time as fallback
		createdAt = time.Now()
	}

	// Create snapshot event
	snapshotEvent := StateSnapshotEventV1{
		AccountId: currentState.AccountId,
		OwnerName: currentState.OwnerName,
		OwnerId:   currentState.OwnerId,
		Balance:   currentState.Balance,
		IsActive:  currentState.IsActive,
		CreatedAt: createdAt,
		Version:   currentState.Version,
		Timestamp: time.Now(),
	}

	_, err = b.appendSnapshotEvent(ctx, snapshotEvent)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	log.Printf("BankAccount %s: Created snapshot at version %d", b.ID(), currentState.Version)
	return nil
}

// findLatestSnapshot finds the most recent snapshot event using reverse loading from the snapshot stream
func (b *BankAccount) findLatestSnapshot(ctx context.Context) (*eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}

	// Load events in reverse order from the snapshot stream to find the latest snapshot quickly
	events, err := b.eventStore.Load(b.getSnapshotStreamID(), eventstore.LoadOptions{
		ExclusiveStartVersion: 0,    // Start from latest
		Limit:                 100,  // Reasonable limit to avoid loading too many events
		Desc:                  true, // Reverse order (latest first)
	})

	if err != nil {
		return nil, err
	}

	// Find the first (latest) snapshot event - should be the first one since it's a snapshot-only stream
	for _, event := range events {
		if EventTypeV1(event.Type) == EventTypeStateSnapshotV1 {
			return &event, nil
		}
	}

	return nil, nil // No snapshot found
}

func (b *BankAccount) computeStateFromEvents(ctx context.Context) error {
	if b.eventStore == nil {
		return fmt.Errorf("event store not configured")
	}

	// Step 1: Try to restore state from snapshot
	if err := b.restoreFromSnapshot(ctx); err != nil {
		return fmt.Errorf("failed to restore from snapshot: %v", err)
	}

	// Step 2: Replay events after the snapshot
	if err := b.replayEventsAfterVersion(ctx); err != nil {
		return fmt.Errorf("failed to replay events: %v", err)
	}

	return nil
}

// restoreFromSnapshot finds the latest snapshot and restores state from it
// Sets b.stateManager directly and returns error if any
func (b *BankAccount) restoreFromSnapshot(ctx context.Context) error {
	latestSnapshot, err := b.findLatestSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find latest snapshot: %v", err)
	}

	// Initialize state manager
	if b.stateManager == nil {
		zero := &BankAccountStateV1{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		}
		b.stateManager = locked.New(zero)
	}

	if latestSnapshot != nil {
		// Convert and apply the snapshot to restore state
		domainEvent, err := ConvertFromEventStore(*latestSnapshot)
		if err != nil {
			return fmt.Errorf("failed to convert snapshot event: %v", err)
		}
		
		if err := b.stateManager.Apply(domainEvent); err != nil {
			return fmt.Errorf("failed to apply snapshot event: %v", err)
		}
		
		// Don't override the version - let the Apply method set the data version from the snapshot
		dataVersion := b.stateManager.GetState().Version
		log.Printf("BankAccount %s: Restored state from snapshot at data version %d (event version %d)", b.ID(), dataVersion, latestSnapshot.Version)
	}

	return nil
}

// replayEventsAfterVersion loads and replays events after the current state version
// Applies events directly to b.stateManager
func (b *BankAccount) replayEventsAfterVersion(ctx context.Context) error {
	// Get the starting version from the current state
	startVersion := b.getCurrentVersion()

	// Load events after the snapshot
	events, err := b.eventStore.Load(b.getStreamID(), eventstore.LoadOptions{
		ExclusiveStartVersion: startVersion, // Only load events after snapshot
		Limit:                 0,            // No limit
		Desc:                  false,        // Chronological order
	})

	if err != nil {
		return fmt.Errorf("failed to load events after snapshot: %v", err)
	}

	// If no events exist at all (including snapshot), account doesn't exist
	if startVersion == 0 && len(events) == 0 {
		b.stateManager = nil
		return nil
	}

	// Apply events after snapshot, skipping any additional snapshots
	eventsApplied := 0
	for _, event := range events {
		// Skip snapshot events as they're used for state restoration, not state changes
		if EventTypeV1(event.Type) == EventTypeStateSnapshotV1 {
			continue
		}

		// Convert eventstore.Event to domain event
		domainEvent, err := ConvertFromEventStore(event)
		if err != nil {
			return fmt.Errorf("failed to convert event %s: %v", event.ID, err)
		}

		if err := b.stateManager.Apply(domainEvent); err != nil {
			return fmt.Errorf("failed to apply event %s: %v", event.ID, err)
		}

		eventsApplied++
	}

	currentVersion := int64(0)
	if b.stateManager != nil {
		currentVersion = b.stateManager.GetState().Version
	}

	if startVersion > 0 {
		log.Printf("BankAccount %s: Replayed %d events after snapshot (version %d -> %d)",
			b.ID(), eventsApplied, startVersion, currentVersion)
	} else {
		log.Printf("BankAccount %s: Replayed %d events from beginning (version 0 -> %d)",
			b.ID(), eventsApplied, currentVersion)
	}

	return nil
}
// ConvertFromEventStore converts an eventstore.Event to our domain event
func ConvertFromEventStore(event eventstore.Event) (eventsourced.Event, error) {
	switch EventTypeV1(event.Type) {
	case EventTypeAccountCreatedV1:
		var data AccountCreatedEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse AccountCreatedV1 event: %v", err)
		}
		// For business events, version comes from eventstore event version
		data.Version = event.Version
		return data, nil

	case EventTypeMoneyDepositedV1:
		var data MoneyDepositedEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse MoneyDepositedV1 event: %v", err)
		}
		// For business events, version comes from eventstore event version
		data.Version = event.Version
		return data, nil

	case EventTypeMoneyWithdrawnV1:
		var data MoneyWithdrawnEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse MoneyWithdrawnV1 event: %v", err)
		}
		// For business events, version comes from eventstore event version
		data.Version = event.Version
		return data, nil

	case EventTypeStateSnapshotV1:
		var data StateSnapshotEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse StateSnapshotV1 event: %v", err)
		}
		// For snapshot events, version comes from the snapshot data (state version at snapshot time)
		// data.Version is already set from the JSON, no need to override
		return data, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}
}
