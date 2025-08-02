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

// JWTHelper provides utilities for JWT token management in tests
type JWTHelper struct {
	generateURL string
	issuer      string
	client      *http.Client
}

// NewJWTHelper creates a new JWT helper instance
func NewJWTHelper(generateURL string) (*JWTHelper, error) {
	if generateURL == "" {
		generateURL = "http://localhost:3000/generate-token"
	}
	
	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:3000"
	}
	
	return &JWTHelper{
		generateURL: generateURL,
		issuer:      issuer,
		client:      &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// GenerateToken creates a JWT token with the specified user and roles
func (j *JWTHelper) GenerateToken(userID string, roles []string) (string, error) {
	// Prepare the request payload
	requestBody := map[string]interface{}{
		"claims": map[string]interface{}{
			"sub":      userID,
			"user_id":  userID,
			"username": userID,
			"email":    userID + "@example.com",
			"roles":    roles,
		},
		"expiresIn": 3600, // 1 hour
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", j.generateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := j.client.Do(req)
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

// HealthCheck verifies that the JWKS Mock API is available
func (j *JWTHelper) HealthCheck(ctx context.Context) (bool, error) {
	healthURL := j.issuer + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

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

// generateExpiredTestToken creates an expired JWT token for testing
func generateExpiredTestToken(userID, username string) (string, error) {
	// Generate a token that expires in 1 second, then wait for it to expire
	token, err := generateTestToken(userID, username, "", []string{}, 1*time.Second)
	if err != nil {
		return "", err
	}
	
	// Wait for the token to expire
	time.Sleep(2 * time.Second)
	return token, nil
}