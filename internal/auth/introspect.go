package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IntrospectionClient handles OAuth 2.0 token introspection (RFC 7662)
type IntrospectionClient struct {
	httpClient      *http.Client
	introspectURL   string
}

// IntrospectionResponse represents the response from the introspection endpoint
type IntrospectionResponse struct {
	Active bool   `json:"active"`
	Sub    string `json:"sub,omitempty"`
}

// NewIntrospectionClient creates a new introspection client
func NewIntrospectionClient(introspectURL string) *IntrospectionClient {
	return &IntrospectionClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		introspectURL: introspectURL,
	}
}

// IntrospectToken validates a token using the OAuth 2.0 introspection endpoint
func (c *IntrospectionClient) IntrospectToken(ctx context.Context, token string) (*IntrospectionResponse, error) {
	// Prepare form data as required by RFC 7662
	data := url.Values{}
	data.Set("token", token)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create introspection request: %w", err)
	}
	
	// Set required headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	
	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspection request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read introspection response: %w", err)
	}
	
	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection endpoint returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var response IntrospectionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse introspection response: %w", err)
	}
	
	return &response, nil
}


