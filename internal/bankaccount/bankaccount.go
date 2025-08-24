package bankaccount

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dapr/go-sdk/actor"
	"github.com/google/uuid"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
	"github.com/shogotsuneto/go-simple-eventstore"
)

// Internal event type constants (not exposed in API)
type AccountEventEventType string

const (
	AccountEventEventTypeAccountCreated AccountEventEventType = "AccountCreated"
	AccountEventEventTypeMoneyDeposited AccountEventEventType = "MoneyDeposited"
	AccountEventEventTypeMoneyWithdrawn AccountEventEventType = "MoneyWithdrawn"
	AccountEventEventTypeStateSnapshot  AccountEventEventType = "StateSnapshot"
)

// BankAccount demonstrates event sourcing pattern with external postgres event store.
// This actor stores events using go-simple-eventstore/postgres for durability and audit trail,
// while maintaining fast access through ephemeral in-memory state cache as long as the actor is activated.
//
// SNAPSHOT OPTIMIZATION:
// The actor now supports periodic snapshots to optimize event replay performance.
// Instead of replaying ALL events from the beginning, it can restore state from the latest snapshot
// and only replay events since that snapshot, significantly improving performance for long-lived accounts.
//
// OPTIMIZATION BENEFITS:
// 1. Fast Access: Operations use cached in-memory state instead of recomputing from events every time
// 2. Actor Pattern: Leverages stateful actor model with in-memory state while actor is active
// 3. External Durability: Events are persisted to postgres for durability and audit trail
// 4. Efficiency: State is computed from events only once (lazy loading) when actor is first accessed
// 5. Consistency: In-memory state is kept in sync with events as operations are performed
// 6. Snapshot Optimization: Periodic snapshots reduce event replay time on actor activation
//
// COMPARISON WITH PREVIOUS IMPLEMENTATION:
// - Before: Used Dapr StateManager for event storage
// - After: Uses external postgres event store for event storage
// - Both: Implement event sourcing patterns but with different persistence layers
type BankAccount struct {
	actor.ServerImplBaseCtx
	
	// Reference to shared external event store (singleton)
	eventStore eventstore.EventStore
	
	// Ephemeral in-memory state for fast access (computed from events)
	state    *BankAccountState
	stateLoaded    bool  // Track if state has been loaded from events
	
	// Snapshot configuration
	snapshotFrequency int64 // Create snapshot every N events (default: 10)
	
	// Mutex to protect concurrent access to state field
	mu sync.RWMutex
}

// getCurrentVersion returns the current stream version from state, or 0 if no state exists
func (b *BankAccount) getCurrentVersion() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	if b.state != nil && b.state.Data != nil {
		return b.state.Data.Version
	}
	return 0
}

// Internal event structures (not exposed in API)
type AccountCreatedEventData struct {
	OwnerName      string    `json:"ownerName"`
	OwnerId        string    `json:"ownerId"`
	InitialDeposit float64   `json:"initialDeposit"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MoneyDepositedEventData struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type MoneyWithdrawnEventData struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// StateSnapshotEventData contains a complete snapshot of the account state
type StateSnapshotEventData struct {
	AccountId string    `json:"accountId"`
	OwnerName string    `json:"ownerName"`
	OwnerId   string    `json:"ownerId"`
	Balance   float64   `json:"balance"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`
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
	state, err := b.computeStateFromEvents(ctx)
	if err != nil {
		return err
	}
	
	if state == nil {
		// Account doesn't exist yet
		b.mu.Lock()
		b.state = nil
		b.mu.Unlock()
	} else {
		// Account exists, cache the computed state for fast access
		b.mu.Lock()
		b.state = state
		b.mu.Unlock()
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
	if b.state == nil {
		if userID != b.ID() {
			return "", fmt.Errorf("insufficient permissions: can only create accounts for yourself")
		}
		return userID, nil
	}
	
	// For existing accounts, check against the stored owner ID
	b.mu.RLock()
	var ownerID string
	if b.state != nil && b.state.Data != nil {
		ownerID = b.state.Data.OwnerId
	}
	b.mu.RUnlock()
	
	if ownerID != userID {
		return "", fmt.Errorf("insufficient permissions: cannot access this account")
	}
	
	return userID, nil
}

// Helper methods for structured responses

func (b *BankAccount) successResponse() *BankAccountState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Return successful response with data nested under Data field
	return b.successResponseWithState(b.state.Data)
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
	if b.state != nil {
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
	
	// Create and store event for durability (include creator info)
	eventData := AccountCreatedEventData{
		OwnerName:      request.OwnerName,
		OwnerId:        userID,
		InitialDeposit: request.InitialDeposit,
		CreatedAt:      time.Now(),
	}
	
	event, err := b.appendEvent(ctx, AccountEventEventTypeAccountCreated, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to create account", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Initialize state and apply the event using centralized logic (protected by lock)
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		},
	}
	
	if err := b.state.Data.applyEvent(*event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	log.Printf("BankAccount %s: Account created by user %s for owner %s", b.ID(), userID, request.OwnerName)
	return b.successResponseWithState(b.state.Data), nil
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
	if b.state == nil {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}
	
	// Create and store event for durability
	eventData := MoneyDepositedEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	event, err := b.appendEvent(ctx, AccountEventEventTypeMoneyDeposited, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to record deposit", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update in-memory state using centralized event application (protected by lock)
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if err := b.state.Data.applyEvent(*event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return b.successResponseWithState(b.state.Data), nil
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
	if b.state == nil {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}
	
	// Check sufficient balance using fast in-memory state (protected by lock)
	b.mu.RLock()
	currentBalance := float64(0)
	if b.state != nil && b.state.Data != nil {
		currentBalance = b.state.Data.Balance
	}
	b.mu.RUnlock()
	if currentBalance < request.Amount {
		return b.errorResponse(ErrorCodeInsufficientFunds, fmt.Sprintf("Insufficient funds: balance %.2f, requested %.2f", currentBalance, request.Amount), map[string]interface{}{
			"currentBalance":   currentBalance,
			"requestedAmount": request.Amount,
		}), nil
	}
	
	// Create and store event for durability
	eventData := MoneyWithdrawnEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	event, err := b.appendEvent(ctx, AccountEventEventTypeMoneyWithdrawn, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to record withdrawal", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update in-memory state using centralized event application (protected by lock)
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if err := b.state.Data.applyEvent(*event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return b.successResponseWithState(b.state.Data), nil
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
	if b.state == nil {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}
	
	// Return fast in-memory cached state
	return b.successResponse(), nil
}



// Event store implementation details

func (b *BankAccount) appendEvent(ctx context.Context, eventType AccountEventEventType, eventData interface{}) (*eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	
	// Convert event data to JSON
	dataBytes, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %v", err)
	}
	
	// Create event for store
	event := eventstore.Event{
		ID:        uuid.New().String(),
		Type:      string(eventType),
		Data:      dataBytes,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"actorType": ActorTypeBankAccount,
			"actorId":   b.ID(),
		},
	}
	
	// Append to event store using stream ID based on actor ID with version check
	streamID := fmt.Sprintf("bankaccount-%s", b.ID())
	events := []eventstore.Event{event}
	_, err = b.eventStore.Append(streamID, events, int(b.getCurrentVersion()))
	if err != nil {
		return nil, err
	}
	
	// The event now has its version set by the event store
	appendedEvent := &events[0]
	
	// Check if we should create a snapshot (only for business events, not snapshots)
	if eventType != AccountEventEventTypeStateSnapshot && 
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

func (b *BankAccount) getAllEvents(ctx context.Context) ([]eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	
	streamID := fmt.Sprintf("bankaccount-%s", b.ID())
	
	// Load all events from event store
	events, err := b.eventStore.Load(streamID, eventstore.LoadOptions{
		ExclusiveStartVersion: 0,
		Limit: 0, // 0 = no limit
		Desc:  false, // chronological order
	})
	
	if err != nil {
		return nil, err
	}
	
	return events, nil
}

// createSnapshot creates a snapshot of the current state and stores it as an event
func (b *BankAccount) createSnapshot(ctx context.Context) error {
	b.mu.RLock()
	if b.state == nil || b.state.Data == nil {
		b.mu.RUnlock()
		return fmt.Errorf("cannot create snapshot: no state available")
	}
	
	// Copy state data while holding the read lock
	snapshotData := StateSnapshotEventData{
		AccountId: b.state.Data.AccountId,
		OwnerName: b.state.Data.OwnerName,
		OwnerId:   b.state.Data.OwnerId,
		Balance:   b.state.Data.Balance,
		IsActive:  b.state.Data.IsActive,
		Version:   b.state.Data.Version,
		Timestamp: time.Now(),
	}
	
	// Parse the CreatedAt timestamp back to time.Time for the snapshot
	createdAt, err := time.Parse(time.RFC3339, b.state.Data.CreatedAt)
	if err != nil {
		// If parsing fails, use current time as fallback
		createdAt = time.Now()
	}
	snapshotData.CreatedAt = createdAt
	b.mu.RUnlock()
	
	_, err = b.appendEvent(ctx, AccountEventEventTypeStateSnapshot, snapshotData)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}
	
	log.Printf("BankAccount %s: Created snapshot at version %d", b.ID(), snapshotData.Version)
	return nil
}

// findLatestSnapshot finds the most recent snapshot event using reverse loading
func (b *BankAccount) findLatestSnapshot(ctx context.Context) (*eventstore.Event, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	
	streamID := fmt.Sprintf("bankaccount-%s", b.ID())
	
	// Load events in reverse order to find the latest snapshot quickly
	events, err := b.eventStore.Load(streamID, eventstore.LoadOptions{
		ExclusiveStartVersion: 0, // Start from latest
		Limit: 100, // Reasonable limit to avoid loading too many events
		Desc:  true, // Reverse order (latest first)
	})
	
	if err != nil {
		return nil, err
	}
	
	// Find the first (latest) snapshot event
	for _, event := range events {
		if AccountEventEventType(event.Type) == AccountEventEventTypeStateSnapshot {
			return &event, nil
		}
	}
	
	return nil, nil // No snapshot found
}

// applyEvent applies a single event to the state in a centralized manner.
// This provides unified event handling and reduces code duplication.
func (state *BankAccountStateData) applyEvent(event eventstore.Event) error {
	switch AccountEventEventType(event.Type) {
	case AccountEventEventTypeAccountCreated:
		var data AccountCreatedEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return fmt.Errorf("failed to parse AccountCreated event: %v", err)
		}
		state.OwnerName = data.OwnerName
		state.OwnerId = data.OwnerId
		state.Balance = data.InitialDeposit
		state.CreatedAt = data.CreatedAt.Format(time.RFC3339)
		
	case AccountEventEventTypeMoneyDeposited:
		var data MoneyDepositedEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return fmt.Errorf("failed to parse MoneyDeposited event: %v", err)
		}
		state.Balance += data.Amount
		
	case AccountEventEventTypeMoneyWithdrawn:
		var data MoneyWithdrawnEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return fmt.Errorf("failed to parse MoneyWithdrawn event: %v", err)
		}
		state.Balance -= data.Amount
		
	case AccountEventEventTypeStateSnapshot:
		// For snapshots, restore the complete state
		var data StateSnapshotEventData
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return fmt.Errorf("failed to parse StateSnapshot event: %v", err)
		}
		state.AccountId = data.AccountId
		state.OwnerName = data.OwnerName
		state.OwnerId = data.OwnerId
		state.Balance = data.Balance
		state.IsActive = data.IsActive
		state.CreatedAt = data.CreatedAt.Format(time.RFC3339)
		// For snapshots, use the version of the snapshot data, not the snapshot event
		state.Version = data.Version
		return nil
		
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
	
	// Update version from the event's version to ensure consistency
	// between state and last applied event
	state.Version = event.Version
	
	return nil
}

func (b *BankAccount) computeStateFromEvents(ctx context.Context) (*BankAccountState, error) {
	if b.eventStore == nil {
		return nil, fmt.Errorf("event store not configured")
	}
	
	streamID := fmt.Sprintf("bankaccount-%s", b.ID())
	
	// Step 1: Try to restore state from snapshot
	if err := b.restoreFromSnapshot(ctx); err != nil {
		return nil, fmt.Errorf("failed to restore from snapshot: %v", err)
	}
	
	// Get the starting version from the restored state
	b.mu.RLock()
	var startVersion int64 = 0
	if b.state != nil {
		startVersion = b.state.Data.Version
	}
	b.mu.RUnlock()
	
	// Step 2: Replay events after the snapshot
	if err := b.replayEventsAfterVersion(ctx, streamID, startVersion); err != nil {
		return nil, fmt.Errorf("failed to replay events: %v", err)
	}
	
	// Return the final state
	b.mu.RLock()
	finalState := b.state
	b.mu.RUnlock()
	
	return finalState, nil
}

// restoreFromSnapshot finds the latest snapshot and restores state from it
// Sets b.state directly and returns error if any
func (b *BankAccount) restoreFromSnapshot(ctx context.Context) error {
	latestSnapshot, err := b.findLatestSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to find latest snapshot: %v", err)
	}
	
	// Initialize empty state
	state := &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		},
	}
	
	if latestSnapshot != nil {
		// Apply the snapshot to restore state (this will set state.Data.Version correctly)
		if err := state.Data.applyEvent(*latestSnapshot); err != nil {
			return fmt.Errorf("failed to apply snapshot event: %v", err)
		}
		log.Printf("BankAccount %s: Restored state from snapshot at version %d", b.ID(), state.Data.Version)
	}
	
	// Set the state with proper locking
	b.mu.Lock()
	b.state = state
	b.mu.Unlock()
	
	return nil
}

// replayEventsAfterVersion loads and replays events after the given version
// Applies events directly to b.state
func (b *BankAccount) replayEventsAfterVersion(ctx context.Context, streamID string, startVersion int64) error {
	// Load events after the snapshot
	events, err := b.eventStore.Load(streamID, eventstore.LoadOptions{
		ExclusiveStartVersion: startVersion, // Only load events after snapshot
		Limit: 0, // No limit
		Desc:  false, // Chronological order
	})
	
	if err != nil {
		return fmt.Errorf("failed to load events after snapshot: %v", err)
	}
	
	// If no events exist at all (including snapshot), account doesn't exist
	if startVersion == 0 && len(events) == 0 {
		b.mu.Lock()
		b.state = nil
		b.mu.Unlock()
		return nil
	}
	
	// Apply events after snapshot, skipping any additional snapshots
	eventsApplied := 0
	for _, event := range events {
		// Skip snapshot events as they're used for state restoration, not state changes
		if AccountEventEventType(event.Type) == AccountEventEventTypeStateSnapshot {
			continue
		}
		
		b.mu.Lock()
		if err := b.state.Data.applyEvent(event); err != nil {
			b.mu.Unlock()
			return fmt.Errorf("failed to apply event %s: %v", event.ID, err)
		}
		b.mu.Unlock()
		eventsApplied++
	}
	
	b.mu.RLock()
	currentVersion := b.state.Data.Version
	b.mu.RUnlock()
	
	if startVersion > 0 {
		log.Printf("BankAccount %s: Replayed %d events after snapshot (version %d -> %d)", 
			b.ID(), eventsApplied, startVersion, currentVersion)
	} else {
		log.Printf("BankAccount %s: Replayed %d events from beginning (version 0 -> %d)", 
			b.ID(), eventsApplied, currentVersion)
	}
	
	return nil
}



