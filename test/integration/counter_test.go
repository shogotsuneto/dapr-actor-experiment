package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/counteractor"
)

func TestCounterActor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup Dapr client - assumes services are already running
	daprClient := NewDaprClient(GetDaprEndpoint())

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running. Start with: docker compose -f test/integration/docker-compose.test.yml up -d")

	t.Run("TestCounterActorBasicOperations", func(t *testing.T) {
		testCounterActorBasicOperations(t, daprClient)
	})

	t.Run("TestCounterActorStateIsolation", func(t *testing.T) {
		testCounterActorStateIsolation(t, daprClient)
	})

	t.Run("TestCounterActorMultipleInstances", func(t *testing.T) {
		testCounterActorMultipleInstances(t, daprClient)
	})
}

func testCounterActorBasicOperations(t *testing.T, client *DaprClient) {
	ctx := context.Background()
	actorID := "counter-test-basic"

	// Test 1: Get initial value (should be 0)
	var initialState counteractor.CounterState
	err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Get",
	}, &initialState)
	require.NoError(t, err)
	assert.Equal(t, int32(0), initialState.Value, "Initial counter value should be 0")

	// Test 2: Increment counter
	var incrementedState counteractor.CounterState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Increment",
	}, &incrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(1), incrementedState.Value, "Counter should be 1 after increment")

	// Test 3: Increment again
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Increment",
	}, &incrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(2), incrementedState.Value, "Counter should be 2 after second increment")

	// Test 4: Set to specific value
	var setState counteractor.CounterState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Set",
		Data:      counteractor.SetValueRequest{Value: int32(10)},
	}, &setState)
	require.NoError(t, err)
	assert.Equal(t, int32(10), setState.Value, "Counter should be 10 after set")

	// Test 5: Decrement
	var decrementedState counteractor.CounterState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Decrement",
	}, &decrementedState)
	require.NoError(t, err)
	assert.Equal(t, int32(9), decrementedState.Value, "Counter should be 9 after decrement")

	// Test 6: Verify final state persistence
	var finalState counteractor.CounterState
	err = client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
		ActorType: "CounterActor",
		ActorID:   actorID,
		Method:    "Get",
	}, &finalState)
	require.NoError(t, err)
	assert.Equal(t, int32(9), finalState.Value, "Final counter value should be 9")
}

func testCounterActorStateIsolation(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test that different actor instances maintain separate state
	actors := []string{"counter-isolation-1", "counter-isolation-2", "counter-isolation-3"}
	expectedValues := []int32{5, 10, 15}

	// Set different values for each actor
	for i, actorID := range actors {
		var state counteractor.CounterState
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Set",
			Data:      counteractor.SetValueRequest{Value: expectedValues[i]},
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, expectedValues[i], state.Value, "Counter should be set to expected value")
	}

	// Verify that each actor maintained its own state
	for i, actorID := range actors {
		var state counteractor.CounterState
		err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
			ActorType: "CounterActor",
			ActorID:   actorID,
			Method:    "Get",
		}, &state)
		require.NoError(t, err)
		assert.Equal(t, expectedValues[i], state.Value, "Actor %s should maintain its own state", actorID)
	}
}

func testCounterActorMultipleInstances(t *testing.T, client *DaprClient) {
	ctx := context.Background()

	// Test scenario similar to the shell script test-counter-actor.sh
	testCases := []struct {
		actorID       string
		operations    []string
		expectedFinal int32
	}{
		{
			actorID:       "counter-001",
			operations:    []string{"Increment", "Increment", "Set:10"},
			expectedFinal: 10,
		},
		{
			actorID:       "counter-002", 
			operations:    []string{"Increment", "Increment", "Increment"},
			expectedFinal: 3,
		},
		{
			actorID:       "counter-003",
			operations:    []string{"Set:25", "Decrement"},
			expectedFinal: 24,
		},
	}

	for _, tc := range testCases {
		t.Run("ActorInstance_"+tc.actorID, func(t *testing.T) {
			// Execute operations
			for _, op := range tc.operations {
				var state counteractor.CounterState
				
				if op == "Increment" {
					err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "CounterActor",
						ActorID:   tc.actorID,
						Method:    "Increment",
					}, &state)
					require.NoError(t, err)
				} else if op == "Decrement" {
					err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "CounterActor",
						ActorID:   tc.actorID,
						Method:    "Decrement",
					}, &state)
					require.NoError(t, err)
				} else if op == "Set:10" {
					err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "CounterActor",
						ActorID:   tc.actorID,
						Method:    "Set",
						Data:      counteractor.SetValueRequest{Value: int32(10)},
					}, &state)
					require.NoError(t, err)
				} else if op == "Set:25" {
					err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
						ActorType: "CounterActor",
						ActorID:   tc.actorID,
						Method:    "Set",
						Data:      counteractor.SetValueRequest{Value: int32(25)},
					}, &state)
					require.NoError(t, err)
				}
			}

			// Verify final state
			var finalState counteractor.CounterState
			err := client.InvokeActorMethodWithResponse(ctx, ActorMethodRequest{
				ActorType: "CounterActor",
				ActorID:   tc.actorID,
				Method:    "Get",
			}, &finalState)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFinal, finalState.Value, "Final value for %s should be %d", tc.actorID, tc.expectedFinal)
		})
	}
}