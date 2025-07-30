package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
	// SecretKey is the key used to verify JWT signatures (for HS256)
	SecretKey []byte
	
	// PublicKey is the key used to verify JWT signatures (for RS256/ES256)
	// If both SecretKey and PublicKey are provided, SecretKey takes precedence
	PublicKey interface{}
	
	// RequiredIssuer specifies the required issuer claim (optional)
	RequiredIssuer string
	
	// RequiredAudience specifies the required audience claim (optional)
	RequiredAudience string
	
	// SkipPaths are paths that should skip JWT validation
	SkipPaths []string
	
	// AllowInsecure allows tokens without proper verification (for testing only)
	AllowInsecure bool
}

// JWTMiddleware creates HTTP middleware that validates JWT tokens and injects claims into context
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

			// Parse and validate token
			claims, err := validateJWT(tokenString, config)
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

// validateJWT validates a JWT token and extracts claims
func validateJWT(tokenString string, config JWTMiddlewareConfig) (*JWTClaims, error) {
	// Parse token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		switch token.Method.(type) {
		case *jwt.SigningMethodHMAC:
			if len(config.SecretKey) == 0 {
				return nil, fmt.Errorf("HMAC secret key not configured")
			}
			return config.SecretKey, nil
		case *jwt.SigningMethodRSA:
			if config.PublicKey == nil {
				return nil, fmt.Errorf("RSA public key not configured")
			}
			return config.PublicKey, nil
		case *jwt.SigningMethodECDSA:
			if config.PublicKey == nil {
				return nil, fmt.Errorf("ECDSA public key not configured")
			}
			return config.PublicKey, nil
		default:
			if config.AllowInsecure {
				// For testing - accept unsigned tokens
				return []byte(""), nil
			}
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid && !config.AllowInsecure {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	claims := &JWTClaims{
		Raw: mapClaims,
	}

	// Extract standard claims
	if sub, ok := mapClaims["sub"].(string); ok {
		claims.Subject = sub
	}
	
	if iss, ok := mapClaims["iss"].(string); ok {
		claims.Issuer = iss
	}
	
	if aud, ok := mapClaims["aud"].(string); ok {
		claims.Audience = aud
	}

	// Handle exp claim (can be float64 or int64)
	if exp, ok := mapClaims["exp"]; ok {
		if expFloat, ok := exp.(float64); ok {
			claims.ExpiresAt = time.Unix(int64(expFloat), 0)
		} else if expInt, ok := exp.(int64); ok {
			claims.ExpiresAt = time.Unix(expInt, 0)
		}
	}

	// Handle iat claim
	if iat, ok := mapClaims["iat"]; ok {
		if iatFloat, ok := iat.(float64); ok {
			claims.IssuedAt = time.Unix(int64(iatFloat), 0)
		} else if iatInt, ok := iat.(int64); ok {
			claims.IssuedAt = time.Unix(iatInt, 0)
		}
	}

	// Handle nbf claim
	if nbf, ok := mapClaims["nbf"]; ok {
		if nbfFloat, ok := nbf.(float64); ok {
			claims.NotBefore = time.Unix(int64(nbfFloat), 0)
		} else if nbfInt, ok := nbf.(int64); ok {
			claims.NotBefore = time.Unix(nbfInt, 0)
		}
	}

	// Extract custom claims
	if userID, ok := mapClaims["user_id"].(string); ok {
		claims.UserID = userID
	}
	
	if username, ok := mapClaims["username"].(string); ok {
		claims.Username = username
	}
	
	if email, ok := mapClaims["email"].(string); ok {
		claims.Email = email
	}

	// Handle roles (can be string array)
	if rolesRaw, ok := mapClaims["roles"]; ok {
		if rolesSlice, ok := rolesRaw.([]interface{}); ok {
			roles := make([]string, len(rolesSlice))
			for i, role := range rolesSlice {
				if roleStr, ok := role.(string); ok {
					roles[i] = roleStr
				}
			}
			claims.Roles = roles
		}
	}

	// Validate required claims
	if config.RequiredIssuer != "" && claims.Issuer != config.RequiredIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", config.RequiredIssuer, claims.Issuer)
	}

	if config.RequiredAudience != "" && claims.Audience != config.RequiredAudience {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", config.RequiredAudience, claims.Audience)
	}

	// Validate expiration
	if !claims.ExpiresAt.IsZero() && time.Now().After(claims.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	// Validate not before
	if !claims.NotBefore.IsZero() && time.Now().Before(claims.NotBefore) {
		return nil, fmt.Errorf("token not valid yet")
	}

	return claims, nil
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