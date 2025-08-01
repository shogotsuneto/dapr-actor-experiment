package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
)

func TestJWTAwareCounterActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewDaprClient(GetDaprEndpoint())
	require.NoError(t, client.CheckHealth(), "Dapr services must be running for this test")

	t.Run("AdminCanAccessAllCounterOperations", func(t *testing.T) {
		// Generate admin token
		adminToken, err := generateTestToken(
			"admin-001", "admin", "admin@example.com",
			[]string{"admin", "counter_admin"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test GET operation
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "Get",
		}, adminToken, nil)
		require.NoError(t, err)

		// Test SET operation (requires admin role)
		setReq := counter.SetValueRequest{Value: 42}
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "Set",
			Data:      setReq,
		}, adminToken, nil)
		require.NoError(t, err)

		// Verify the value was set
		var result counter.CounterState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin",
			Method:    "Get",
		}, adminToken, &result)
		require.NoError(t, err)
		assert.Equal(t, int32(42), result.Value)
	})

	t.Run("RegularUserCanSetCounterValue", func(t *testing.T) {
		// Generate regular user token (without admin roles)
		userToken, err := generateTestToken(
			"user-123", "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test that regular user can read
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-user",
			Method:    "Get",
		}, userToken, nil)
		require.NoError(t, err)

		// Test that regular user CAN set value (authentication is sufficient)
		var result counter.CounterState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-user",
			Method:    "Set",
			Data:      counter.SetValueRequest{Value: 99},
		}, userToken, &result)
		require.NoError(t, err)
		assert.Equal(t, int32(99), result.Value)
	})

	t.Run("AnyAuthenticatedUserCanSetValue", func(t *testing.T) {
		// Generate user token (roles don't matter for Counter)
		adminUserToken, err := generateTestToken(
			"user-456", "jane_smith", "jane@example.com",
			[]string{"user", "counter_admin"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Test that any authenticated user can set value
		var result counter.CounterState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin-user",
			Method:    "Set",
			Data:      counter.SetValueRequest{Value: 777},
		}, adminUserToken, &result)
		require.NoError(t, err)
		assert.Equal(t, int32(777), result.Value)

		// Verify the value was set
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-admin-user",
			Method:    "Get",
		}, adminUserToken, &result)
		require.NoError(t, err)
		assert.Equal(t, int32(777), result.Value)
	})

	t.Run("MissingTokenIsRejected", func(t *testing.T) {
		// Test that missing token is rejected
		resp, err := client.InvokeActorMethod(context.Background(), ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   "test-counter-no-token",
			Method:    "Get",
		})
		
		require.NoError(t, err) // HTTP call should succeed
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode) // But should return 401
		assert.Contains(t, string(resp.Body), "Unauthorized")
	})
}

func TestJWTAwareBankAccountActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewDaprClient(GetDaprEndpoint())
	require.NoError(t, client.CheckHealth(), "Dapr services must be running for this test")

	t.Run("UserCanCreateAndAccessOwnAccount", func(t *testing.T) {
		// Use unique user ID for this test
		userID := fmt.Sprintf("user-jwt-test-%d", time.Now().UnixNano()%10000)
		
		// Generate user token
		userToken, err := generateTestToken(
			userID, "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		accountID := userID // Same as user ID for ownership

		// Create account
		var createResult bankaccount.BankAccountState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "CreateAccount",
			Data:      bankaccount.CreateAccountRequest{
				OwnerName:      "John Doe",
				InitialDeposit: 1000.0,
			},
		}, userToken, &createResult)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", createResult.OwnerName)
		assert.Equal(t, 1000.0, createResult.Balance)

		// Test deposit
		var depositResult bankaccount.BankAccountState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "Deposit",
			Data:      bankaccount.DepositRequest{
				Amount:      250.0,
				Description: "Salary deposit",
			},
		}, userToken, &depositResult)
		require.NoError(t, err)
		assert.Equal(t, 1250.0, depositResult.Balance)

		// Test get balance
		var balanceResult bankaccount.BankAccountState
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   accountID,
			Method:    "GetBalance",
		}, userToken, &balanceResult)
		require.NoError(t, err)
		assert.Equal(t, 1250.0, balanceResult.Balance)
	})

	t.Run("UserCannotAccessOthersAccount", func(t *testing.T) {
		// Generate token for user-123 
		userID := fmt.Sprintf("user-jwt-access-%d", time.Now().UnixNano()%10000)
		user1Token, err := generateTestToken(
			userID, "john_doe", "john@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// User1 tries to access User2's account (different user ID)
		otherUserID := fmt.Sprintf("other-user-%d", time.Now().UnixNano()%10000)
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   otherUserID, // Different user ID
			Method:    "GetBalance",
		}, user1Token, nil) // User1's token

		require.Error(t, err)
		// Check for authorization failure (500 error indicates access denied)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("UserCanOnlyCreateOwnAccount", func(t *testing.T) {
		// Generate token for a user
		userID := fmt.Sprintf("user-jwt-create-%d", time.Now().UnixNano()%10000)
		userToken, err := generateTestToken(
			userID, "bob_wilson", "bob@example.com",
			[]string{"user"}, 1*time.Hour,
		)
		require.NoError(t, err)

		// Try to create account with different ID than user ID  
		differentUserID := fmt.Sprintf("different-user-%d", time.Now().UnixNano()%10000)
		_, err = client.InvokeActorMethodWithJWT(context.Background(), ActorMethodRequest{
			ActorType: "BankAccount",
			ActorID:   differentUserID, // Different from userID
			Method:    "CreateAccount",
			Data:      bankaccount.CreateAccountRequest{
				OwnerName:      "Someone Else",
				InitialDeposit: 500.0,
			},
		}, userToken, nil)

		require.Error(t, err)
		// Check for authorization failure (500 error indicates access denied)
		assert.Contains(t, err.Error(), "500")
	})
}