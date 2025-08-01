package integration

import (
"context"
"testing"
"time"

"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"

"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
)

func TestSimpleJWTReuse(t *testing.T) {
if testing.Short() {
t.Skip("Skipping integration test in short mode")
}

// Setup Dapr client
daprClient := NewDaprClient(GetDaprEndpoint())
require.NoError(t, daprClient.CheckHealth())

// Generate ONE JWT token for all operations
userToken, err := generateTestToken("shared-user", "shared-user", "shared@example.com", []string{"user"}, 1*time.Hour)
require.NoError(t, err, "Failed to generate JWT token for testing")

ctx := context.Background()

// Test multiple actors with the same token
actorIDs := []string{"reuse-test-1", "reuse-test-2", "reuse-test-3"}

for _, actorID := range actorIDs {
t.Run("Actor_"+actorID, func(t *testing.T) {
// Create account
var createResult interface{}
_, err = daprClient.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
ActorType: "BankAccount",
ActorID:   actorID,
Method:    "CreateAccount",
Data: bankaccount.CreateAccountRequest{
OwnerName:      "Shared User " + actorID,
InitialDeposit: 100.0,
},
}, userToken, &createResult)
require.NoError(t, err, "Create account should succeed for %s", actorID)

// Get balance
var balance bankaccount.BankAccountState
_, err = daprClient.InvokeActorMethodWithJWT(ctx, ActorMethodRequest{
ActorType: "BankAccount",
ActorID:   actorID,
Method:    "GetBalance",
}, userToken, &balance)
require.NoError(t, err, "Get balance should succeed for %s", actorID)
assert.Equal(t, 100.0, balance.Balance, "Balance should be 100 for %s", actorID)
})
}
}
