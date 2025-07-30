package auth

import (
	"bytes"
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
	Active    bool                   `json:"active"`
	TokenType string                 `json:"token_type,omitempty"`
	Scope     string                 `json:"scope,omitempty"`
	ClientID  string                 `json:"client_id,omitempty"`
	Username  string                 `json:"username,omitempty"`
	Exp       int64                  `json:"exp,omitempty"`
	Iat       int64                  `json:"iat,omitempty"`
	Nbf       int64                  `json:"nbf,omitempty"`
	Sub       string                 `json:"sub,omitempty"`
	Aud       string                 `json:"aud,omitempty"`
	Iss       string                 `json:"iss,omitempty"`
	Jti       string                 `json:"jti,omitempty"`
	
	// Additional custom claims
	UserID   string   `json:"user_id,omitempty"`
	Email    string   `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	
	// Raw contains all claims for access to any additional fields
	Raw map[string]interface{} `json:"-"`
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
	
	// Parse the raw response to extract all claims
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err == nil {
		response.Raw = rawResponse
		
		// Extract roles if present as array
		if rolesRaw, ok := rawResponse["roles"]; ok {
			if rolesSlice, ok := rolesRaw.([]interface{}); ok {
				roles := make([]string, len(rolesSlice))
				for i, role := range rolesSlice {
					if roleStr, ok := role.(string); ok {
						roles[i] = roleStr
					}
				}
				response.Roles = roles
			}
		}
		
		// Extract additional custom fields
		if userID, ok := rawResponse["user_id"].(string); ok {
			response.UserID = userID
		}
		if email, ok := rawResponse["email"].(string); ok {
			response.Email = email
		}
	}
	
	return &response, nil
}

// TokenGeneratorClient handles token generation using the JWKS mock API
type TokenGeneratorClient struct {
	httpClient   *http.Client
	generateURL  string
}

// TokenGenerationRequest represents a request to generate a token
type TokenGenerationRequest struct {
	Claims    map[string]interface{} `json:"claims"`
	ExpiresIn *int                   `json:"expiresIn,omitempty"` // seconds
}

// TokenGenerationResponse represents the response from token generation
type TokenGenerationResponse struct {
	Token      string                 `json:"token"`
	ExpiresIn  int                    `json:"expires_in"`
	KeyID      string                 `json:"key_id"`
	RawRequest map[string]interface{} `json:"raw_request"`
}

// NewTokenGeneratorClient creates a new token generator client
func NewTokenGeneratorClient(generateURL string) *TokenGeneratorClient {
	return &TokenGeneratorClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		generateURL: generateURL,
	}
}

// GenerateToken generates a new JWT token with the specified claims
func (c *TokenGeneratorClient) GenerateToken(ctx context.Context, userID, username, email string, roles []string, expiresIn time.Duration) (string, error) {
	// Prepare claims
	claims := map[string]interface{}{
		"sub":      userID,
		"user_id":  userID,
		"username": username,
		"email":    email,
		"roles":    roles,
	}
	
	return c.GenerateTokenWithCustomClaims(ctx, claims, expiresIn)
}

// GenerateTokenWithCustomClaims generates a token with custom claims
func (c *TokenGeneratorClient) GenerateTokenWithCustomClaims(ctx context.Context, claims map[string]interface{}, expiresIn time.Duration) (string, error) {
	expiresInSeconds := int(expiresIn.Seconds())
	
	request := TokenGenerationRequest{
		Claims:    claims,
		ExpiresIn: &expiresInSeconds,
	}
	
	// Marshal request
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token generation request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.generateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create token generation request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token generation request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token generation response: %w", err)
	}
	
	// Check status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token generation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var response TokenGenerationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse token generation response: %w", err)
	}
	
	return response.Token, nil
}

// GenerateExpiredToken generates an expired token for testing (using past expiration)
func (c *TokenGeneratorClient) GenerateExpiredToken(ctx context.Context, userID, username string) (string, error) {
	claims := map[string]interface{}{
		"sub":      userID,
		"user_id":  userID,
		"username": username,
	}
	
	// Generate a token that expires immediately (1 second in the past when it arrives)
	return c.GenerateTokenWithCustomClaims(ctx, claims, 1*time.Second)
}

// ConvertIntrospectionToClaims converts an introspection response to JWTClaims for compatibility
func ConvertIntrospectionToClaims(resp *IntrospectionResponse) *JWTClaims {
	if !resp.Active {
		return nil
	}
	
	claims := &JWTClaims{
		Subject:  resp.Sub,
		Issuer:   resp.Iss,
		Audience: resp.Aud,
		UserID:   resp.UserID,
		Username: resp.Username,
		Email:    resp.Email,
		Roles:    resp.Roles,
		Raw:      resp.Raw,
	}
	
	// Convert timestamps
	if resp.Exp != 0 {
		claims.ExpiresAt = time.Unix(resp.Exp, 0)
	}
	if resp.Iat != 0 {
		claims.IssuedAt = time.Unix(resp.Iat, 0)
	}
	if resp.Nbf != 0 {
		claims.NotBefore = time.Unix(resp.Nbf, 0)
	}
	
	return claims
}