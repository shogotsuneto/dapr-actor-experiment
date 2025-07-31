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