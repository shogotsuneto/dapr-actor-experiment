package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenGenerator helps generate JWT tokens for testing purposes
type TokenGenerator struct {
	secretKey []byte
	issuer    string
}

// NewTokenGenerator creates a new token generator with the given secret key
func NewTokenGenerator(secretKey []byte, issuer string) *TokenGenerator {
	return &TokenGenerator{
		secretKey: secretKey,
		issuer:    issuer,
	}
}

// GenerateToken creates a JWT token with the specified claims
func (tg *TokenGenerator) GenerateToken(userID, username, email string, roles []string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"user_id":  userID,
		"username": username,
		"email":    email,
		"roles":    roles,
		"iss":      tg.issuer,
		"aud":      "dapr-actor-service",
		"iat":      now.Unix(),
		"exp":      now.Add(expiresIn).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tg.secretKey)
}

// GenerateTokenWithCustomClaims creates a JWT token with custom claims
func (tg *TokenGenerator) GenerateTokenWithCustomClaims(claims map[string]interface{}, expiresIn time.Duration) (string, error) {
	now := time.Now()
	
	// Add standard claims if not present
	if _, exists := claims["iss"]; !exists {
		claims["iss"] = tg.issuer
	}
	if _, exists := claims["aud"]; !exists {
		claims["aud"] = "dapr-actor-service"
	}
	if _, exists := claims["iat"]; !exists {
		claims["iat"] = now.Unix()
	}
	if _, exists := claims["exp"]; !exists {
		claims["exp"] = now.Add(expiresIn).Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	return token.SignedString(tg.secretKey)
}

// GenerateExpiredToken creates an expired JWT token for testing
func (tg *TokenGenerator) GenerateExpiredToken(userID, username string) (string, error) {
	pastTime := time.Now().Add(-1 * time.Hour)
	claims := jwt.MapClaims{
		"sub":      userID,
		"user_id":  userID,
		"username": username,
		"iss":      tg.issuer,
		"aud":      "dapr-actor-service",
		"iat":      pastTime.Unix(),
		"exp":      pastTime.Add(30 * time.Minute).Unix(), // Expired 30 minutes ago
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(tg.secretKey)
}

// Default values for testing
const (
	// DefaultTestSecret is a default secret for testing (never use in production)
	DefaultTestSecret = "test-secret-key-do-not-use-in-production"
	
	// DefaultTestIssuer is a default issuer for testing
	DefaultTestIssuer = "dapr-actor-test"
)

// NewDefaultTestGenerator creates a token generator with default test values
func NewDefaultTestGenerator() *TokenGenerator {
	return NewTokenGenerator([]byte(DefaultTestSecret), DefaultTestIssuer)
}