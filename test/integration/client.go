package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// GetDaprEndpoint returns the Dapr HTTP endpoint URL, configurable via environment variable
func GetDaprEndpoint() string {
	if endpoint := os.Getenv("DAPR_HTTP_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "http://localhost:3500"
}

// GetActorServiceEndpoint returns the actor service endpoint URL, configurable via environment variable
func GetActorServiceEndpoint() string {
	if endpoint := os.Getenv("ACTOR_SERVICE_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "http://localhost:8080"
}

// DaprClient provides utilities for making HTTP calls to Dapr actor endpoints
type DaprClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewDaprClient creates a new Dapr client
func NewDaprClient(baseURL string) *DaprClient {
	return &DaprClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// ActorMethodRequest represents a request to invoke an actor method
type ActorMethodRequest struct {
	ActorType string
	ActorID   string
	Method    string
	Data      interface{}
}

// ActorMethodResponse represents the response from an actor method call
type ActorMethodResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// InvokeActorMethod invokes an actor method via Dapr HTTP API
func (c *DaprClient) InvokeActorMethod(ctx context.Context, req ActorMethodRequest) (*ActorMethodResponse, error) {
	url := fmt.Sprintf("%s/v1.0/actors/%s/%s/method/%s", c.baseURL, req.ActorType, req.ActorID, req.Method)

	var httpReq *http.Request
	var err error

	if req.Data != nil {
		// POST request with data
		jsonData, err := json.Marshal(req.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}

		httpReq, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
	} else {
		// GET request
		httpReq, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP request: %w", err)
		}
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &ActorMethodResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    resp.Header,
	}, nil
}

// InvokeActorMethodWithResponse invokes an actor method and unmarshals the response
func (c *DaprClient) InvokeActorMethodWithResponse(ctx context.Context, req ActorMethodRequest, responseObj interface{}) error {
	resp, err := c.InvokeActorMethod(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("actor method returned status %d: %s", resp.StatusCode, string(resp.Body))
	}

	if responseObj != nil {
		if err := json.Unmarshal(resp.Body, responseObj); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// CounterState represents the counter actor state
type CounterState struct {
	Value int `json:"value"`
}

// BankAccountBalance represents the bank account balance response
type BankAccountBalance struct {
	Balance   float64 `json:"balance"`
	OwnerName string  `json:"ownerName"`
	AccountID string  `json:"accountId"`
}

// BankAccountHistory represents the bank account transaction history
type BankAccountHistory struct {
	Transactions []Transaction `json:"transactions"`
}

// Transaction represents a single transaction
type Transaction struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	Timestamp   string  `json:"timestamp"`
	Balance     float64 `json:"balance"`
}

// CreateAccountRequest represents a create account request
type CreateAccountRequest struct {
	OwnerName      string  `json:"ownerName"`
	InitialDeposit float64 `json:"initialDeposit"`
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// WithdrawRequest represents a withdraw request
type WithdrawRequest struct {
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
}

// SetValueRequest represents a set counter value request
type SetValueRequest struct {
	Value int `json:"value"`
}

// CheckHealth verifies that Dapr services are available
func (c *DaprClient) CheckHealth() error {
	// Check Dapr sidecar health
	resp, err := c.httpClient.Get(c.baseURL + "/v1.0/healthz")
	if err != nil {
		return fmt.Errorf("failed to connect to Dapr sidecar at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Dapr sidecar health check failed with status %d", resp.StatusCode)
	}

	// Check actor service health
	actorServiceURL := GetActorServiceEndpoint() + "/health"
	resp, err = c.httpClient.Get(actorServiceURL)
	if err != nil {
		return fmt.Errorf("failed to connect to actor service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("actor service health check failed with status %d", resp.StatusCode)
	}

	return nil
}