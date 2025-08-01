package auth

import (
	"context"
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
	// IntrospectURL is the URL of the OAuth 2.0 introspection endpoint
	IntrospectURL string
	
	// SkipPaths are paths that should skip JWT validation
	SkipPaths []string
}

// JWTMiddleware creates HTTP middleware that validates JWT tokens using OAuth 2.0 introspection
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

			// For actor invocations, check if this is a legitimate Dapr sidecar call
			// In a proper Dapr setup, authentication should be handled at the Dapr level
			if strings.HasPrefix(r.URL.Path, "/actors/") {
				// Check if this is coming from Dapr sidecar (has traceparent header and appropriate user agent)
				if isDaprInternalCall(r) {
					// For Dapr internal calls, skip authentication but don't inject user context
					// This allows the actor methods to handle the absence of user context appropriately
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// If no introspection URL is configured, skip authentication
				if config.IntrospectURL == "" {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer token format
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Create introspection client
			introspectClient := NewIntrospectionClient(config.IntrospectURL)
			
			// Validate token using introspection
			userID, err := validateJWTWithIntrospection(r.Context(), tokenString, config, introspectClient)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid JWT token: %v", err), http.StatusUnauthorized)
				return
			}

			// Inject userID into request context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// isDaprInternalCall detects if a request is coming from the Dapr sidecar
func isDaprInternalCall(r *http.Request) bool {
	// Check for Dapr-specific tracing headers (traceparent is added by Dapr)
	if r.Header.Get("Traceparent") != "" {
		// Additional validation: check if it's from the expected internal network
		remoteAddr := r.RemoteAddr
		if colonPos := strings.LastIndex(remoteAddr, ":"); colonPos != -1 {
			host := remoteAddr[:colonPos]
			// Allow calls from Docker internal network (typically 172.x.x.x range)
			if strings.HasPrefix(host, "172.") {
				return true
			}
		}
	}
	
	return false
}

// validateJWTWithIntrospection validates a JWT token using OAuth 2.0 introspection
func validateJWTWithIntrospection(ctx context.Context, tokenString string, config JWTMiddlewareConfig, client *IntrospectionClient) (string, error) {
	// Call introspection endpoint
	resp, err := client.IntrospectToken(ctx, tokenString)
	if err != nil {
		return "", fmt.Errorf("introspection failed: %w", err)
	}
	
	// Check if token is active
	if !resp.Active {
		return "", fmt.Errorf("token is not active")
	}
	
	// Ensure we have a subject claim
	if resp.Sub == "" {
		return "", fmt.Errorf("token missing subject claim")
	}

	return resp.Sub, nil
}

// GetUserID extracts user ID from the given context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

