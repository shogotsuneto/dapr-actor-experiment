package main

import (
	"fmt"
	"log"
	"time"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
)

func main() {
	// Create token generator
	generator := auth.NewDefaultTestGenerator()
	
	fmt.Println("=== JWT Token Generator for Testing ===")
	fmt.Println()
	
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
		log.Fatalf("Failed to generate admin token: %v", err)
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
		log.Fatalf("Failed to generate user token: %v", err)
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
		log.Fatalf("Failed to generate user2 token: %v", err)
	}
	
	// 4. Expired token
	expiredToken, err := generator.GenerateExpiredToken("expired-user", "expired")
	if err != nil {
		log.Fatalf("Failed to generate expired token: %v", err)
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
	
	// Configuration info
	fmt.Println("=== Environment Configuration ===")
	fmt.Println("To run the service with default test configuration:")
	fmt.Println("JWT_SECRET=test-secret-key-do-not-use-in-production")
	fmt.Println("JWT_ISSUER=dapr-actor-test")
	fmt.Println("JWT_INSECURE_MODE=false")
	fmt.Println()
	fmt.Println("Or start the service and it will use these defaults automatically.")
}