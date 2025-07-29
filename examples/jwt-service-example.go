package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

// jwtAwareCounterHandler demonstrates how to access JWT information at the service level
// and use it with JWT-aware actor methods
func jwtAwareCounterHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	// This is where you would extract JWT information from headers
	// In the current JWT gateway implementation, these headers are available:
	// - X-JWT-Subject: The user identifier from the JWT token
	// - X-JWT-Role: The user role from the JWT token
	
	// Note: In a real implementation, you would access these from the HTTP request context
	// For demonstration purposes, we'll show the concept
	
	response := map[string]interface{}{
		"message": "JWT-aware Counter Service",
		"available_headers": map[string]string{
			"X-JWT-Subject": "User identifier from JWT token (e.g., 'user123')",
			"X-JWT-Role":    "User role from JWT token (e.g., 'admin', 'user')",
		},
		"actor_methods": []string{
			"GetWithOwnership - Get counter value with ownership check",
			"SetWithOwnership - Set counter value and establish/check ownership",
			"TransferOwnership - Transfer counter ownership (admin only)",
		},
		"example_usage": map[string]interface{}{
			"note": "JWT information from headers would be extracted and passed to actor methods",
			"flow": []string{
				"1. Extract X-JWT-Subject and X-JWT-Role from request headers",
				"2. Create JWTContext with this information",
				"3. Call actor methods with JWTContext parameter",
				"4. Actor enforces authorization based on JWT claims",
			},
		},
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

// This function shows how JWT headers would be extracted in a real service
func extractJWTFromHeaders(req *http.Request) counter.JWTContext {
	return counter.JWTContext{
		Subject: req.Header.Get("X-JWT-Subject"),
		Role:    req.Header.Get("X-JWT-Role"),
	}
}

// exampleActorInvocation shows how to call JWT-aware actor methods
func exampleActorInvocation() {
	// This is pseudo-code showing how you would use the JWT-aware counter
	/*
	
	// Extract JWT context from headers
	jwtCtx := extractJWTFromHeaders(httpRequest)
	
	// Get actor client
	client := dapr.NewClient()
	actorID := "counter-1"
	
	// Call JWT-aware methods
	result, err := client.InvokeActor(ctx, &dapr.InvokeActorRequest{
		ActorType: counter.ActorTypeCounter,
		ActorID:   actorID,
		Method:    "GetWithOwnership",
		Data:      jwtCtx,
	})
	
	*/
	
	log.Println("This is example pseudo-code for JWT-aware actor invocation")
}

func main() {
	log.Println("JWT Service Example - Shows how JWT information flows from service to actors")
	
	s := daprd.NewService(":8081")
	
	// Register the JWT-aware handler
	s.AddServiceInvocationHandler("jwt-counter-info", jwtAwareCounterHandler)
	
	log.Println("Service registered. This demonstrates JWT information flow concepts.")
	log.Fatal(s.Start())
}