package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/dapr/go-sdk/client"
	
	counteractor "github.com/shogotsuneto/dapr-actor-experiment/internal/actor"
)

func main() {
	// Create Dapr client
	c, err := client.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Dapr client: %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	actorType := "CounterActor"
	actorID := "counter-1"

	log.Println("=== Dapr Actor Demo Client ===")
	log.Printf("Interacting with actor: %s/%s\n", actorType, actorID)

	// Test 1: Get initial value
	log.Println("\n1. Getting initial counter value...")
	response, err := c.InvokeActor(ctx, &client.InvokeActorRequest{
		ActorType: actorType,
		ActorID:   actorID,
		Method:    "get",
	})
	if err != nil {
		log.Fatalf("Failed to get counter value: %v", err)
	}

	var state counteractor.CounterState
	if err := json.Unmarshal(response.Data, &state); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	log.Printf("Initial value: %d", state.Value)

	// Test 2: Increment counter 5 times
	log.Println("\n2. Incrementing counter 5 times...")
	for i := 0; i < 5; i++ {
		response, err := c.InvokeActor(ctx, &client.InvokeActorRequest{
			ActorType: actorType,
			ActorID:   actorID,
			Method:    "increment",
		})
		if err != nil {
			log.Fatalf("Failed to increment counter: %v", err)
		}

		if err := json.Unmarshal(response.Data, &state); err != nil {
			log.Fatalf("Failed to unmarshal response: %v", err)
		}
		log.Printf("After increment %d: %d", i+1, state.Value)
		time.Sleep(500 * time.Millisecond)
	}

	// Test 3: Decrement counter 2 times
	log.Println("\n3. Decrementing counter 2 times...")
	for i := 0; i < 2; i++ {
		response, err := c.InvokeActor(ctx, &client.InvokeActorRequest{
			ActorType: actorType,
			ActorID:   actorID,
			Method:    "decrement",
		})
		if err != nil {
			log.Fatalf("Failed to decrement counter: %v", err)
		}

		if err := json.Unmarshal(response.Data, &state); err != nil {
			log.Fatalf("Failed to unmarshal response: %v", err)
		}
		log.Printf("After decrement %d: %d", i+1, state.Value)
		time.Sleep(500 * time.Millisecond)
	}

	// Test 4: Set counter to specific value
	log.Println("\n4. Setting counter to 100...")
	setValue := map[string]int{"value": 100}
	setData, _ := json.Marshal(setValue)

	response, err = c.InvokeActor(ctx, &client.InvokeActorRequest{
		ActorType: actorType,
		ActorID:   actorID,
		Method:    "set",
		Data:      setData,
	})
	if err != nil {
		log.Fatalf("Failed to set counter value: %v", err)
	}

	if err := json.Unmarshal(response.Data, &state); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	log.Printf("After setting to 100: %d", state.Value)

	// Test 5: Final value check
	log.Println("\n5. Getting final counter value...")
	response, err = c.InvokeActor(ctx, &client.InvokeActorRequest{
		ActorType: actorType,
		ActorID:   actorID,
		Method:    "get",
	})
	if err != nil {
		log.Fatalf("Failed to get final counter value: %v", err)
	}

	if err := json.Unmarshal(response.Data, &state); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	log.Printf("Final value: %d", state.Value)

	// Test 6: Test with different actor instance
	log.Println("\n6. Testing with different actor instance (counter-2)...")
	actorID2 := "counter-2"
	
	response, err = c.InvokeActor(ctx, &client.InvokeActorRequest{
		ActorType: actorType,
		ActorID:   actorID2,
		Method:    "get",
	})
	if err != nil {
		log.Fatalf("Failed to get counter-2 value: %v", err)
	}

	if err := json.Unmarshal(response.Data, &state); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	log.Printf("Counter-2 initial value: %d", state.Value)

	// Increment counter-2
	response, err = c.InvokeActor(ctx, &client.InvokeActorRequest{
		ActorType: actorType,
		ActorID:   actorID2,
		Method:    "increment",
	})
	if err != nil {
		log.Fatalf("Failed to increment counter-2: %v", err)
	}

	if err := json.Unmarshal(response.Data, &state); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}
	log.Printf("Counter-2 after increment: %d", state.Value)

	log.Println("\n=== Demo completed successfully! ===")
	log.Println("This demonstrates:")
	log.Println("- Actor state persistence")
	log.Println("- Method invocation")
	log.Println("- Multiple actor instances with independent state")
}