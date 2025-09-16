package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shogotsuneto/dapr-actor-experiment/internal/bankaccount"
)

// TransactionRow represents a projected transaction from the transactions table
type TransactionRow struct {
	ID                     int       `json:"id"`
	AccountID              string    `json:"account_id"`
	OwnerID                string    `json:"owner_id"`
	TransactionType        string    `json:"transaction_type"`
	Amount                 string    `json:"amount"`
	Description            string    `json:"description"`
	TransactionTimestamp   time.Time `json:"transaction_timestamp"`
	EventVersion           int       `json:"event_version"`
	CreatedAt              time.Time `json:"created_at"`
}

// QueryRowsResponse represents the response from query endpoints
type QueryRowsResponse struct {
	Rows []TransactionRow `json:"rows"`
}

// BalanceRow represents a balance calculation result
type BalanceRow struct {
	AccountID string `json:"account_id"`
	Balance   string `json:"balance"`
}

// QueryBalanceResponse represents the response from balance queries
type QueryBalanceResponse struct {
	Rows []BalanceRow `json:"rows"`
}

// SummaryRow represents a transaction summary result
type SummaryRow struct {
	TransactionType string `json:"transaction_type"`
	Count           int    `json:"count"`
	TotalAmount     string `json:"total_amount"`
}

// QuerySummaryResponse represents the response from summary queries
type QuerySummaryResponse struct {
	Rows []SummaryRow `json:"rows"`
}

func TestProjection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup clients
	daprClient := NewDaprClient(GetDaprEndpoint())
	queryClient := NewQueryClient(GetQueryServerEndpoint())

	// Verify services are available
	require.NoError(t, daprClient.CheckHealth(), "Dapr services must be running")
	require.NoError(t, queryClient.CheckHealth(), "Query server must be running")

	t.Run("TestCQRSProjectionFlow", func(t *testing.T) {
		testCQRSProjectionFlow(t, daprClient, queryClient)
	})

	t.Run("TestProjectionUserIsolation", func(t *testing.T) {
		testProjectionUserIsolation(t, daprClient, queryClient)
	})

	t.Run("TestProjectionUnauthorizedAccess", func(t *testing.T) {
		testProjectionUnauthorizedAccess(t, queryClient)
	})
}

func testCQRSProjectionFlow(t *testing.T, daprClient *DaprClient, queryClient *QueryClient) {
	ctx := context.Background()

	// Generate unique user ID to avoid conflicts
	userID := "test-user-" + time.Now().Format("20060102-150405")
	// Account ID must match the user ID due to authorization
	accountID := userID

	// Generate JWT token for authentication
	token, err := generateTestToken(userID, "Test User", "test@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate JWT token")

	// Step 1: Create account via BankAccount actor (Command side)
	createReq := ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   accountID,
		Method:    "CreateAccount",
		Data: map[string]interface{}{
			"initialDeposit": 1000.0,
		},
	}

	var createResp bankaccount.BankAccountState
	_, err = daprClient.InvokeActorMethodWithJWT(ctx, createReq, token, &createResp)
	require.NoError(t, err, "Failed to create account")
	t.Logf("Create response: Success=%v, Data=%+v, Error=%+v", createResp.Success, createResp.Data, createResp.Error)
	if !createResp.Success && createResp.Error != nil {
		t.Fatalf("Account creation failed: %s", createResp.Error.Message)
	}
	require.True(t, createResp.Success, "Account creation should succeed")
	require.Equal(t, 1000.0, createResp.Data.Balance, "Initial balance should be 1000")

	// Step 2: Make some transactions
	depositReq := ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   accountID,
		Method:    "Deposit",
		Data: map[string]interface{}{
			"amount":      500.0,
			"description": "Test deposit for projection",
		},
	}

	var depositResp bankaccount.BankAccountState
	_, err = daprClient.InvokeActorMethodWithJWT(ctx, depositReq, token, &depositResp)
	require.NoError(t, err, "Failed to make deposit")
	require.True(t, depositResp.Success, "Deposit should succeed")
	require.Equal(t, 1500.0, depositResp.Data.Balance, "Balance should be 1500 after deposit")

	withdrawReq := ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   accountID,
		Method:    "Withdraw",
		Data: map[string]interface{}{
			"amount":      200.0,
			"description": "Test withdrawal for projection",
		},
	}

	var withdrawResp bankaccount.BankAccountState
	_, err = daprClient.InvokeActorMethodWithJWT(ctx, withdrawReq, token, &withdrawResp)
	require.NoError(t, err, "Failed to make withdrawal")
	require.True(t, withdrawResp.Success, "Withdrawal should succeed")
	require.Equal(t, 1300.0, withdrawResp.Data.Balance, "Balance should be 1300 after withdrawal")

	// Step 3: Wait for projector to process events
	t.Log("Waiting for projector to process events...")
	time.Sleep(20 * time.Second) // Give projector time to process

	// Step 4: Query projected data (Query side)
	
	// Test my_transactions query
	transactionsReq := QueryRequest{
		QueryName: "my_transactions",
		Params:    map[string]interface{}{"limit": 10},
		JWTToken:  token,
	}

	var transactionsResp QueryRowsResponse
	err = queryClient.ExecuteQueryWithResponse(ctx, transactionsReq, &transactionsResp)
	require.NoError(t, err, "Failed to query transactions")
	
	// Verify we have the expected transactions
	require.GreaterOrEqual(t, len(transactionsResp.Rows), 3, "Should have at least 3 transactions (create, deposit, withdraw)")
	
	// Find our test account transactions
	accountTransactions := make([]TransactionRow, 0)
	for _, row := range transactionsResp.Rows {
		if row.AccountID == accountID {
			accountTransactions = append(accountTransactions, row)
		}
	}
	require.Equal(t, 3, len(accountTransactions), "Should have exactly 3 transactions for test account")

	// Verify transaction types and amounts
	expectedTransactions := map[string]string{
		"account_created": "1000.00",
		"deposit":        "500.00", 
		"withdrawal":     "200.00",
	}

	for _, tx := range accountTransactions {
		expectedAmount, exists := expectedTransactions[tx.TransactionType]
		require.True(t, exists, "Unexpected transaction type: %s", tx.TransactionType)
		assert.Equal(t, expectedAmount, tx.Amount, "Transaction amount mismatch for type %s", tx.TransactionType)
		assert.Equal(t, userID, tx.OwnerID, "Owner ID should match JWT sub claim")
	}

	// Test my_account_balance query
	balanceReq := QueryRequest{
		QueryName: "my_account_balance",
		Params:    map[string]interface{}{},
		JWTToken:  token,
	}

	var balanceResp QueryBalanceResponse
	err = queryClient.ExecuteQueryWithResponse(ctx, balanceReq, &balanceResp)
	require.NoError(t, err, "Failed to query account balance")

	// Find our test account balance
	var testAccountBalance *BalanceRow
	for _, row := range balanceResp.Rows {
		if row.AccountID == accountID {
			testAccountBalance = &row
			break
		}
	}
	require.NotNil(t, testAccountBalance, "Should find balance for test account")
	assert.Equal(t, "1300.00", testAccountBalance.Balance, "Projected balance should match actor balance")

	// Test my_transaction_summary query
	summaryReq := QueryRequest{
		QueryName: "my_transaction_summary",
		Params:    map[string]interface{}{},
		JWTToken:  token,
	}

	var summaryResp QuerySummaryResponse
	err = queryClient.ExecuteQueryWithResponse(ctx, summaryReq, &summaryResp)
	require.NoError(t, err, "Failed to query transaction summary")

	// Verify summary includes our transaction types
	summaryMap := make(map[string]SummaryRow)
	for _, row := range summaryResp.Rows {
		summaryMap[row.TransactionType] = row
	}

	assert.Contains(t, summaryMap, "account_created", "Summary should include account_created transactions")
	assert.Contains(t, summaryMap, "deposit", "Summary should include deposit transactions")
	assert.Contains(t, summaryMap, "withdrawal", "Summary should include withdrawal transactions")

	t.Log("✅ CQRS projection flow test completed successfully")
}

func testProjectionUserIsolation(t *testing.T, daprClient *DaprClient, queryClient *QueryClient) {
	ctx := context.Background()

	// Create two different users - account ID must match user ID
	userA := "user-a-" + time.Now().Format("20060102-150405")
	userB := "user-b-" + time.Now().Format("20060102-150405")
	accountA := userA // Account ID must match user ID
	accountB := userB // Account ID must match user ID

	tokenA, err := generateTestToken(userA, "User A", "userA@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate token for user A")

	tokenB, err := generateTestToken(userB, "User B", "userB@example.com", []string{"user"}, 1*time.Hour)
	require.NoError(t, err, "Failed to generate token for user B")

	// Create accounts for both users
	createReqA := ActorMethodRequest{
		ActorType: "BankAccount",
		ActorID:   accountA,
		Method:    "CreateAccount",
		Data:      map[string]interface{}{"initialDeposit": 1000.0},
	}

	createReqB := ActorMethodRequest{
		ActorType: "BankAccount", 
		ActorID:   accountB,
		Method:    "CreateAccount",
		Data:      map[string]interface{}{"initialDeposit": 2000.0},
	}

	var respA, respB bankaccount.BankAccountState
	_, err = daprClient.InvokeActorMethodWithJWT(ctx, createReqA, tokenA, &respA)
	require.NoError(t, err, "Failed to create account A")

	_, err = daprClient.InvokeActorMethodWithJWT(ctx, createReqB, tokenB, &respB)
	require.NoError(t, err, "Failed to create account B")

	// Wait for projection
	time.Sleep(15 * time.Second)

	// User A should only see their own transactions
	transactionsReqA := QueryRequest{
		QueryName: "my_transactions",
		Params:    map[string]interface{}{"limit": 10},
		JWTToken:  tokenA,
	}

	var transactionsRespA QueryRowsResponse
	err = queryClient.ExecuteQueryWithResponse(ctx, transactionsReqA, &transactionsRespA)
	require.NoError(t, err, "Failed to query transactions for user A")

	// Verify user A only sees their own account
	for _, tx := range transactionsRespA.Rows {
		assert.Equal(t, userA, tx.OwnerID, "User A should only see their own transactions")
	}

	// User B should only see their own transactions
	transactionsReqB := QueryRequest{
		QueryName: "my_transactions",
		Params:    map[string]interface{}{"limit": 10},
		JWTToken:  tokenB,
	}

	var transactionsRespB QueryRowsResponse
	err = queryClient.ExecuteQueryWithResponse(ctx, transactionsReqB, &transactionsRespB)
	require.NoError(t, err, "Failed to query transactions for user B")

	// Verify user B only sees their own account
	for _, tx := range transactionsRespB.Rows {
		assert.Equal(t, userB, tx.OwnerID, "User B should only see their own transactions")
	}

	t.Log("✅ User isolation test completed successfully")
}

func testProjectionUnauthorizedAccess(t *testing.T, queryClient *QueryClient) {
	ctx := context.Background()

	// Test query without JWT token
	unauthorizedReq := QueryRequest{
		QueryName: "my_transactions",
		Params:    map[string]interface{}{"limit": 10},
		JWTToken:  "", // No token
	}

	resp, err := queryClient.ExecuteQuery(ctx, unauthorizedReq)
	require.NoError(t, err, "HTTP request should succeed")
	assert.Equal(t, 401, resp.StatusCode, "Should return 401 Unauthorized")

	// Test query with invalid JWT token
	invalidTokenReq := QueryRequest{
		QueryName: "my_transactions",
		Params:    map[string]interface{}{"limit": 10},
		JWTToken:  "invalid.jwt.token",
	}

	resp, err = queryClient.ExecuteQuery(ctx, invalidTokenReq)
	require.NoError(t, err, "HTTP request should succeed")
	assert.Equal(t, 401, resp.StatusCode, "Should return 401 Unauthorized for invalid token")

	t.Log("✅ Unauthorized access test completed successfully")
}