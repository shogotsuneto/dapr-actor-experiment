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
	// IntrospectURL when set, enables JWT authentication (Bearer middleware should be configured)
	// When empty, authentication is disabled for development
	IntrospectURL string
	
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
			userID := r.Header.Get("X-Sub")
			if userID == "" {
				// If authentication is not configured, skip validation
				if config.IntrospectURL == "" {
					next.ServeHTTP(w, r)
					return
				}
				// If authentication is configured but no user ID forwarded, 
				// it means JWT validation failed at Dapr level
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

