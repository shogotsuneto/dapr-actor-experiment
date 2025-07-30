package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
)

func main() {
	fmt.Println("=== JWT Token Generator for Testing (using JWKS Mock API) ===")
	fmt.Println()

	// Check if JWKS URL is configured
	generateURL := os.Getenv("JWKS_GENERATE_URL")
	if generateURL == "" {
		generateURL = "http://localhost:3000/generate-token"
	}

	fmt.Printf("Using JWKS Mock API at: %s\n", generateURL)
	fmt.Println()

	// Create token generator that uses JWKS API
	generator := auth.NewDefaultTestGenerator()
	
	// Generate tokens for different users and scenarios
	
	// 1. Admin user token
	adminToken, err := generator.GenerateToken(
		"admin-001", 
		"admin", 
		"admin@example.com", 
		[]string{"admin", "counter_admin", "bank_admin"}, 
		1*time.Hour,
	)
	if err != nil {
		log.Printf("Warning: Failed to generate admin token (ensure JWKS Mock API is running): %v", err)
		adminToken = "PLACEHOLDER_ADMIN_TOKEN"
	}
	
	// 2. Regular user token
	userToken, err := generator.GenerateToken(
		"user-123", 
		"john_doe", 
		"john@example.com", 
		[]string{"user"}, 
		1*time.Hour,
	)
	if err != nil {
		log.Printf("Warning: Failed to generate user token (ensure JWKS Mock API is running): %v", err)
		userToken = "PLACEHOLDER_USER_TOKEN"
	}
	
	// 3. Another user token
	user2Token, err := generator.GenerateToken(
		"user-456", 
		"jane_smith", 
		"jane@example.com", 
		[]string{"user", "counter_admin"}, 
		1*time.Hour,
	)
	if err != nil {
		log.Printf("Warning: Failed to generate user2 token (ensure JWKS Mock API is running): %v", err)
		user2Token = "PLACEHOLDER_USER2_TOKEN"
	}
	
	// 4. Expired token
	expiredToken, err := generator.GenerateExpiredToken("expired-user", "expired")
	if err != nil {
		log.Printf("Warning: Failed to generate expired token (ensure JWKS Mock API is running): %v", err)
		expiredToken = "PLACEHOLDER_EXPIRED_TOKEN"
	}
	
	// Display tokens
	fmt.Printf("Admin Token (user: admin-001, roles: admin, counter_admin, bank_admin):\n")
	fmt.Printf("Bearer %s\n\n", adminToken)
	
	fmt.Printf("User Token (user: user-123, roles: user):\n")
	fmt.Printf("Bearer %s\n\n", userToken)
	
	fmt.Printf("User2 Token (user: user-456, roles: user, counter_admin):\n")
	fmt.Printf("Bearer %s\n\n", user2Token)
	
	fmt.Printf("Expired Token (for testing expired scenarios):\n")
	fmt.Printf("Bearer %s\n\n", expiredToken)
	
	// Example curl commands
	fmt.Println("=== Example curl commands ===")
	fmt.Println()
	
	fmt.Println("# Test admin user accessing counter-1:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:3500/v1.0/actors/Counter/counter-1/method/get\n\n", adminToken)
	
	fmt.Println("# Test regular user creating their own bank account (user-123):")
	fmt.Printf("curl -X POST -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\" -d '{\"ownerName\": \"John Doe\", \"initialDeposit\": 1000}' http://localhost:3500/v1.0/actors/BankAccount/user-123/method/createAccount\n\n", userToken)
	
	fmt.Println("# Test user trying to access someone else's account (should fail):")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:3500/v1.0/actors/BankAccount/user-456/method/getBalance\n\n", userToken)
	
	fmt.Println("# Test user2 accessing their own account:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:3500/v1.0/actors/BankAccount/user-456/method/getBalance\n\n", user2Token)
	
	fmt.Println("# Test admin accessing any account:")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:3500/v1.0/actors/BankAccount/user-123/method/getBalance\n\n", adminToken)
	
	fmt.Println("# Test user2 setting counter value (has counter_admin role):")
	fmt.Printf("curl -X POST -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\" -d '{\"value\": 42}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/set\n\n", user2Token)
	
	fmt.Println("# Test regular user trying to set counter value (should fail):")
	fmt.Printf("curl -X POST -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\" -d '{\"value\": 99}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/set\n\n", userToken)
	
	fmt.Println("# Test with expired token (should fail):")
	fmt.Printf("curl -H \"Authorization: Bearer %s\" http://localhost:3500/v1.0/actors/Counter/counter-1/method/get\n\n", expiredToken)
	
	fmt.Println("# Test health endpoint (should work without token):")
	fmt.Println("curl http://localhost:8080/health")
	fmt.Println()
	
	// JWKS API information
	fmt.Println("=== JWKS Mock API Setup ===")
	fmt.Println("This token generator now uses the JWKS Mock API for token generation.")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("1. Start the JWKS Mock API service:")
	fmt.Println("   docker run -p 3000:3000 ghcr.io/shogotsuneto/jwks-mock-api:v0.0.4")
	fmt.Println()
	fmt.Println("2. Or use docker-compose (includes JWKS API):")
	fmt.Println("   docker compose up -d")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Printf("   JWKS_GENERATE_URL=%s\n", generateURL)
	fmt.Printf("   JWT_ISSUER=%s\n", os.Getenv("JWT_ISSUER"))
	fmt.Println()
	fmt.Println("Manual token generation using curl:")
	fmt.Printf("curl -X POST %s \\\n", generateURL)
	fmt.Println("  -H 'Content-Type: application/json' \\")
	fmt.Println("  -d '{")
	fmt.Println("    \"claims\": {")
	fmt.Println("      \"sub\": \"user-123\",")
	fmt.Println("      \"user_id\": \"user-123\",")
	fmt.Println("      \"username\": \"john_doe\",")
	fmt.Println("      \"email\": \"john@example.com\",")
	fmt.Println("      \"roles\": [\"user\"]")
	fmt.Println("    },")
	fmt.Println("    \"expiresIn\": 3600")
	fmt.Println("  }'")
	fmt.Println()
	fmt.Println("Check token validity:")
	fmt.Printf("curl -X POST %s \\\n", os.Getenv("JWKS_INTROSPECT_URL"))
	fmt.Println("  -H 'Content-Type: application/x-www-form-urlencoded' \\")
	fmt.Println("  -d 'token=YOUR_TOKEN_HERE'")
}