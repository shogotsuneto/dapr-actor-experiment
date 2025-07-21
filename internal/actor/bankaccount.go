package actor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dapr/go-sdk/actor"
	"github.com/google/uuid"
	generated "github.com/shogotsuneto/dapr-actor-experiment/internal/generated/openapi"
)

// BankAccountActor demonstrates event sourcing pattern in contrast to CounterActor's state-based approach.
// This actor stores events and reconstructs state from the event history, providing full audit trail.
type BankAccountActor struct {
	actor.ServerImplBaseCtx
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

func (b *BankAccountActor) Type() string {
	return generated.ActorTypeBankAccountActor
}

func (b *BankAccountActor) CreateAccount(ctx context.Context, request generated.CreateAccountRequest) (*generated.BankAccountState, error) {
	// Check if account already exists
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) > 0 {
		return nil, errors.New("account already exists")
	}
	
	// Validate request
	if request.OwnerName == "" {
		return nil, errors.New("owner name is required")
	}
	if request.InitialDeposit < 0 {
		return nil, errors.New("initial deposit cannot be negative")
	}
	
	// Create and store event
	eventData := AccountCreatedEventData{
		OwnerName:      request.OwnerName,
		InitialDeposit: request.InitialDeposit,
		CreatedAt:      time.Now(),
	}
	
	if err := b.appendEvent(ctx, AccountCreatedEvent, eventData); err != nil {
		return nil, err
	}
	
	// Return current state computed from events
	return b.computeStateFromEvents(ctx)
}

func (b *BankAccountActor) Deposit(ctx context.Context, request generated.DepositRequest) (*generated.BankAccountState, error) {
	// Validate request
	if request.Amount <= 0 {
		return nil, errors.New("deposit amount must be positive")
	}
	
	// Ensure account exists
	currentState, err := b.computeStateFromEvents(ctx)
	if err != nil {
		return nil, err
	}
	if currentState == nil {
		return nil, errors.New("account does not exist - create account first")
	}
	
	// Create and store event
	eventData := MoneyDepositedEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	if err := b.appendEvent(ctx, MoneyDepositedEvent, eventData); err != nil {
		return nil, err
	}
	
	// Return updated state
	return b.computeStateFromEvents(ctx)
}

func (b *BankAccountActor) Withdraw(ctx context.Context, request generated.WithdrawRequest) (*generated.BankAccountState, error) {
	// Validate request
	if request.Amount <= 0 {
		return nil, errors.New("withdrawal amount must be positive")
	}
	
	// Ensure account exists and has sufficient balance
	currentState, err := b.computeStateFromEvents(ctx)
	if err != nil {
		return nil, err
	}
	if currentState == nil {
		return nil, errors.New("account does not exist - create account first")
	}
	if currentState.Balance < request.Amount {
		return nil, fmt.Errorf("insufficient funds: balance %.2f, requested %.2f", currentState.Balance, request.Amount)
	}
	
	// Create and store event
	eventData := MoneyWithdrawnEventData{
		Amount:      request.Amount,
		Description: request.Description,
		Timestamp:   time.Now(),
	}
	
	if err := b.appendEvent(ctx, MoneyWithdrawnEvent, eventData); err != nil {
		return nil, err
	}
	
	// Return updated state
	return b.computeStateFromEvents(ctx)
}

func (b *BankAccountActor) GetBalance(ctx context.Context) (*generated.BankAccountState, error) {
	state, err := b.computeStateFromEvents(ctx)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, errors.New("account does not exist - create account first")
	}
	return state, nil
}

func (b *BankAccountActor) GetHistory(ctx context.Context) (*generated.TransactionHistory, error) {
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		return nil, errors.New("account does not exist - create account first")
	}
	
	// Convert internal events to API events
	var apiEvents []interface{}
	for _, event := range events {
		apiEvent := generated.AccountEvent{
			EventId:   event.EventID,
			EventType: event.EventType,
			Timestamp: event.Timestamp.Format(time.RFC3339),
			Data:      b.convertEventDataToMap(event.Data),
		}
		apiEvents = append(apiEvents, apiEvent)
	}
	
	return &generated.TransactionHistory{
		AccountId: "placeholder-id", // TODO: Get actual actor ID
		Events:    apiEvents,
	}, nil
}

// Event sourcing implementation details

func (b *BankAccountActor) appendEvent(ctx context.Context, eventType string, eventData interface{}) error {
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

func (b *BankAccountActor) getAllEvents(ctx context.Context) ([]StoredEvent, error) {
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

func (b *BankAccountActor) computeStateFromEvents(ctx context.Context) (*generated.BankAccountState, error) {
	events, err := b.getAllEvents(ctx)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		return nil, nil // Account doesn't exist
	}
	
	// Initialize state
	state := &generated.BankAccountState{
		AccountId: "placeholder-id", // TODO: Get actual actor ID
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

func (b *BankAccountActor) parseEventData(data interface{}, target interface{}) (interface{}, error) {
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

func (b *BankAccountActor) convertEventDataToMap(data interface{}) map[string]interface{} {
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