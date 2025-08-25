package bankaccount

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/shogotsuneto/go-eventsourced"
	eventstore "github.com/shogotsuneto/go-simple-eventstore"
)

// Event type constants with V1 versioning for future evolution
type EventTypeV1 string

const (
	EventTypeAccountCreatedV1 EventTypeV1 = "AccountCreatedV1"
	EventTypeMoneyDepositedV1 EventTypeV1 = "MoneyDepositedV1"
	EventTypeMoneyWithdrawnV1 EventTypeV1 = "MoneyWithdrawnV1"
	EventTypeStateSnapshotV1  EventTypeV1 = "StateSnapshotV1"
)

// Event implementations with V1 versioning

// AccountCreatedEventV1 represents the creation of a bank account
type AccountCreatedEventV1 struct {
	OwnerName      string    `json:"ownerName"`
	OwnerId        string    `json:"ownerId"`
	InitialDeposit float64   `json:"initialDeposit"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (e AccountCreatedEventV1) Type() string { return string(EventTypeAccountCreatedV1) }

// MoneyDepositedEventV1 represents a deposit transaction
type MoneyDepositedEventV1 struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e MoneyDepositedEventV1) Type() string { return string(EventTypeMoneyDepositedV1) }

// MoneyWithdrawnEventV1 represents a withdrawal transaction
type MoneyWithdrawnEventV1 struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e MoneyWithdrawnEventV1) Type() string { return string(EventTypeMoneyWithdrawnV1) }

// StateSnapshotEventV1 represents a complete snapshot of account state
type StateSnapshotEventV1 struct {
	AccountId string    `json:"accountId"`
	OwnerName string    `json:"ownerName"`
	OwnerId   string    `json:"ownerId"`
	Balance   float64   `json:"balance"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	Version   int64     `json:"version"`
	Timestamp time.Time `json:"timestamp"`
}

func (e StateSnapshotEventV1) Type() string { return string(EventTypeStateSnapshotV1) }

// BankAccountStateV1 implements the State interface from go-eventsourced
type BankAccountStateV1 struct {
	AccountId string  `json:"accountId"`
	OwnerName string  `json:"ownerName"`
	OwnerId   string  `json:"ownerId"`
	Balance   float64 `json:"balance"`
	IsActive  bool    `json:"isActive"`
	CreatedAt string  `json:"createdAt"` // RFC3339 formatted string
	Version   int64   `json:"version"`
}

// Apply implements the eventsourced.State interface
func (s *BankAccountStateV1) Apply(event eventsourced.Event) error {
	switch e := event.(type) {
	case AccountCreatedEventV1:
		s.OwnerName = e.OwnerName
		s.OwnerId = e.OwnerId
		s.Balance = e.InitialDeposit
		s.CreatedAt = e.CreatedAt.Format(time.RFC3339)
		s.IsActive = true

	case MoneyDepositedEventV1:
		s.Balance += e.Amount

	case MoneyWithdrawnEventV1:
		s.Balance -= e.Amount

	case StateSnapshotEventV1:
		// For snapshots, restore the complete state
		s.AccountId = e.AccountId
		s.OwnerName = e.OwnerName
		s.OwnerId = e.OwnerId
		s.Balance = e.Balance
		s.IsActive = e.IsActive
		s.CreatedAt = e.CreatedAt.Format(time.RFC3339)
		s.Version = e.Version
		return nil

	default:
		return fmt.Errorf("unknown event type: %T", event)
	}

	return nil
}

// Clone implements the eventsourced.State interface
func (s *BankAccountStateV1) Clone() BankAccountStateV1 {
	return BankAccountStateV1{
		AccountId: s.AccountId,
		OwnerName: s.OwnerName,
		OwnerId:   s.OwnerId,
		Balance:   s.Balance,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt,
		Version:   s.Version,
	}
}

// StateManager manages the event sourced state for bank accounts
type StateManager struct {
	state BankAccountStateV1
}

// NewStateManager creates a new state manager
func NewStateManager(accountId string) *StateManager {
	return &StateManager{
		state: BankAccountStateV1{
			AccountId: accountId,
			Balance:   0,
			IsActive:  true,
		},
	}
}

// Apply implements the eventsourced.EventApplier interface
func (sm *StateManager) Apply(event eventsourced.Event) error {
	return sm.state.Apply(event)
}

// GetState implements the eventsourced.StateGetter interface
func (sm *StateManager) GetState() BankAccountStateV1 {
	return sm.state.Clone()
}

// SetVersion sets the version on the internal state
func (sm *StateManager) SetVersion(version int64) {
	sm.state.Version = version
}

// ConvertFromEventStore converts an eventstore.Event to our domain event
func ConvertFromEventStore(event eventstore.Event) (eventsourced.Event, error) {
	switch EventTypeV1(event.Type) {
	case EventTypeAccountCreatedV1:
		var data AccountCreatedEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse AccountCreatedV1 event: %v", err)
		}
		return data, nil

	case EventTypeMoneyDepositedV1:
		var data MoneyDepositedEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse MoneyDepositedV1 event: %v", err)
		}
		return data, nil

	case EventTypeMoneyWithdrawnV1:
		var data MoneyWithdrawnEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse MoneyWithdrawnV1 event: %v", err)
		}
		return data, nil

	case EventTypeStateSnapshotV1:
		var data StateSnapshotEventV1
		if err := json.Unmarshal(event.Data, &data); err != nil {
			return nil, fmt.Errorf("failed to parse StateSnapshotV1 event: %v", err)
		}
		return data, nil

	default:
		return nil, fmt.Errorf("unknown event type: %s", event.Type)
	}
}