package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
	"github.com/shogotsuneto/dapr-actor-experiment/internal/counter"
)

// CounterWithJWT demonstrates an enhanced counter that can access JWT information
type CounterWithJWT struct {
	counter.Counter
}

// GetWithJWT shows how to access JWT information from within an actor method
func (c *CounterWithJWT) GetWithJWT(ctx context.Context) (map[string]interface{}, error) {
	// Get the current counter state
	state, err := c.Get(ctx)
	if err != nil {
		return nil, err
	}
	
	// In the current implementation, JWT information is passed as HTTP headers
	// However, these headers are not directly accessible in actor methods since
	// Dapr actors abstract away the HTTP layer.
	
	// JWT information would need to be extracted at the service level and passed
	// to actors through the invocation context or as method parameters.
	
	response := map[string]interface{}{
		"value": state.Value,
		"jwt_info": map[string]interface{}{
			"note": "JWT headers (X-JWT-Subject, X-JWT-Role) are added by the JWT gateway",
			"availability": "Available at Dapr sidecar level, not directly in actor methods",
			"solution": "Use service-level middleware or pass JWT claims as method parameters",
		},
	}
	
	return response, nil
}

// demonstrateServiceLevelJWT shows how JWT information could be accessed at the service level
func demonstrateServiceLevelJWT(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	// At the service level, you have access to the raw HTTP request
	// JWT information would be available in headers if passed through by the JWT gateway
	
	response := map[string]interface{}{
		"message": "This is a service-level endpoint where JWT headers would be accessible",
		"headers_available": []string{
			"X-JWT-Subject (user identifier)",
			"X-JWT-Role (user role/permissions)",
		},
		"use_cases": []string{
			"Check resource ownership based on subject",
			"Authorize actions based on role",
			"Log user actions for audit",
		},
	}
	
	data, _ := json.Marshal(response)
	out = &common.Content{
		Data:        data,
		ContentType: "application/json",
	}
	return
}

func main() {
	fmt.Println("JWT Actor Demo - This demonstrates how JWT information flows to actors")
	fmt.Println("Note: This is an example file to show concepts, not part of the main application")
	
	// In a real implementation, you would:
	// 1. Access JWT headers at the service level (like in demonstrateServiceLevelJWT)
	// 2. Extract user/role information
	// 3. Pass this information to actor methods as needed
	// 4. Implement authorization logic within actors
	
	os.Exit(0)
}