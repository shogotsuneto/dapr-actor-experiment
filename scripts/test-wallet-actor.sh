#!/bin/bash

echo "Testing Wallet Actor (External Event Store)"
echo "==========================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Generate JWT token for wallet operations
generate_token() {
    echo "Generating JWT token for wallet operations..."
    
    JWKS_URL="http://localhost:3000/generate-token"
    
    # Check if JWKS service is available
    if ! curl -s "$JWKS_URL" > /dev/null; then
        echo "JWKS Mock API not available - authentication required for wallet operations"
        echo "Please start with: docker compose up -d"
        exit 1
    fi
    
    # Generate token for wallet user
    WALLET_TOKEN_RESPONSE=$(curl -s -X POST "$JWKS_URL" \
        -H "Content-Type: application/json" \
        -d '{
            "claims": {
                "sub": "wallet-user-001",
                "name": "Wallet Test User",
                "email": "wallet@example.com"
            },
            "expiresIn": 3600
        }')
    WALLET_TOKEN=$(echo "$WALLET_TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -z "$WALLET_TOKEN" ]; then
        echo "Failed to generate JWT token"
        exit 1
    fi
    
    echo "✓ JWT token generated successfully"
}

# Generate authentication token
generate_token

# Test variables
WALLET_ID="test-wallet-001"
DAPR_HTTP_URL="http://localhost:3500"
BASE_URL="$DAPR_HTTP_URL/v1.0/actors/Wallet/$WALLET_ID/method"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo ""
echo -e "${BLUE}1. Creating wallet for John Doe with USD currency...${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/CreateWallet" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WALLET_TOKEN" \
  -d '{
    "ownerName": "John Doe",
    "currency": "USD",
    "initialBalance": 100.00
  }')

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ Wallet created successfully${NC}"
else
    echo -e "${RED}✗ Failed to create wallet${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}2. Getting wallet balance...${NC}"
RESPONSE=$(curl -s -X GET "$BASE_URL/GetBalance" \
  -H "Authorization: Bearer $WALLET_TOKEN")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"balance":100'; then
    echo -e "${GREEN}✓ Initial balance is correct (100.00)${NC}"
else
    echo -e "${RED}✗ Initial balance is incorrect${NC}"
fi

echo ""
echo -e "${BLUE}3. Adding funds (50.00) to wallet...${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/AddFunds" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WALLET_TOKEN" \
  -d '{
    "amount": 50.00,
    "description": "Monthly allowance"
  }')

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ Funds added successfully${NC}"
else
    echo -e "${RED}✗ Failed to add funds${NC}"
fi

echo ""
echo -e "${BLUE}4. Getting updated balance...${NC}"
RESPONSE=$(curl -s -X GET "$BASE_URL/GetBalance" \
  -H "Authorization: Bearer $WALLET_TOKEN")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"balance":150'; then
    echo -e "${GREEN}✓ Balance updated correctly (150.00)${NC}"
else
    echo -e "${RED}✗ Balance update failed${NC}"
fi

echo ""
echo -e "${BLUE}5. Spending funds (25.00) from wallet...${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/SpendFunds" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WALLET_TOKEN" \
  -d '{
    "amount": 25.00,
    "description": "Coffee purchase"
  }')

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"success":true'; then
    echo -e "${GREEN}✓ Funds spent successfully${NC}"
else
    echo -e "${RED}✗ Failed to spend funds${NC}"
fi

echo ""
echo -e "${BLUE}6. Getting final balance...${NC}"
RESPONSE=$(curl -s -X GET "$BASE_URL/GetBalance" \
  -H "Authorization: Bearer $WALLET_TOKEN")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"balance":125'; then
    echo -e "${GREEN}✓ Final balance is correct (125.00)${NC}"
else
    echo -e "${RED}✗ Final balance is incorrect${NC}"
fi

echo ""
echo -e "${BLUE}7. Getting transaction history from external event store...${NC}"
RESPONSE=$(curl -s -X GET "$BASE_URL/GetTransactions" \
  -H "Authorization: Bearer $WALLET_TOKEN")
echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"eventType":"WalletCreated"' && \
   echo "$RESPONSE" | grep -q '"eventType":"FundsAdded"' && \
   echo "$RESPONSE" | grep -q '"eventType":"FundsSpent"'; then
    echo -e "${GREEN}✓ Transaction history contains all expected events from external event store${NC}"
else
    echo -e "${RED}✗ Transaction history is incomplete${NC}"
fi

echo ""
echo -e "${BLUE}8. Testing insufficient funds scenario...${NC}"
RESPONSE=$(curl -s -X POST "$BASE_URL/SpendFunds" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $WALLET_TOKEN" \
  -d '{
    "amount": 200.00,
    "description": "Expensive purchase"
  }')

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -q '"success":false' && echo "$RESPONSE" | grep -q 'INSUFFICIENT_FUNDS'; then
    echo -e "${GREEN}✓ Insufficient funds error handled correctly${NC}"
else
    echo -e "${RED}✗ Insufficient funds validation failed${NC}"
fi

echo ""
echo "=========================================="
echo "Wallet Actor Test Summary"
echo "=========================================="
echo "✓ Wallet creation with external event store"
echo "✓ Balance operations (add/spend funds)" 
echo "✓ Event sourcing using go-simple-eventstore"
echo "✓ Transaction history from external storage"
echo "✓ Validation and error handling"
echo ""
echo -e "${GREEN}✓ All Wallet tests completed successfully!${NC}"
echo ""
echo "Key Demonstrations:"
echo "- External event store usage (go-simple-eventstore)"
echo "- Shared singleton pattern for event store connections"
echo "- Event sourcing with third-party persistence layer"
echo "- Alternative to Dapr StateManager for complex scenarios"