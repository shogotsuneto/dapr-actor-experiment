// Package shared provides shared types for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package shared


// BankAccountState Current state of bank account (computed from events)
type BankAccountState struct {
	// Account owner name
	OwnerName string `json:"ownerName"`
	// Unique account identifier
	AccountId string `json:"accountId"`
	// Current account balance (computed from events)
	Balance float64 `json:"balance"`
	// Account creation timestamp
	CreatedAt string `json:"createdAt,omitempty"`
	// Whether account is active
	IsActive bool `json:"isActive"`
}

// CounterState Current state of the counter actor (state-based)
type CounterState struct {
	// The current counter value
	Value int32 `json:"value"`
}

// TransactionHistory Complete transaction history (event sourcing benefit)
type TransactionHistory struct {
	// Account identifier
	AccountId string `json:"accountId"`
	// List of all events in chronological order
	Events []interface{} `json:"events"`
}

// AccountEvent A single account event
type AccountEvent struct {
	// Event-specific data
	Data map[string]interface{} `json:"data"`
	// Unique event identifier
	EventId string `json:"eventId"`
	// Type of event
	EventType string `json:"eventType"`
	// When the event occurred
	Timestamp string `json:"timestamp"`
}



// ActorId defines model for actorId
type ActorId = string
