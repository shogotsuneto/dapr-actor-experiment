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
	"github.com/shogotsuneto/go-simple-eventstore"
)

// Internal event type constants (not exposed in API)
type AccountEventEventType string

const (
	AccountEventEventTypeAccountCreated AccountEventEventType = "AccountCreated"
	AccountEventEventTypeMoneyDeposited AccountEventEventType = "MoneyDeposited"
	AccountEventEventTypeMoneyWithdrawn AccountEventEventType = "MoneyWithdrawn"
)

// BankAccount demonstrates event sourcing pattern with external postgres event store.
// This actor stores events using go-simple-eventstore/postgres for durability and audit trail,
// while maintaining fast access through ephemeral in-memory state cache as long as the actor is activated.
//
// OPTIMIZATION BENEFITS:
// 1. Fast Access: Operations use cached in-memory state instead of recomputing from events every time
// 2. Actor Pattern: Leverages stateful actor model with in-memory state while actor is active
// 3. External Durability: Events are persisted to postgres for durability and audit trail
// 4. Efficiency: State is computed from events only once (lazy loading) when actor is first accessed
// 5. Consistency: In-memory state is kept in sync with events as operations are performed
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
	accountExists  bool  // Track if account exists to avoid repeated checks
	streamVersion  int64 // Track current stream version for optimistic concurrency
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

// NewBankAccount creates a new BankAccount actor instance with access to the provided event store.
func NewBankAccount(eventStore eventstore.EventStore) *BankAccount {
	if eventStore == nil {
		log.Printf("WARNING: BankAccount created without event store. External persistence disabled.")
	}
	
	return &BankAccount{
		eventStore: eventStore,
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
		b.accountExists = false
		b.state = nil
	} else {
		// Account exists, cache the computed state for fast access
		b.accountExists = true
		b.state = state
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
	if !b.accountExists {
		if userID != b.ID() {
			return "", fmt.Errorf("insufficient permissions: can only create accounts for yourself")
		}
		return userID, nil
	}
	
	// For existing accounts, check against the stored owner ID
	if b.state != nil && b.state.Data.OwnerId != userID {
		return "", fmt.Errorf("insufficient permissions: cannot access this account")
	}
	
	return userID, nil
}

// Helper methods for structured responses

func (b *BankAccount) successResponse() *BankAccountState {
	// Return successful response with data nested under Data field
	return &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: b.state.Data.AccountId,
			OwnerName: b.state.Data.OwnerName,
			OwnerId:   b.state.Data.OwnerId,
			Balance:   b.state.Data.Balance,
			IsActive:  b.state.Data.IsActive,
			CreatedAt: b.state.Data.CreatedAt,
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
	if b.accountExists {
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
	
	// Initialize stream version for new account
	b.streamVersion = 0
	
	event, err := b.appendEvent(ctx, AccountEventEventTypeAccountCreated, eventData)
	if err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to create account", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Initialize state and apply the event using centralized logic
	b.state = &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		},
	}
	
	if err := b.applyEventToState(b.state.Data, *event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	b.accountExists = true
	
	log.Printf("BankAccount %s: Account created by user %s for owner %s", b.ID(), userID, request.OwnerName)
	return b.state, nil
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
	if !b.accountExists {
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
	
	// Update in-memory state using centralized event application
	if err := b.applyEventToState(b.state.Data, *event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return b.successResponse(), nil
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
	
	// Ensure account exists
	if !b.accountExists {
		return b.errorResponse(ErrorCodeAccountNotFound, "Account does not exist - create account first", nil), nil
	}
	
	// Check sufficient balance using fast in-memory state
	if b.state.Data.Balance < request.Amount {
		return b.errorResponse(ErrorCodeInsufficientFunds, fmt.Sprintf("Insufficient funds: balance %.2f, requested %.2f", b.state.Data.Balance, request.Amount), map[string]interface{}{
			"currentBalance":   b.state.Data.Balance,
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
	
	// Update in-memory state using centralized event application
	if err := b.applyEventToState(b.state.Data, *event); err != nil {
		return b.errorResponse(ErrorCodeInternalError, "Failed to apply state change", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	return b.successResponse(), nil
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
	if !b.accountExists {
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
	latestVersion, err := b.eventStore.Append(streamID, []eventstore.Event{event}, int(b.streamVersion))
	if err != nil {
		return nil, err
	}
	
	// Update stream version with the version returned by append
	b.streamVersion = latestVersion
	
	// The event now has its version set by the event store
	return &event, nil
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

// applyEventToState applies a single event to the state in a centralized manner.
// This provides unified event handling and reduces code duplication.
func (b *BankAccount) applyEventToState(state *BankAccountStateData, event eventstore.Event) error {
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
		
	default:
		return fmt.Errorf("unknown event type: %s", event.Type)
	}
	
	return nil
}

func (b *BankAccount) computeStateFromEvents(ctx context.Context) (*BankAccountState, error) {
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		b.streamVersion = 0 // No events yet
		return nil, nil // Account doesn't exist
	}
	
	// Initialize state with nested data structure
	state := &BankAccountState{
		Success: true,
		Data: &BankAccountStateData{
			AccountId: b.ID(),
			Balance:   0,
			IsActive:  true,
		},
	}
	
	// Apply all events in sequence using centralized event application logic
	for _, event := range events {
		if err := b.applyEventToState(state.Data, event); err != nil {
			return nil, fmt.Errorf("failed to apply event %s: %v", event.ID, err)
		}
	}
	
	// Update stream version from the last event's version field
	if len(events) > 0 {
		b.streamVersion = events[len(events)-1].Version
	} else {
		b.streamVersion = 0
	}
	
	return state, nil
}



