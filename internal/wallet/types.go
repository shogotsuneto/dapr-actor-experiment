// Package wallet provides primitives for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package wallet


// AddFundsRequest Request to add funds to wallet
type AddFundsRequest struct {
	// Amount to add
	Amount float64 `json:"amount"`
	// Description of the fund addition
	Description string `json:"description"`
}

// CreateWalletRequest Request to create a new wallet using external event store
type CreateWalletRequest struct {
	// Currency code for the wallet
	Currency string `json:"currency"`
	// Initial balance for the wallet
	InitialBalance float64 `json:"initialBalance,omitempty"`
	// Name of the wallet owner
	OwnerName string `json:"ownerName"`
}

// Error Error information returned within 200 responses
type Error struct {
	// Error code identifying the type of error
	Code ErrorCode `json:"code"`
	// Additional error-specific details
	Details map[string]interface{} `json:"details,omitempty"`
	// Human-readable error message
	Message string `json:"message"`
}

// SpendFundsRequest Request to spend funds from wallet
type SpendFundsRequest struct {
	// Amount to spend
	Amount float64 `json:"amount"`
	// Description of the spending
	Description string `json:"description"`
}

// WalletEvent A single wallet event from external event store
type WalletEvent struct {
	// Event-specific data
	Data map[string]interface{} `json:"data"`
	// Unique event identifier
	EventId string `json:"eventId"`
	// Type of wallet event
	EventType WalletEventEventType `json:"eventType"`
	// When the event occurred
	Timestamp string `json:"timestamp"`
}

// WalletState Current state of the wallet (from external event store)
type WalletState struct {
	// Wallet state data (only present if success=true)
	Data *WalletStateData `json:"data,omitempty"`
	// Error information returned within 200 responses
	Error *Error `json:"error,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
}

// WalletStateData Wallet state data (only present if success=true)
type WalletStateData struct {
	// Current balance
	Balance float64 `json:"balance"`
	// When the wallet was created
	CreatedAt string `json:"createdAt"`
	// Currency code
	Currency string `json:"currency"`
	// Whether the wallet is active
	IsActive bool `json:"isActive"`
	// Name of the wallet owner
	OwnerName string `json:"ownerName"`
	// Wallet identifier
	WalletId string `json:"walletId"`
}

// WalletTransactionHistory Complete transaction history from external event store
type WalletTransactionHistory struct {
	// Transaction history data from external event store
	Data *WalletTransactionHistoryData `json:"data,omitempty"`
	// Error information returned within 200 responses
	Error *Error `json:"error,omitempty"`
	// Whether the operation was successful
	Success bool `json:"success"`
}

// WalletTransactionHistoryData Transaction history data from external event store
type WalletTransactionHistoryData struct {
	// List of all wallet events in chronological order
	Events []WalletEvent `json:"events"`
	// Wallet identifier
	WalletId string `json:"walletId"`
}





// ErrorCode defines valid values for Error.code
type ErrorCode string

// ErrorCode constants
const (
	ErrorCodeValidationError ErrorCode = "VALIDATION_ERROR"
	ErrorCodeAuthenticationError ErrorCode = "AUTHENTICATION_ERROR"
	ErrorCodeAuthorizationError ErrorCode = "AUTHORIZATION_ERROR"
	ErrorCodeInsufficientFunds ErrorCode = "INSUFFICIENT_FUNDS"
	ErrorCodeAccountNotFound ErrorCode = "ACCOUNT_NOT_FOUND"
	ErrorCodeAccountAlreadyExists ErrorCode = "ACCOUNT_ALREADY_EXISTS"
	ErrorCodeValueOutOfRange ErrorCode = "VALUE_OUT_OF_RANGE"
	ErrorCodeInternalError ErrorCode = "INTERNAL_ERROR"
)

// WalletEventEventType defines valid values for WalletEvent.eventType
type WalletEventEventType string

// WalletEventEventType constants
const (
	WalletEventEventTypeWalletCreated WalletEventEventType = "WalletCreated"
	WalletEventEventTypeFundsAdded WalletEventEventType = "FundsAdded"
	WalletEventEventTypeFundsSpent WalletEventEventType = "FundsSpent"
)
