package auth

import (
	"context"
	"os"
	"time"
)

// TokenGenerator helps generate JWT tokens using the JWKS mock API
type TokenGenerator struct {
	client *TokenGeneratorClient
}

// NewTokenGenerator creates a new token generator that uses the JWKS mock API
func NewTokenGenerator(generateURL string) *TokenGenerator {
	return &TokenGenerator{
		client: NewTokenGeneratorClient(generateURL),
	}
}

// GenerateToken creates a JWT token with the specified claims using the JWKS service
func (tg *TokenGenerator) GenerateToken(userID, username, email string, roles []string, expiresIn time.Duration) (string, error) {
	return tg.client.GenerateToken(context.Background(), userID, username, email, roles, expiresIn)
}

// GenerateTokenWithCustomClaims creates a JWT token with custom claims using the JWKS service
func (tg *TokenGenerator) GenerateTokenWithCustomClaims(claims map[string]interface{}, expiresIn time.Duration) (string, error) {
	return tg.client.GenerateTokenWithCustomClaims(context.Background(), claims, expiresIn)
}

// GenerateExpiredToken creates an expired JWT token for testing using the JWKS service
func (tg *TokenGenerator) GenerateExpiredToken(userID, username string) (string, error) {
	return tg.client.GenerateExpiredToken(context.Background(), userID, username)
}

// NewDefaultTestGenerator creates a token generator with default test values from environment
func NewDefaultTestGenerator() *TokenGenerator {
	generateURL := os.Getenv("JWKS_GENERATE_URL")
	if generateURL == "" {
		generateURL = "http://localhost:3000/generate-token"
	}
	return NewTokenGenerator(generateURL)
}

