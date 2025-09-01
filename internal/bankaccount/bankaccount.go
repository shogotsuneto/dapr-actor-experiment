package bankaccount

import (
	"fmt"
	"time"

	"github.com/shogotsuneto/go-eventsourced"
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
	OwnerId        string    `json:"ownerId"`
	InitialDeposit float64   `json:"initialDeposit"`
	CreatedAt      time.Time `json:"createdAt"`
	Version        int64     `json:"-"`
}

func (e AccountCreatedEventV1) Type() string { return string(EventTypeAccountCreatedV1) }

// MoneyDepositedEventV1 represents a deposit transaction
type MoneyDepositedEventV1 struct {
	OwnerId     string    `json:"ownerId"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int64     `json:"-"`
}

func (e MoneyDepositedEventV1) Type() string { return string(EventTypeMoneyDepositedV1) }

// MoneyWithdrawnEventV1 represents a withdrawal transaction
type MoneyWithdrawnEventV1 struct {
	OwnerId     string    `json:"ownerId"`
	Amount      float64   `json:"amount"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Version     int64     `json:"-"`
}

func (e MoneyWithdrawnEventV1) Type() string { return string(EventTypeMoneyWithdrawnV1) }

// StateSnapshotEventV1 represents a complete snapshot of account state
type StateSnapshotEventV1 struct {
	AccountId string    `json:"accountId"`
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
		s.OwnerId = e.OwnerId
		s.Balance = e.Balance
		s.IsActive = e.IsActive
		s.CreatedAt = e.CreatedAt.Format(time.RFC3339)
		s.Version = e.Version

	default:
		return fmt.Errorf("unknown event type: %T", event)
	}

	return nil
}

// Clone implements the eventsourced.State interface
func (s *BankAccountStateV1) Clone() *BankAccountStateV1 {
	return &BankAccountStateV1{
		AccountId: s.AccountId,
		OwnerId:   s.OwnerId,
		Balance:   s.Balance,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt,
		Version:   s.Version,
	}
}


