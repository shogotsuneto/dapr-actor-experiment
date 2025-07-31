package auth

import (
	"testing"
	"time"
)



func TestConvertIntrospectionToClaims(t *testing.T) {
	// Test active token
	resp := &IntrospectionResponse{
		Active:   true,
		Sub:      "user-123",
		Iss:      "http://test-issuer",
		Aud:      "test-audience",
		Exp:      1640995200,
		Iat:      1640991600,
		UserID:   "user-123",
		Username: "john_doe",
		Email:    "john@example.com",
		Roles:    []string{"user", "admin"},
		Raw: map[string]interface{}{
			"custom_claim": "custom_value",
		},
	}

	claims := ConvertIntrospectionToClaims(resp)
	if claims == nil {
		t.Fatal("Expected claims, got nil")
	}

	if claims.Subject != "user-123" {
		t.Errorf("Expected subject=user-123, got %s", claims.Subject)
	}

	if claims.Issuer != "http://test-issuer" {
		t.Errorf("Expected issuer=http://test-issuer, got %s", claims.Issuer)
	}

	if len(claims.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(claims.Roles))
	}

	expectedTime := time.Unix(1640995200, 0)
	if !claims.ExpiresAt.Equal(expectedTime) {
		t.Errorf("Expected expiry time %v, got %v", expectedTime, claims.ExpiresAt)
	}

	// Test inactive token
	inactiveResp := &IntrospectionResponse{
		Active: false,
	}

	inactiveClaims := ConvertIntrospectionToClaims(inactiveResp)
	if inactiveClaims != nil {
		t.Error("Expected nil claims for inactive token")
	}
}

func TestJWTMiddlewareConfig(t *testing.T) {
	// Test that the config struct is properly defined with new fields
	config := JWTMiddlewareConfig{
		IntrospectURL:    "http://localhost:3000/introspect",
		RequiredIssuer:   "http://localhost:3000",
		RequiredAudience: "test-audience",
		SkipPaths:        []string{"/health", "/status"},
	}

	if config.IntrospectURL != "http://localhost:3000/introspect" {
		t.Errorf("Expected introspect URL to be set")
	}

	if len(config.SkipPaths) != 2 {
		t.Errorf("Expected 2 skip paths, got %d", len(config.SkipPaths))
	}
}