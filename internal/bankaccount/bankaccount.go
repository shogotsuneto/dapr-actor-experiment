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
)

// AccountEvent represents a single account event (temporary definition until generator fix)
type AccountEvent struct {
	EventId   string                 `json:"eventId"`
	EventType string                 `json:"eventType"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// BankAccount demonstrates event sourcing pattern with in-memory state caching.
// This actor stores events for durability and audit trail, while maintaining fast access
// through ephemeral in-memory state cache as long as the actor is activated.
//
// OPTIMIZATION BENEFITS:
// 1. Fast Access: Operations use cached in-memory state instead of recomputing from events every time
// 2. Actor Pattern: Leverages stateful actor model with in-memory state while actor is active
// 3. Durability: Events are still persisted for durability and audit trail
// 4. Efficiency: State is computed from events only once (lazy loading) when actor is first accessed
// 5. Consistency: In-memory state is kept in sync with events as operations are performed
//
// COMPARISON WITH PURE EVENT SOURCING:
// - Before: Every operation called getAllEvents() + computeStateFromEvents() = O(n) events read
// - After: State loaded once, operations use cached state = O(1) access time
type BankAccount struct {
	actor.ServerImplBaseCtx
	
	// Ephemeral in-memory state for fast access (cached from events)
	cachedState    *BankAccountState
	stateLoaded    bool  // Track if state has been loaded from events
	accountExists  bool  // Track if account exists to avoid repeated checks
}

// Event types
const (
	AccountCreatedEvent  = "AccountCreated"
	MoneyDepositedEvent  = "MoneyDeposited"
	MoneyWithdrawnEvent  = "MoneyWithdrawn"
)

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

// StoredEvent represents an event as stored in the state store
type StoredEvent struct {
	EventID   string      `json:"eventId"`
	EventType string      `json:"eventType"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func (b *BankAccount) Type() string {
	return ActorTypeBankAccount
}

// ensureStateLoaded loads and caches state from events if not already loaded.
// This provides fast in-memory access while maintaining event sourcing benefits.
// 
// PERFORMANCE: This method implements lazy loading - state is computed from events
// only once when the actor is first accessed, then cached for subsequent operations.
func (b *BankAccount) ensureStateLoaded(ctx context.Context) error {
	if b.stateLoaded {
		return nil // State already loaded and cached - fast path!
	}
	
	// Load state from events for the first time (expensive operation)
	state, err := b.computeStateFromEvents(ctx)
	if err != nil {
		return err
	}
	
	if state == nil {
		// Account doesn't exist yet
		b.accountExists = false
		b.cachedState = nil
	} else {
		// Account exists, cache the computed state for fast access
		b.accountExists = true
		b.cachedState = state
	}
	
	b.stateLoaded = true
	return nil
}

// getCachedState returns the in-memory cached state for fast O(1) access.
// This leverages the actor pattern's stateful nature for optimal performance.
func (b *BankAccount) getCachedState() (*BankAccountState, error) {
	if !b.accountExists {
		return nil, fmt.Errorf("account does not exist - create account first")
	}
	return b.cachedState, nil
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
	if b.cachedState != nil && b.cachedState.OwnerId != userID {
		return "", fmt.Errorf("insufficient permissions: cannot access this account")
	}
	
	return userID, nil
}

// Helper methods for structured responses

func (b *BankAccount) successResponse() *BankAccountState {
	// Return successful response without Error field (omitempty will exclude it)
	response := &BankAccountState{
		Success:   true,
		AccountId: b.cachedState.AccountId,
		OwnerName: b.cachedState.OwnerName,
		OwnerId:   b.cachedState.OwnerId,
		Balance:   b.cachedState.Balance,
		IsActive:  b.cachedState.IsActive,
		CreatedAt: b.cachedState.CreatedAt,
	}
	// Don't set Error field - omitempty will exclude it from JSON
	return response
}

func (b *BankAccount) errorResponse(code, message string, details map[string]interface{}) *BankAccountState {
	return &BankAccountState{
		Success: false,
		Error: Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

func (b *BankAccount) errorResponseHistory(code, message string, details map[string]interface{}) *TransactionHistory {
	return &TransactionHistory{
		Success: false,
		Error: Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

func (b *BankAccount) CreateAccount(ctx context.Context, request CreateAccountRequest) (*BankAccountState, error) {
	// Check ownership and get user ID
	userID, err := b.checkOwnership(ctx)
	if err != nil {
		log.Printf("BankAccount %s: User access denied - %v", b.ID(), err)
		return b.errorResponse("AUTHORIZATION_ERROR", err.Error(), nil), nil
	}
	
	// Ensure state is loaded
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check if account already exists (fast in-memory check)
	if b.accountExists {
		return b.errorResponse("ACCOUNT_ALREADY_EXISTS", "Account already exists", map[string]interface{}{
			"accountId": b.ID(),
		}), nil
	}
	
	// Validate request
	if request.OwnerName == "" {
		return b.errorResponse("VALIDATION_ERROR", "Owner name is required", nil), nil
	}
	if request.InitialDeposit < 0 {
		return b.errorResponse("VALIDATION_ERROR", "Initial deposit cannot be negative", map[string]interface{}{
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
	
	if err := b.appendEvent(ctx, AccountCreatedEvent, eventData); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to create account", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update in-memory cached state for fast access
	b.cachedState = &BankAccountState{
		Success:   true,
		AccountId: b.ID(),
		OwnerName: request.OwnerName,
		OwnerId:   userID,
		Balance:   request.InitialDeposit,
		IsActive:  true,
		CreatedAt: eventData.CreatedAt.Format(time.RFC3339),
	}
	b.accountExists = true
	
	log.Printf("BankAccount %s: Account created by user %s for owner %s", b.ID(), userID, request.OwnerName)
	return b.cachedState, nil
}

func (b *BankAccount) Deposit(ctx context.Context, request DepositRequest) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse("AUTHORIZATION_ERROR", err.Error(), nil), nil
	}
	
	// Validate request
	if request.Amount <= 0 {
		return b.errorResponse("VALIDATION_ERROR", "Deposit amount must be positive", map[string]interface{}{
			"providedAmount": request.Amount,
		}), nil
	}
	
	// Ensure account exists
	if !b.accountExists {
		return b.errorResponse("ACCOUNT_NOT_FOUND", "Account does not exist - create account first", nil), nil
	}
	
	// Create and store event for durability
	eventData := MoneyDepositedEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	if err := b.appendEvent(ctx, MoneyDepositedEvent, eventData); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to record deposit", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update in-memory cached state for fast access
	b.cachedState.Balance += request.Amount
	
	return b.successResponse(), nil
}

func (b *BankAccount) Withdraw(ctx context.Context, request WithdrawRequest) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse("AUTHORIZATION_ERROR", err.Error(), nil), nil
	}
	
	// Validate request
	if request.Amount <= 0 {
		return b.errorResponse("VALIDATION_ERROR", "Withdrawal amount must be positive", map[string]interface{}{
			"providedAmount": request.Amount,
		}), nil
	}
	
	// Ensure account exists
	if !b.accountExists {
		return b.errorResponse("ACCOUNT_NOT_FOUND", "Account does not exist - create account first", nil), nil
	}
	
	// Check sufficient balance using fast in-memory state
	if b.cachedState.Balance < request.Amount {
		return b.errorResponse("INSUFFICIENT_FUNDS", fmt.Sprintf("Insufficient funds: balance %.2f, requested %.2f", b.cachedState.Balance, request.Amount), map[string]interface{}{
			"currentBalance":   b.cachedState.Balance,
			"requestedAmount": request.Amount,
		}), nil
	}
	
	// Create and store event for durability
	eventData := MoneyWithdrawnEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	if err := b.appendEvent(ctx, MoneyWithdrawnEvent, eventData); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to record withdrawal", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update in-memory cached state for fast access
	b.cachedState.Balance -= request.Amount
	
	return b.successResponse(), nil
}

func (b *BankAccount) GetBalance(ctx context.Context) (*BankAccountState, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponse("INTERNAL_ERROR", "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponse("AUTHORIZATION_ERROR", err.Error(), nil), nil
	}
	
	// Check if account exists
	if !b.accountExists {
		return b.errorResponse("ACCOUNT_NOT_FOUND", "Account does not exist - create account first", nil), nil
	}
	
	// Return fast in-memory cached state
	return b.successResponse(), nil
}

func (b *BankAccount) GetHistory(ctx context.Context) (*TransactionHistory, error) {
	// Ensure state is loaded first so checkOwnership can validate
	if err := b.ensureStateLoaded(ctx); err != nil {
		return b.errorResponseHistory("INTERNAL_ERROR", "Failed to load account state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check ownership
	_, err := b.checkOwnership(ctx)
	if err != nil {
		return b.errorResponseHistory("AUTHORIZATION_ERROR", err.Error(), nil), nil
	}
	
	if !b.accountExists {
		return b.errorResponseHistory("ACCOUNT_NOT_FOUND", "Account does not exist - create account first", nil), nil
	}
	
	// Get events for history (still need to read from storage for complete audit trail)
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return b.errorResponseHistory("INTERNAL_ERROR", "Failed to retrieve transaction history", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Convert internal events to API events
	var apiEvents []interface{}
	for _, event := range events {
		apiEvent := AccountEvent{
			EventId:   event.EventID,
			EventType: event.EventType,
			Timestamp: event.Timestamp.Format(time.RFC3339),
			Data:      b.convertEventDataToMap(event.Data),
		}
		apiEvents = append(apiEvents, apiEvent)
	}
	
	return &TransactionHistory{
		Success:   true,
		AccountId: b.ID(),
		Events:    apiEvents,
	}, nil
}

// Event sourcing implementation details

func (b *BankAccount) appendEvent(ctx context.Context, eventType string, eventData interface{}) error {
	event := StoredEvent{
		EventID:   uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now(),
		Data:      eventData,
	}
	
	// Load existing events
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return err
	}
	
	// Append new event
	events = append(events, event)
	
	// Store back to state manager
	eventsKey := "events"
	return b.GetStateManager().Set(ctx, eventsKey, events)
}

func (b *BankAccount) getAllEvents(ctx context.Context) ([]StoredEvent, error) {
	eventsKey := "events"
	var events []StoredEvent
	
	ok, err := b.GetStateManager().Contains(ctx, eventsKey)
	if err != nil {
		return nil, err
	}
	
	if !ok {
		return []StoredEvent{}, nil
	}
	
	err = b.GetStateManager().Get(ctx, eventsKey, &events)
	if err != nil {
		return nil, err
	}
	
	return events, nil
}

func (b *BankAccount) computeStateFromEvents(ctx context.Context) (*BankAccountState, error) {
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		return nil, nil // Account doesn't exist
	}
	
	// Initialize state
	state := &BankAccountState{
		AccountId: b.ID(),
		Balance:   0,
		IsActive:  true,
	}
	
	// Replay events to compute current state
	for _, event := range events {
		switch event.EventType {
		case AccountCreatedEvent:
			data, err := b.parseEventData(event.Data, &AccountCreatedEventData{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse AccountCreated event: %v", err)
			}
			createdData := data.(*AccountCreatedEventData)
			state.OwnerName = createdData.OwnerName
			state.OwnerId = createdData.OwnerId
			state.Balance = createdData.InitialDeposit
			state.CreatedAt = createdData.CreatedAt.Format(time.RFC3339)
			
		case MoneyDepositedEvent:
			data, err := b.parseEventData(event.Data, &MoneyDepositedEventData{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse MoneyDeposited event: %v", err)
			}
			depositData := data.(*MoneyDepositedEventData)
			state.Balance += depositData.Amount
			
		case MoneyWithdrawnEvent:
			data, err := b.parseEventData(event.Data, &MoneyWithdrawnEventData{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse MoneyWithdrawn event: %v", err)
			}
			withdrawData := data.(*MoneyWithdrawnEventData)
			state.Balance -= withdrawData.Amount
		}
	}
	
	return state, nil
}

func (b *BankAccount) parseEventData(data interface{}, target interface{}) (interface{}, error) {
	// Convert to JSON and back to parse properly
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	err = json.Unmarshal(jsonData, target)
	if err != nil {
		return nil, err
	}
	
	return target, nil
}

func (b *BankAccount) convertEventDataToMap(data interface{}) map[string]interface{} {
	// Convert to JSON and back to get a map
	jsonData, err := json.Marshal(data)
	if err != nil {
		return map[string]interface{}{"error": "failed to convert event data"}
	}
	
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return map[string]interface{}{"error": "failed to parse event data"}
	}
	
	return result
}

