package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)


// generateTestToken creates a JWT token for testing using the JWKS Mock API
// This is a simple helper that makes direct HTTP calls without a complex client
func generateTestToken(userID, username, email string, roles []string, expiresIn time.Duration) (string, error) {
	jwksURL := os.Getenv("JWKS_GENERATE_URL")
	if jwksURL == "" {
		jwksURL = "http://localhost:3000/generate-token"
	}

	// Prepare the request payload
	requestBody := map[string]interface{}{
		"claims": map[string]interface{}{
			"sub":      userID,
			"user_id":  userID,
			"username": username,
			"email":    email,
			"roles":    roles,
		},
		"expiresIn": int(expiresIn.Seconds()),
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", jwksURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token generation failed with status %d", resp.StatusCode)
	}

	// Parse response
	var response struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Token, nil
}
