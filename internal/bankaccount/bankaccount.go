package bankaccount

import (
	"fmt"
	"sync"
	"time"

	"github.com/shogotsuneto/go-eventsourced"
	"github.com/shogotsuneto/go-eventsourced/locked"
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
	Version        int64     `json:"-"`
}

func (e AccountCreatedEventV1) Type() string { return string(EventTypeAccountCreatedV1) }

// MoneyDepositedEventV1 represents a deposit transaction
type MoneyDepositedEventV1 struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int64     `json:"-"`
}

func (e MoneyDepositedEventV1) Type() string { return string(EventTypeMoneyDepositedV1) }

// MoneyWithdrawnEventV1 represents a withdrawal transaction
type MoneyWithdrawnEventV1 struct {
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int64     `json:"-"`
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
		s.Version = e.Version

	case MoneyDepositedEventV1:
		s.Balance += e.Amount
		s.Version = e.Version

	case MoneyWithdrawnEventV1:
		s.Balance -= e.Amount
		s.Version = e.Version

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
func (s *BankAccountStateV1) Clone() *BankAccountStateV1 {
	return &BankAccountStateV1{
		AccountId: s.AccountId,
		OwnerName: s.OwnerName,
		OwnerId:   s.OwnerId,
		Balance:   s.Balance,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt,
		Version:   s.Version,
	}
}

// LockedStateManager wraps the locked.LockedES with version management for event sourcing
type LockedStateManager struct {
	es *locked.LockedES[*BankAccountStateV1]
	mu sync.RWMutex // For version-specific operations only
}

// NewLockedStateManager creates a new locked state manager
func NewLockedStateManager(accountId string) *LockedStateManager {
	zero := &BankAccountStateV1{
		AccountId: accountId,
		Balance:   0,
		IsActive:  true,
	}
	return &LockedStateManager{
		es: locked.New(zero),
	}
}

// Apply implements the eventsourced.EventApplier interface
func (lsm *LockedStateManager) Apply(event eventsourced.Event) error {
	return lsm.es.Apply(event)
}

// GetState implements the eventsourced.StateGetter interface
func (lsm *LockedStateManager) GetState() *BankAccountStateV1 {
	return lsm.es.GetState()
}

// SetVersion sets the version on the internal state
// This requires additional locking since LockedES doesn't expose direct state mutation
func (lsm *LockedStateManager) SetVersion(version int64) {
	lsm.mu.Lock()
	defer lsm.mu.Unlock()
	
	// Get current state, modify version, and reapply
	current := lsm.es.GetState()
	current.Version = version
	
	// Replace the internal state by creating a new LockedES with the updated state
	lsm.es = locked.New(current)
}