// Package shared provides shared types for OpenAPI-based schema validation.
//
// Code generated from OpenAPI specification. DO NOT EDIT manually.
package shared


// AccountEvent A single account event
type AccountEvent struct {
	// Type of event
	EventType string `json:"eventType"`
	// When the event occurred
	Timestamp string `json:"timestamp"`
	// Event-specific data
	Data map[string]interface{} `json:"data"`
	// Unique event identifier
	EventId string `json:"eventId"`
}

