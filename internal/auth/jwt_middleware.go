package auth

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig holds the configuration for JWT validation
type JWTConfig struct {
	JWKSUrl  string
	Issuer   string
	Audience string
}

// JWTMiddleware provides JWT validation functionality
type JWTMiddleware struct {
	config *JWTConfig
	jwks   *keyfunc.JWKS
}

// NewJWTMiddleware creates a new JWT middleware instance
func NewJWTMiddleware(config *JWTConfig) (*JWTMiddleware, error) {
	// Create a JWKS client with auto-refresh
	jwks, err := keyfunc.Get(config.JWKSUrl, keyfunc.Options{
		RefreshInterval: time.Hour, // Refresh JWKS every hour
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS client: %w", err)
	}

	return &JWTMiddleware{
		config: config,
		jwks:   jwks,
	}, nil
}

// ValidateJWT validates a JWT token string
func (m *JWTMiddleware) ValidateJWT(tokenString string) (*jwt.Token, error) {
	// Parse and validate the token
	token, err := jwt.Parse(tokenString, m.jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Verify the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	// Get claims as a map
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer if configured
	if m.config.Issuer != "" {
		if iss, ok := claims["iss"]; !ok || iss != m.config.Issuer {
			return nil, fmt.Errorf("invalid issuer: expected %s, got %v", m.config.Issuer, iss)
		}
	}

	// Validate audience if configured
	if m.config.Audience != "" {
		if aud, ok := claims["aud"]; !ok || aud != m.config.Audience {
			return nil, fmt.Errorf("invalid audience: expected %s, got %v", m.config.Audience, aud)
		}
	}

	return token, nil
}

// ExtractTokenFromAuthHeader extracts the JWT token from an Authorization header value
func ExtractTokenFromAuthHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header not found")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("token not found in authorization header")
	}

	return token, nil
}

// GetTokenClaims returns the claims from a validated JWT token
func GetTokenClaims(token *jwt.Token) (jwt.MapClaims, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// Close cleans up the JWKS client
func (m *JWTMiddleware) Close() {
	if m.jwks != nil {
		m.jwks.EndBackground()
	}
}

// GetJWKSUrl returns the JWKS URL for this middleware
func (m *JWTMiddleware) GetJWKSUrl() string {
	if m.config == nil {
		return ""
	}
	return m.config.JWKSUrl
}

// ValidateTokenString is a convenience method that validates a token string and returns claims
func (m *JWTMiddleware) ValidateTokenString(tokenString string) (jwt.MapClaims, error) {
	token, err := m.ValidateJWT(tokenString)
	if err != nil {
		return nil, err
	}
	
	claims, err := GetTokenClaims(token)
	if err != nil {
		return nil, err
	}
	
	log.Printf("JWT validation successful for subject: %v", claims["sub"])
	return claims, nil
}