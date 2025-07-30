package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIntrospectionClient_IntrospectToken(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		token          string
		expectedActive bool
		expectedError  bool
		expectedRoles  []string
		expectedSub    string
	}{
		{
			name: "valid token with roles",
			serverResponse: `{
				"active": true,
				"sub": "user-123",
				"user_id": "user-123",
				"username": "john_doe",
				"email": "john@example.com",
				"roles": ["user", "admin"],
				"exp": 1640995200,
				"iat": 1640991600,
				"iss": "http://test-issuer",
				"aud": "test-audience"
			}`,
			serverStatus:   200,
			token:          "valid-token",
			expectedActive: true,
			expectedError:  false,
			expectedRoles:  []string{"user", "admin"},
			expectedSub:    "user-123",
		},
		{
			name: "inactive token",
			serverResponse: `{
				"active": false
			}`,
			serverStatus:   200,
			token:          "invalid-token",
			expectedActive: false,
			expectedError:  false,
		},
		{
			name:           "server error",
			serverResponse: `{"error": "internal server error"}`,
			serverStatus:   500,
			token:          "any-token",
			expectedActive: false,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request format
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("Expected application/x-www-form-urlencoded content type")
				}
				
				// Parse form data
				r.ParseForm()
				if r.FormValue("token") != tt.token {
					t.Errorf("Expected token %s, got %s", tt.token, r.FormValue("token"))
				}
				
				// Send response
				w.WriteHeader(tt.serverStatus)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create client and test
			client := NewIntrospectionClient(server.URL)
			resp, err := client.IntrospectToken(context.Background(), tt.token)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if resp.Active != tt.expectedActive {
				t.Errorf("Expected active=%v, got %v", tt.expectedActive, resp.Active)
			}

			if tt.expectedActive {
				if resp.Sub != tt.expectedSub {
					t.Errorf("Expected sub=%s, got %s", tt.expectedSub, resp.Sub)
				}

				if len(resp.Roles) != len(tt.expectedRoles) {
					t.Errorf("Expected %d roles, got %d", len(tt.expectedRoles), len(resp.Roles))
				} else {
					for i, role := range tt.expectedRoles {
						if resp.Roles[i] != role {
							t.Errorf("Expected role[%d]=%s, got %s", i, role, resp.Roles[i])
						}
					}
				}
			}
		})
	}
}

func TestTokenGeneratorClient_GenerateToken(t *testing.T) {
	// Create test server that simulates JWKS mock API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type")
		}
		
		// Parse request
		var req TokenGenerationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			w.WriteHeader(400)
			return
		}
		
		// Verify claims
		if req.Claims["sub"] != "test-user" {
			t.Errorf("Expected sub=test-user, got %v", req.Claims["sub"])
		}
		
		// Send response
		response := TokenGenerationResponse{
			Token:     "test-jwt-token-12345",
			ExpiresIn: 3600,
			KeyID:     "key-1",
			RawRequest: req.Claims,
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client and test
	client := NewTokenGeneratorClient(server.URL)
	token, err := client.GenerateToken(
		context.Background(),
		"test-user",
		"testuser",
		"test@example.com",
		[]string{"user"},
		1*time.Hour,
	)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	expectedToken := "test-jwt-token-12345"
	if token != expectedToken {
		t.Errorf("Expected token %s, got %s", expectedToken, token)
	}
}

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
		// Deprecated fields should be ignored
		SecretKey:     []byte("deprecated"),
		AllowInsecure: true,
	}

	if config.IntrospectURL != "http://localhost:3000/introspect" {
		t.Errorf("Expected introspect URL to be set")
	}

	if len(config.SkipPaths) != 2 {
		t.Errorf("Expected 2 skip paths, got %d", len(config.SkipPaths))
	}
}