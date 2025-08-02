package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKubernetesMultiNodeDeployment tests the application running on multiple Kubernetes nodes
func TestKubernetesMultiNodeDeployment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Kubernetes integration test in short mode")
	}

	// Skip if not running against Kubernetes
	if os.Getenv("KUBERNETES_TEST") != "true" {
		t.Skip("Skipping Kubernetes test (set KUBERNETES_TEST=true to enable)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Use environment variables for endpoints (configured by k8s-test.sh)
	daprEndpoint := os.Getenv("DAPR_HTTP_ENDPOINT")
	if daprEndpoint == "" {
		daprEndpoint = "http://localhost:3500"
	}

	jwksGenerateURL := os.Getenv("JWKS_GENERATE_URL")
	if jwksGenerateURL == "" {
		jwksGenerateURL = "http://localhost:3000/generate-token"
	}

	daprClient := NewDaprClient(daprEndpoint)
	jwtHelper, err := NewJWTHelper(jwksGenerateURL)
	require.NoError(t, err, "Failed to create JWT helper")

	// Generate JWT token for authentication
	token, err := jwtHelper.GenerateToken("test-user", []string{"user"})
	require.NoError(t, err, "Failed to generate JWT token")

	t.Run("MultiNodeCounterActors", func(t *testing.T) {
		// Test multiple counter actors on different nodes
		actorIDs := []string{"k8s-counter-1", "k8s-counter-2", "k8s-counter-3"}
		
		for _, actorID := range actorIDs {
			t.Run(fmt.Sprintf("CounterActor_%s", actorID), func(t *testing.T) {
				// Initialize counter
				err := daprClient.InvokeActorMethod(ctx, "CounterActor", actorID, "set", map[string]interface{}{"value": 10}, token)
				require.NoError(t, err, "Failed to set initial counter value")

				// Increment multiple times
				for i := 0; i < 5; i++ {
					err = daprClient.InvokeActorMethod(ctx, "CounterActor", actorID, "increment", nil, token)
					require.NoError(t, err, "Failed to increment counter")
				}

				// Verify final value
				var result map[string]interface{}
				err = daprClient.InvokeActorMethodWithResult(ctx, "CounterActor", actorID, "get", nil, &result, token)
				require.NoError(t, err, "Failed to get counter value")
				
				value, ok := result["value"].(float64)
				require.True(t, ok, "Counter value should be a number")
				assert.Equal(t, float64(15), value, "Counter should be 15 after 5 increments from 10")
			})
		}
	})

	t.Run("MultiNodeBankAccounts", func(t *testing.T) {
		// Test multiple bank account actors on different nodes
		accounts := []struct {
			ID      string
			Owner   string
			Initial float64
		}{
			{"k8s-account-alice", "Alice Johnson", 1000.0},
			{"k8s-account-bob", "Bob Smith", 2000.0},
			{"k8s-account-charlie", "Charlie Brown", 500.0},
		}

		for _, account := range accounts {
			t.Run(fmt.Sprintf("BankAccount_%s", account.ID), func(t *testing.T) {
				// Create account
				createReq := map[string]interface{}{
					"ownerName":      account.Owner,
					"initialDeposit": account.Initial,
				}
				err := daprClient.InvokeActorMethod(ctx, "BankAccountActor", account.ID, "createAccount", createReq, token)
				require.NoError(t, err, "Failed to create bank account")

				// Make deposits
				depositReq := map[string]interface{}{
					"amount":      250.0,
					"description": "Kubernetes test deposit",
				}
				err = daprClient.InvokeActorMethod(ctx, "BankAccountActor", account.ID, "deposit", depositReq, token)
				require.NoError(t, err, "Failed to deposit money")

				// Make withdrawal
				withdrawReq := map[string]interface{}{
					"amount":      100.0,
					"description": "Kubernetes test withdrawal",
				}
				err = daprClient.InvokeActorMethod(ctx, "BankAccountActor", account.ID, "withdraw", withdrawReq, token)
				require.NoError(t, err, "Failed to withdraw money")

				// Check balance
				var balanceResult map[string]interface{}
				err = daprClient.InvokeActorMethodWithResult(ctx, "BankAccountActor", account.ID, "getBalance", nil, &balanceResult, token)
				require.NoError(t, err, "Failed to get balance")
				
				balance, ok := balanceResult["balance"].(float64)
				require.True(t, ok, "Balance should be a number")
				expectedBalance := account.Initial + 250.0 - 100.0
				assert.Equal(t, expectedBalance, balance, "Balance should match expected value")

				// Verify transaction history
				var historyResult map[string]interface{}
				err = daprClient.InvokeActorMethodWithResult(ctx, "BankAccountActor", account.ID, "getHistory", nil, &historyResult, token)
				require.NoError(t, err, "Failed to get transaction history")
				
				transactions, ok := historyResult["transactions"].([]interface{})
				require.True(t, ok, "History should contain transactions array")
				assert.Len(t, transactions, 3, "Should have 3 transactions (create, deposit, withdraw)")
			})
		}
	})

	t.Run("CrossNodeActorInteraction", func(t *testing.T) {
		// Test that actors can interact across different nodes
		// This demonstrates that Dapr placement service is working correctly
		
		// Create multiple actors of the same type with different IDs
		// They should be distributed across available nodes
		actorPrefix := "cross-node-test"
		numActors := 6 // More than the number of pods to ensure distribution
		
		// Initialize all actors
		for i := 0; i < numActors; i++ {
			actorID := fmt.Sprintf("%s-%d", actorPrefix, i)
			initialValue := i * 10
			
			setReq := map[string]interface{}{"value": initialValue}
			err := daprClient.InvokeActorMethod(ctx, "CounterActor", actorID, "set", setReq, token)
			require.NoError(t, err, "Failed to set counter value for actor %s", actorID)
		}
		
		// Perform operations on all actors
		for i := 0; i < numActors; i++ {
			actorID := fmt.Sprintf("%s-%d", actorPrefix, i)
			
			// Increment each actor
			err := daprClient.InvokeActorMethod(ctx, "CounterActor", actorID, "increment", nil, token)
			require.NoError(t, err, "Failed to increment counter for actor %s", actorID)
			
			// Verify the value
			var result map[string]interface{}
			err = daprClient.InvokeActorMethodWithResult(ctx, "CounterActor", actorID, "get", nil, &result, token)
			require.NoError(t, err, "Failed to get counter value for actor %s", actorID)
			
			value, ok := result["value"].(float64)
			require.True(t, ok, "Counter value should be a number for actor %s", actorID)
			expectedValue := float64(i*10 + 1) // initial value + 1
			assert.Equal(t, expectedValue, value, "Actor %s should have value %f", actorID, expectedValue)
		}
	})

	t.Run("HealthChecks", func(t *testing.T) {
		// Verify that all services are healthy in the Kubernetes environment
		t.Run("DaprHealth", func(t *testing.T) {
			healthy, err := daprClient.HealthCheck(ctx)
			require.NoError(t, err, "Failed to check Dapr health")
			assert.True(t, healthy, "Dapr should be healthy")
		})

		t.Run("JWKSHealth", func(t *testing.T) {
			healthy, err := jwtHelper.HealthCheck(ctx)
			require.NoError(t, err, "Failed to check JWKS health")
			assert.True(t, healthy, "JWKS Mock API should be healthy")
		})
	})
}

// TestKubernetesActorPlacement tests that actors are properly distributed across nodes
func TestKubernetesActorPlacement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Kubernetes integration test in short mode")
	}

	// Skip if not running against Kubernetes
	if os.Getenv("KUBERNETES_TEST") != "true" {
		t.Skip("Skipping Kubernetes test (set KUBERNETES_TEST=true to enable)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	daprEndpoint := os.Getenv("DAPR_HTTP_ENDPOINT")
	if daprEndpoint == "" {
		daprEndpoint = "http://localhost:3500"
	}

	jwksGenerateURL := os.Getenv("JWKS_GENERATE_URL")
	if jwksGenerateURL == "" {
		jwksGenerateURL = "http://localhost:3000/generate-token"
	}

	daprClient := NewDaprClient(daprEndpoint)
	jwtHelper, err := NewJWTHelper(jwksGenerateURL)
	require.NoError(t, err, "Failed to create JWT helper")

	token, err := jwtHelper.GenerateToken("placement-test-user", []string{"user"})
	require.NoError(t, err, "Failed to generate JWT token")

	// Create many actors to test placement across nodes
	numActors := 20
	actorType := "CounterActor"
	actorPrefix := "placement-test"

	t.Run("CreateMultipleActors", func(t *testing.T) {
		for i := 0; i < numActors; i++ {
			actorID := fmt.Sprintf("%s-%d", actorPrefix, i)
			
			// Set each actor to a unique value
			setReq := map[string]interface{}{"value": i}
			err := daprClient.InvokeActorMethod(ctx, actorType, actorID, "set", setReq, token)
			require.NoError(t, err, "Failed to set value for actor %s", actorID)
			
			// Small delay to allow for placement
			time.Sleep(100 * time.Millisecond)
		}
	})

	t.Run("VerifyActorPersistence", func(t *testing.T) {
		// Verify that all actors maintain their state
		for i := 0; i < numActors; i++ {
			actorID := fmt.Sprintf("%s-%d", actorPrefix, i)
			
			var result map[string]interface{}
			err := daprClient.InvokeActorMethodWithResult(ctx, actorType, actorID, "get", nil, &result, token)
			require.NoError(t, err, "Failed to get value for actor %s", actorID)
			
			value, ok := result["value"].(float64)
			require.True(t, ok, "Value should be a number for actor %s", actorID)
			assert.Equal(t, float64(i), value, "Actor %s should have its original value", actorID)
		}
	})
}