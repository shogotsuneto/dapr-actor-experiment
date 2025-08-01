package auth

import (
	"context"
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
			// and the validated claims are forwarded via headers
			
			// Extract userID from the forwarded JWT payload
			// Dapr Bearer middleware forwards JWT claims as headers with "X-" prefix
			// Available headers from Dapr Bearer middleware:
			// - X-Sub: Subject (user ID)
			// - X-Iss: Issuer
			// - X-Aud: Audience  
			// - X-Exp: Expiration (Unix timestamp)
			// - X-Iat: Issued At (Unix timestamp)
			// - X-Nbf: Not Before (Unix timestamp)
			// - X-Jti: JWT ID
			// - X-{claim}: Any custom claims from the JWT
			userID := r.Header.Get("X-Sub")
			if userID == "" {
				// If no user ID forwarded, JWT validation failed at Dapr level
				http.Error(w, "Authentication required", http.StatusUnauthorized)
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

