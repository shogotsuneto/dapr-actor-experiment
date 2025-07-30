package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// JWTContextKey is the context key for JWT claims
type JWTContextKey string

const (
	// JWTClaimsKey is the context key for accessing JWT claims
	JWTClaimsKey JWTContextKey = "jwt_claims"
)

// JWTClaims represents the claims extracted from a JWT token
type JWTClaims struct {
	// Standard JWT claims
	Subject   string    `json:"sub,omitempty"`
	Issuer    string    `json:"iss,omitempty"`
	Audience  string    `json:"aud,omitempty"`
	ExpiresAt time.Time `json:"exp,omitempty"`
	IssuedAt  time.Time `json:"iat,omitempty"`
	NotBefore time.Time `json:"nbf,omitempty"`
	
	// Custom claims that might be useful for resource ownership
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	
	// Raw claims for any additional custom fields
	Raw map[string]interface{} `json:"-"`
}

// JWTMiddlewareConfig configures the JWT middleware
type JWTMiddlewareConfig struct {
	// IntrospectURL is the URL of the OAuth 2.0 introspection endpoint
	IntrospectURL string
	
	// RequiredIssuer specifies the required issuer claim (optional)
	RequiredIssuer string
	
	// RequiredAudience specifies the required audience claim (optional)
	RequiredAudience string
	
	// SkipPaths are paths that should skip JWT validation
	SkipPaths []string
	
	// Deprecated fields (kept for backward compatibility but ignored)
	SecretKey     []byte      `json:"-"`
	PublicKey     interface{} `json:"-"`
	AllowInsecure bool        `json:"-"`
}

// JWTMiddleware creates HTTP middleware that validates JWT tokens using OAuth 2.0 introspection
func JWTMiddleware(config JWTMiddlewareConfig) func(http.Handler) http.Handler {
	// Create introspection client
	introspectClient := NewIntrospectionClient(config.IntrospectURL)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path should skip validation
			for _, skipPath := range config.SkipPaths {
				if strings.HasPrefix(r.URL.Path, skipPath) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer token format
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
				return
			}

			// Validate token using introspection
			claims, err := validateJWTWithIntrospection(r.Context(), tokenString, config, introspectClient)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid JWT token: %v", err), http.StatusUnauthorized)
				return
			}

			// Inject claims into request context
			ctx := context.WithValue(r.Context(), JWTClaimsKey, claims)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// validateJWTWithIntrospection validates a JWT token using OAuth 2.0 introspection
func validateJWTWithIntrospection(ctx context.Context, tokenString string, config JWTMiddlewareConfig, client *IntrospectionClient) (*JWTClaims, error) {
	// Call introspection endpoint
	resp, err := client.IntrospectToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("introspection failed: %w", err)
	}
	
	// Check if token is active
	if !resp.Active {
		return nil, fmt.Errorf("token is not active")
	}
	
	// Validate required claims if configured
	if config.RequiredIssuer != "" && resp.Iss != config.RequiredIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", config.RequiredIssuer, resp.Iss)
	}

	if config.RequiredAudience != "" && resp.Aud != config.RequiredAudience {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", config.RequiredAudience, resp.Aud)
	}

	// Check expiration
	if resp.Exp != 0 && time.Now().Unix() > resp.Exp {
		return nil, fmt.Errorf("token has expired")
	}

	// Check not before
	if resp.Nbf != 0 && time.Now().Unix() < resp.Nbf {
		return nil, fmt.Errorf("token not valid yet")
	}

	// Convert to JWTClaims format for compatibility
	return ConvertIntrospectionToClaims(resp), nil
}

// GetJWTClaims extracts JWT claims from the given context
func GetJWTClaims(ctx context.Context) (*JWTClaims, bool) {
	claims, ok := ctx.Value(JWTClaimsKey).(*JWTClaims)
	return claims, ok
}

// HasRole checks if the JWT claims contain a specific role
func HasRole(ctx context.Context, role string) bool {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return false
	}
	
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsResourceOwner checks if the JWT subject matches the given owner ID
func IsResourceOwner(ctx context.Context, ownerID string) bool {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return false
	}
	
	// Check multiple possible owner identifiers
	return claims.Subject == ownerID || 
		   claims.UserID == ownerID || 
		   claims.Username == ownerID ||
		   claims.Email == ownerID
}

// GetUserIdentifier returns the best available user identifier from JWT claims
func GetUserIdentifier(ctx context.Context) string {
	claims, ok := GetJWTClaims(ctx)
	if !ok {
		return ""
	}
	
	// Return the first available identifier
	if claims.UserID != "" {
		return claims.UserID
	}
	if claims.Subject != "" {
		return claims.Subject
	}
	if claims.Username != "" {
		return claims.Username
	}
	if claims.Email != "" {
		return claims.Email
	}
	
	return ""
}