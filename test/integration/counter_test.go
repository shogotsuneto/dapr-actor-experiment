package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

func TestCounter(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient(GetDaprEndpoint())

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running. Start with: docker compose -f test/integration/docker-compose.test.yml up -d")

	t.Run("TestCounterBasicOperations", func(t *testing.T) {
		testCounterBasicOperations(t, daprClient)
	})

	t.Run("TestCounterStateIsolation", func(t *testing.T) {
		testCounterStateIsolation(t, daprClient)
	})

	t.Run("TestCounterMultipleInstances", func(t *testing.T) {
		testCounterMultipleInstances(t, daprClient)
	})
}

func testCounterBasicOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := "counter-test-basic-" + fmt.Sprintf("%d", time.Now().UnixNano()%10000) // Unique ID

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test 1: Get initial value (should be 0) 
	var initialState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Get",
	}, userToken, &initialState)
	require.NoError(t, err)
	assert.Equal(t, int32(0), initialState.Value, "Initial counter value should be 0")

	// Test 2: Increment counter
	var incrementedState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Increment",
	}, userToken, &incrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(1), incrementedState.Value, "Counter should be 1 after increment")

	// Test 3: Increment again
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Increment",
	}, userToken, &incrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(2), incrementedState.Value, "Counter should be 2 after second increment")

	// Test 4: Set to specific value
	var setState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Set",
		Data:      counter.SetValueRequest{Value: int32(10)},
	}, userToken, &setState)
	require.NoError(t, err)
	assert.Equal(t, int32(10), setState.Value, "Counter should be 10 after set")

	// Test 5: Decrement
	var decrementedState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Decrement",
	}, userToken, &decrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(9), decrementedState.Value, "Counter should be 9 after decrement")

	// Test 6: Verify final state persistence
	var finalState counter.CounterState
	_, err = client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
		ActorType: "Counter",
		ActorID:   actorID,
		Method:    "Get",
	}, userToken, &finalState)
	require.NoError(t, err)
	assert.Equal(t, int32(9), finalState.Value, "Final counter value should be 9")
}

func testCounterStateIsolation(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test that different actor instances maintain separate state
	actors := []string{
		fmt.Sprintf("counter-isolation-1-%d", time.Now().UnixNano()%10000),
		fmt.Sprintf("counter-isolation-2-%d", time.Now().UnixNano()%10000),
		fmt.Sprintf("counter-isolation-3-%d", time.Now().UnixNano()%10000),
	}
	expectedValues := []int32{5, 10, 15}

	// Set different values for each actor
	for i, actorID := range actors {
		var state counter.CounterState
		_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   actorID,
			Method:    "Set",
			Data:      counter.SetValueRequest{Value: expectedValues[i]},
		}, userToken, &state)
		require.NoError(t, err)
		assert.Equal(t, expectedValues[i], state.Value, "Counter should be set to expected value")
	}

	// Verify that each actor maintained its own state
	for i, actorID := range actors {
		var state counter.CounterState
		_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
			ActorType: "Counter",
			ActorID:   actorID,
			Method:    "Get",
		}, userToken, &state)
		require.NoError(t, err)
		assert.Equal(t, expectedValues[i], state.Value, "Actor %s should maintain its own state", actorID)
	}
}

func testCounterMultipleInstances(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Generate JWT token for authenticated operations
	userToken, err := generateTestToken("test-user", "test-user", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token for testing")

	// Test scenario similar to the shell script test-counter-actor.sh
	testCases := []struct {
		actorID       string
		operations    []string
		expectedFinal int32
	}{
		{
			actorID:       fmt.Sprintf("counter-001-%d", time.Now().UnixNano()%10000),
			operations:    []string{"Increment", "Increment", "Set:10"},
			expectedFinal: 10,
		},
		{
			actorID:       fmt.Sprintf("counter-002-%d", time.Now().UnixNano()%10000), 
			operations:    []string{"Increment", "Increment", "Increment"},
			expectedFinal: 3,
		},
		{
			actorID:       fmt.Sprintf("counter-003-%d", time.Now().UnixNano()%10000),
			operations:    []string{"Set:25", "Decrement"},
			expectedFinal: 24,
		},
	}

	for _, tc := range testCases {
		t.Run("ActorInstance_"+tc.actorID, func(t *testing.T) {
			// Execute operations
			for _, op := range tc.operations {
				var state counter.CounterState
				
				if op == "Increment" {
					_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "Counter",
						ActorID:   tc.actorID,
						Method:    "Increment",
					}, userToken, &state)
					require.NoError(t, err)
				} else if op == "Decrement" {
					_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "Counter",
						ActorID:   tc.actorID,
						Method:    "Decrement",
					}, userToken, &state)
					require.NoError(t, err)
				} else if op == "Set:10" {
					_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "Counter",
						ActorID:   tc.actorID,
						Method:    "Set",
						Data:      counter.SetValueRequest{Value: int32(10)},
					}, userToken, &state)
					require.NoError(t, err)
				} else if op == "Set:25" {
					_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
						ActorType: "Counter",
						ActorID:   tc.actorID,
						Method:    "Set",
						Data:      counter.SetValueRequest{Value: int32(25)},
					}, userToken, &state)
					require.NoError(t, err)
				}
			}

			// Verify final state
			var finalState counter.CounterState
			_, err := client.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
				ActorType: "Counter",
				ActorID:   tc.actorID,
				Method:    "Get",
			}, userToken, &finalState)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFinal, finalState.Value, "Final value for %s should be %d", tc.actorID, tc.expectedFinal)
		})
	}
}