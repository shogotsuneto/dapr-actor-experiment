package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dapr/go-sdk/actor"
	"github.com/google/uuid"
	"github.com/shogotsuneto/go-simple-eventstore"
)

// Wallet demonstrates external event store usage with go-simple-eventstore.
// This actor shows how to:
// 1. Use a third-party event store instead of Dapr's StateManager
// 2. Share a global event store connection across actors (singleton pattern)
// 3. Maintain event sourcing patterns with external persistence
//
// COMPARISON WITH BANK ACCOUNT:
// - BankAccount: Uses Dapr StateManager for event storage
// - Wallet: Uses external go-simple-eventstore for event storage
// - Both: Implement event sourcing patterns but with different persistence layers
type Wallet struct {
	actor.ServerImplBaseCtx
	
	// Reference to shared event store (singleton)
	eventStore eventstore.EventStore
	
	// Ephemeral in-memory state for fast access (cached from events)
	cachedState    *WalletState
	stateLoaded    bool  // Track if state has been loaded from events
	walletExists   bool  // Track if wallet exists to avoid repeated checks
}

// Internal event structures for external event store
type WalletCreatedEventData struct {
	OwnerName      string    `json:"ownerName"`
	Currency       string    `json:"currency"`
	InitialBalance float64   `json:"initialBalance"`
	CreatedAt      time.Time `json:"createdAt"`
}

type FundsAddedEventData struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type FundsSpentEventData struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// Global event store instance (singleton pattern)
var globalEventStore eventstore.EventStore

// SetGlobalEventStore sets the shared event store instance that all Wallet actors will use.
// This demonstrates how actors can share global singletons like database connection pools.
func SetGlobalEventStore(store eventstore.EventStore) {
	globalEventStore = store
	log.Printf("Wallet: Global event store configured")
}

// NewWallet creates a new Wallet actor instance with access to the global event store.
func NewWallet() *Wallet {
	if globalEventStore == nil {
		log.Printf("WARNING: Wallet created without global event store. External persistence disabled.")
	}
	
	return &Wallet{
		eventStore: globalEventStore,
	}
}

func (w *Wallet) Type() string {
	return ActorTypeWallet
}

// ensureStateLoaded loads and caches state from external event store if not already loaded.
func (w *Wallet) ensureStateLoaded(ctx context.Context) error {
	if w.stateLoaded {
		return nil // State already loaded and cached
	}
	
	if w.eventStore == nil {
		return fmt.Errorf("external event store not configured")
	}
	
	// Load state from external event store
	state, err := w.computeStateFromExternalEvents(ctx)
	if err != nil {
		return err
	}
	
	if state == nil {
		// Wallet doesn't exist yet
		w.walletExists = false
		w.cachedState = nil
	} else {
		// Wallet exists, cache the computed state
		w.walletExists = true
		w.cachedState = state
	}
	
	w.stateLoaded = true
	return nil
}

// getCachedState returns the in-memory cached state for fast O(1) access.
func (w *Wallet) getCachedState() (*WalletState, error) {
	if !w.walletExists {
		return nil, fmt.Errorf("wallet does not exist - create wallet first")
	}
	return w.cachedState, nil
}

func (w *Wallet) CreateWallet(ctx context.Context, request CreateWalletRequest) (*WalletState, error) {
	// Ensure state is loaded to check if wallet already exists
	if err := w.ensureStateLoaded(ctx); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to load wallet state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check if wallet already exists
	if w.walletExists {
		return w.errorResponse(ErrorCodeInternalError, "Wallet already exists", map[string]interface{}{
			"walletId": w.ID(),
		}), nil
	}
	
	// Validate request
	if request.OwnerName == "" {
		return w.errorResponse(ErrorCodeValidationError, "Owner name is required", map[string]interface{}{
			"field": "ownerName",
		}), nil
	}
	
	if request.Currency == "" {
		return w.errorResponse(ErrorCodeValidationError, "Currency is required", map[string]interface{}{
			"field": "currency",
		}), nil
	}
	
	// Create event data
	eventData := WalletCreatedEventData{
		OwnerName:      request.OwnerName,
		Currency:       request.Currency,
		InitialBalance: request.InitialBalance,
		CreatedAt:      time.Now(),
	}
	
	// Append event to external event store
	if err := w.appendExternalEvent(ctx, WalletEventEventTypeWalletCreated, eventData); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to create wallet", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update cached state
	w.cachedState = &WalletState{
		Success: true,
		Data: &WalletStateData{
			WalletId:  w.ID(),
			OwnerName: request.OwnerName,
			Currency:  request.Currency,
			Balance:   request.InitialBalance,
			IsActive:  true,
			CreatedAt: eventData.CreatedAt.Format(time.RFC3339),
		},
	}
	w.walletExists = true
	
	log.Printf("Wallet %s: Created for owner %s with currency %s", w.ID(), request.OwnerName, request.Currency)
	return w.cachedState, nil
}

func (w *Wallet) AddFunds(ctx context.Context, request AddFundsRequest) (*WalletState, error) {
	// Ensure state is loaded
	if err := w.ensureStateLoaded(ctx); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to load wallet state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check if wallet exists
	currentState, err := w.getCachedState()
	if err != nil {
		return w.errorResponse(ErrorCodeInternalError, err.Error(), nil), nil
	}
	
	// Validate request
	if request.Amount <= 0 {
		return w.errorResponse(ErrorCodeValidationError, "Amount must be positive", map[string]interface{}{
			"amount": request.Amount,
		}), nil
	}
	
	// Create event data
	eventData := FundsAddedEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	// Append event to external event store
	if err := w.appendExternalEvent(ctx, WalletEventEventTypeFundsAdded, eventData); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to add funds", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update cached state
	currentState.Data.Balance += request.Amount
	
	log.Printf("Wallet %s: Added %.2f, new balance: %.2f", w.ID(), request.Amount, currentState.Data.Balance)
	return currentState, nil
}

func (w *Wallet) SpendFunds(ctx context.Context, request SpendFundsRequest) (*WalletState, error) {
	// Ensure state is loaded
	if err := w.ensureStateLoaded(ctx); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to load wallet state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Check if wallet exists
	currentState, err := w.getCachedState()
	if err != nil {
		return w.errorResponse(ErrorCodeInternalError, err.Error(), nil), nil
	}
	
	// Validate request
	if request.Amount <= 0 {
		return w.errorResponse(ErrorCodeValidationError, "Amount must be positive", map[string]interface{}{
			"amount": request.Amount,
		}), nil
	}
	
	// Check sufficient funds
	if currentState.Data.Balance < request.Amount {
		return w.errorResponse(ErrorCodeInsufficientFunds, "Insufficient funds", map[string]interface{}{
			"currentBalance": currentState.Data.Balance,
			"requestedAmount": request.Amount,
		}), nil
	}
	
	// Create event data
	eventData := FundsSpentEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	// Append event to external event store
	if err := w.appendExternalEvent(ctx, WalletEventEventTypeFundsSpent, eventData); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to spend funds", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Update cached state
	currentState.Data.Balance -= request.Amount
	
	log.Printf("Wallet %s: Spent %.2f, new balance: %.2f", w.ID(), request.Amount, currentState.Data.Balance)
	return currentState, nil
}

func (w *Wallet) GetBalance(ctx context.Context) (*WalletState, error) {
	// Ensure state is loaded
	if err := w.ensureStateLoaded(ctx); err != nil {
		return w.errorResponse(ErrorCodeInternalError, "Failed to load wallet state", map[string]interface{}{
			"error": err.Error(),
		}), nil
	}
	
	// Get cached state
	currentState, err := w.getCachedState()
	if err != nil {
		return w.errorResponse(ErrorCodeInternalError, err.Error(), nil), nil
	}
	
	return currentState, nil
}

func (w *Wallet) GetTransactions(ctx context.Context) (*WalletTransactionHistory, error) {
	// Ensure state is loaded
	if err := w.ensureStateLoaded(ctx); err != nil {
		return &WalletTransactionHistory{
			Success: false,
			Error: &Error{
				Code:    ErrorCodeInternalError,
				Message: "Failed to load wallet state",
				Details: map[string]interface{}{"error": err.Error()},
			},
		}, nil
	}
	
	if !w.walletExists {
		return &WalletTransactionHistory{
			Success: false,
			Error: &Error{
				Code:    ErrorCodeInternalError,
				Message: "Wallet does not exist",
			},
		}, nil
	}
	
	// Load all events from external event store
	events, err := w.getAllExternalEvents(ctx)
	if err != nil {
		return &WalletTransactionHistory{
			Success: false,
			Error: &Error{
				Code:    ErrorCodeInternalError,
				Message: "Failed to load transaction history",
				Details: map[string]interface{}{"error": err.Error()},
			},
		}, nil
	}
	
	// Convert to API format
	apiEvents := make([]WalletEvent, len(events))
	for i, event := range events {
		apiEvents[i] = WalletEvent{
			EventId:   event.ID,
			EventType: WalletEventEventType(event.Type),
			Timestamp: event.Timestamp.Format(time.RFC3339),
			Data:      w.convertEventDataToMap(event.Data),
		}
	}
	
	return &WalletTransactionHistory{
		Success: true,
		Data: &WalletTransactionHistoryData{
			WalletId: w.ID(),
			Events:   apiEvents,
		},
	}, nil
}

// External event store implementation details

func (w *Wallet) appendExternalEvent(ctx context.Context, eventType WalletEventEventType, eventData interface{}) error {
	if w.eventStore == nil {
		return fmt.Errorf("external event store not configured")
	}
	
	// Convert event data to JSON
	dataBytes, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %v", err)
	}
	
	// Create event for external store
	event := eventstore.Event{
		ID:        uuid.New().String(),
		Type:      string(eventType),
		Data:      dataBytes,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"actorType": ActorTypeWallet,
			"actorId":   w.ID(),
		},
	}
	
	// Append to external event store using stream ID based on actor ID
	streamID := fmt.Sprintf("wallet-%s", w.ID())
	return w.eventStore.Append(streamID, []eventstore.Event{event}, -1) // -1 = no version check
}

func (w *Wallet) getAllExternalEvents(ctx context.Context) ([]eventstore.Event, error) {
	if w.eventStore == nil {
		return nil, fmt.Errorf("external event store not configured")
	}
	
	streamID := fmt.Sprintf("wallet-%s", w.ID())
	
	// Load all events from external event store
	events, err := w.eventStore.Load(streamID, eventstore.LoadOptions{
		ExclusiveStartVersion: 0,
		Limit: 0, // 0 = no limit
		Desc:  false, // chronological order
	})
	
	if err != nil {
		return nil, err
	}
	
	return events, nil
}

func (w *Wallet) computeStateFromExternalEvents(ctx context.Context) (*WalletState, error) {
	events, err := w.getAllExternalEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		return nil, nil // Wallet doesn't exist
	}
	
	// Initialize state
	state := &WalletState{
		Success: true,
		Data: &WalletStateData{
			WalletId: w.ID(),
			Balance:  0,
			IsActive: true,
		},
	}
	
	// Replay events to compute current state
	for _, event := range events {
		switch event.Type {
		case string(WalletEventEventTypeWalletCreated):
			var data WalletCreatedEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, fmt.Errorf("failed to parse WalletCreated event: %v", err)
			}
			state.Data.OwnerName = data.OwnerName
			state.Data.Currency = data.Currency
			state.Data.Balance = data.InitialBalance
			state.Data.CreatedAt = data.CreatedAt.Format(time.RFC3339)
			
		case string(WalletEventEventTypeFundsAdded):
			var data FundsAddedEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, fmt.Errorf("failed to parse FundsAdded event: %v", err)
			}
			state.Data.Balance += data.Amount
			
		case string(WalletEventEventTypeFundsSpent):
			var data FundsSpentEventData
			if err := json.Unmarshal(event.Data, &data); err != nil {
				return nil, fmt.Errorf("failed to parse FundsSpent event: %v", err)
			}
			state.Data.Balance -= data.Amount
		}
	}
	
	return state, nil
}

func (w *Wallet) convertEventDataToMap(data []byte) map[string]interface{} {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]interface{}{"error": "failed to parse event data"}
	}
	return result
}

func (w *Wallet) errorResponse(code ErrorCode, message string, details map[string]interface{}) *WalletState {
	return &WalletState{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}