package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// UserIDContextKey is the context key for user ID
type UserIDContextKey string

const (
	// UserIDKey is the context key for accessing user ID from JWT sub claim
	UserIDKey UserIDContextKey = "user_id"
)



// JWTMiddlewareConfig configures the JWT middleware
type JWTMiddlewareConfig struct {
	// SkipPaths are paths that should skip JWT validation
	SkipPaths []string
}

// JWTMiddleware creates HTTP middleware that extracts user ID from Dapr Bearer middleware headers
func JWTMiddleware(config JWTMiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip validation
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// When using Dapr Bearer middleware, JWT validation is done by Dapr
			// but the middleware forwards the Authorization header as-is instead of
			// individual claim headers. We need to parse the JWT token directly.
			
			// Extract the JWT token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}
			
			// Extract Bearer token
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}
			
			jwtToken := strings.TrimPrefix(authHeader, "Bearer ")
			
			// Parse JWT token to extract claims (simple base64 decode of payload)
			// Since Dapr already validated the token, we can safely parse it
			userID, err := extractSubFromJWT(jwtToken)
			if err != nil {
				http.Error(w, "Invalid JWT token format", http.StatusUnauthorized)
				return
			}

			// Inject userID into request context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts user ID from the given context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// extractSubFromJWT extracts the 'sub' claim from a JWT token payload
// Since Dapr Bearer middleware has already validated the token, we can safely decode it
func extractSubFromJWT(jwtToken string) (string, error) {
	// JWT tokens have 3 parts separated by dots: header.payload.signature
	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT token format")
	}
	
	// Decode the payload (second part)
	payload := parts[1]
	
	// Add padding if needed for base64 decoding
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}
	
	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}
	
	// Parse JSON to extract claims
	var claims map[string]interface{}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}
	
	// Extract 'sub' claim
	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("sub claim not found or invalid")
	}
	
	return sub, nil
}

