package integration

import (
	"context"
	"testing"
	"time"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTAwareCounterActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewDaprClient(GetDaprEndpoint())
	require.NoError(t, client.CheckHealth(), "Dapr services must be running for this test")

	generator := auth.NewDefaultTestGenerator()

	t.Run("AdminCanAccessAllCounterOperations", func(t *testing.T) {
		// Generate admin token
		adminToken, err := generator.GenerateToken(
			"admin-001", "admin", "admin@example.com",
			[]string{"admin", "counter_admin"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test GET operation
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "get",
		}, adminToken, nil)
		require.NoError(t, err)

		// Test SET operation (requires admin role)
		setReq := map[string]interface{}{"value": 42}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "set",
			Data:      setReq,
		}, adminToken, nil)
		require.NoError(t, err)

		// Verify the value was set
		var result map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "get",
		}, adminToken, &result)
		require.NoError(t, err)
		assert.Equal(t, float64(42), result["value"])
	})

	t.Run("RegularUserCannotSetCounterValue", func(t *testing.T) {
		// Generate regular user token (without admin roles)
		userToken, err := generator.GenerateToken(
			"user-123", "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test that regular user can read
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-user",
			Method:    "get",
		}, userToken, nil)
		require.NoError(t, err)

		// Test that regular user cannot set value
		setReq := map[string]interface{}{"value": 99}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-user",
			Method:    "set",
			Data:      setReq,
		}, userToken, nil)
		
		// Should get 401 or an error response
		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("CounterAdminCanSetValue", func(t *testing.T) {
		// Generate user token with counter_admin role
		adminUserToken, err := generator.GenerateToken(
			"user-456", "jane_smith", "jane@example.com",
			[]string{"user", "counter_admin"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test that counter admin can set value
		setReq := map[string]interface{}{"value": 777}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin-user",
			Method:    "set",
			Data:      setReq,
		}, adminUserToken, nil)
		require.NoError(t, err)

		// Verify the value was set
		var result map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin-user",
			Method:    "get",
		}, adminUserToken, &result)
		require.NoError(t, err)
		assert.Equal(t, float64(777), result["value"])
	})

	t.Run("ExpiredTokenIsRejected", func(t *testing.T) {
		// Generate expired token
		expiredToken, err := generator.GenerateExpiredToken("expired-user", "expired")
		require.NoError(t, err)

		// Test that expired token is rejected
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-expired",
			Method:    "get",
		}, expiredToken, nil)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("MissingTokenIsRejected", func(t *testing.T) {
		// Test that missing token is rejected
		_, err := client.InvokeActorMethod(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-no-token",
			Method:    "get",
		})
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

func TestJWTAwareBankAccountActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewDaprClient(GetDaprEndpoint())
	require.NoError(t, client.CheckHealth(), "Dapr services must be running for this test")

	generator := auth.NewDefaultTestGenerator()

	t.Run("UserCanCreateAndAccessOwnAccount", func(t *testing.T) {
		// Generate user token
		userToken, err := generator.GenerateToken(
			"user-123", "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		accountID := "user-123" // Same as user ID for ownership

		// Create account
		createReq := map[string]interface{}{
			"ownerName":      "John Doe",
			"initialDeposit": 1000.0,
		}
		var createResult map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "createAccount",
			Data:      createReq,
		}, userToken, &createResult)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", createResult["ownerName"])
		assert.Equal(t, float64(1000), createResult["balance"])

		// Test deposit
		depositReq := map[string]interface{}{
			"amount":      250.0,
			"description": "Salary deposit",
		}
		var depositResult map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "deposit",
			Data:      depositReq,
		}, userToken, &depositResult)
		require.NoError(t, err)
		assert.Equal(t, float64(1250), depositResult["balance"])

		// Test get balance
		var balanceResult map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "getBalance",
		}, userToken, &balanceResult)
		require.NoError(t, err)
		assert.Equal(t, float64(1250), balanceResult["balance"])
	})

	t.Run("UserCannotAccessOthersAccount", func(t *testing.T) {
		// Generate tokens for two different users
		user1Token, err := generator.GenerateToken(
			"user-123", "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// User1 tries to access User2's account
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   "user-456", // User2's account
			Method:    "getBalance",
		}, user1Token, nil) // User1's token

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("AdminCanAccessAnyAccount", func(t *testing.T) {
		// Generate admin token
		adminToken, err := generator.GenerateToken(
			"admin-001", "admin", "admin@example.com",
			[]string{"admin", "bank_admin"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Admin can access any account (using previous test's account)
		var balanceResult map[string]interface{}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   "user-123", // Another user's account
			Method:    "getBalance",
		}, adminToken, &balanceResult)
		require.NoError(t, err)
		// Should get balance without error
	})

	t.Run("UserCannotCreateAccountForOthers", func(t *testing.T) {
		// Generate user token
		userToken, err := generator.GenerateToken(
			"user-789", "bob_wilson", "bob@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Try to create account with different ID than user ID
		createReq := map[string]interface{}{
			"ownerName":      "Someone Else",
			"initialDeposit": 500.0,
		}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   "different-user-id", // Different from user-789
			Method:    "createAccount",
			Data:      createReq,
		}, userToken, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})
}