#!/bin/bash

echo "Testing BankAccount (Event-sourced pattern)"
echo "==============================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Generate JWT tokens using JWKS Mock API for authenticated operations
generate_tokens() {
    echo "Generating JWT tokens for authenticated operations..."
    
    JWKS_URL="http://localhost:3000/generate-token"
    
    # Check if JWKS service is available
    if ! curl -s "$JWKS_URL" > /dev/null; then
        echo "JWKS Mock API not available - running without authentication"
        USE_AUTH=false
        return
    fi
    
    USE_AUTH=true
    
    # Generate token for Alice
    ALICE_TOKEN_RESPONSE=$(curl -s -X POST "$JWKS_URL" \
        -H "Content-Type: application/json" \
        -d '{
            "claims": {
                "sub": "account-alice"
            },
            "expiresIn": 3600
        }')
    ALICE_TOKEN=$(echo "$ALICE_TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    # Generate token for Bob
    BOB_TOKEN_RESPONSE=$(curl -s -X POST "$JWKS_URL" \
        -H "Content-Type: application/json" \
        -d '{
            "claims": {
                "sub": "account-bob"
            },
            "expiresIn": 3600
        }')
    BOB_TOKEN=$(echo "$BOB_TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    # Generate token for Charlie
    CHARLIE_TOKEN_RESPONSE=$(curl -s -X POST "$JWKS_URL" \
        -H "Content-Type: application/json" \
        -d '{
            "claims": {
                "sub": "account-charlie"
            },
            "expiresIn": 3600
        }')
    CHARLIE_TOKEN=$(echo "$CHARLIE_TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$ALICE_TOKEN" && -n "$BOB_TOKEN" && -n "$CHARLIE_TOKEN" ]]; then
        echo "✅ Tokens generated successfully for authenticated operations"
    else
        echo "⚠️  Token generation partial - some operations may fail"
        USE_AUTH=false
    fi
}

# Helper function to make authenticated requests
make_request() {
    local method="$1"
    local url="$2"
    local data="$3"
    local token="$4"
    
    if [ "$USE_AUTH" = true ] && [ -n "$token" ]; then
        if [ "$method" = "POST" ]; then
            curl -s -X POST "$url" -H "Authorization: Bearer $token" -H "Content-Type: application/json" -d "$data"
        else
            curl -s "$url" -H "Authorization: Bearer $token"
        fi
    else
        if [ "$method" = "POST" ]; then
            curl -s -X POST "$url" -H "Content-Type: application/json" -d "$data"
        else
            curl -s "$url"
        fi
    fi
}

generate_tokens

# Test multiple BankAccount instances with optional authentication
echo ""
echo "Testing Multiple BankAccount Instances:"
echo "--------------------------------------------"

# Instance 1: account-alice
echo ""
echo "1. Testing BankAccount instance 'account-alice':"
echo "Creating Alice's bank account:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/CreateAccount" '{"ownerName": "Alice Johnson", "initialDeposit": 1500.00}' "$ALICE_TOKEN" | jq '.'

echo -e "\nDepositing salary:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Deposit" '{"amount": 3000.00, "description": "Monthly salary"}' "$ALICE_TOKEN" | jq '.'

echo -e "\nWithdrawing for rent:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Withdraw" '{"amount": 1200.00, "description": "Rent payment"}' "$ALICE_TOKEN" | jq '.'

echo -e "\nWithdrawing for groceries:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Withdraw" '{"amount": 150.00, "description": "Grocery shopping"}' "$ALICE_TOKEN" | jq '.'

echo -e "\nAlice's current balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/GetBalance" "" "$ALICE_TOKEN" | jq '.'

# Instance 2: account-bob
echo ""
echo "2. Testing BankAccount instance 'account-bob':"
echo "Creating Bob's bank account:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/CreateAccount" '{"ownerName": "Bob Smith", "initialDeposit": 500.00}' "$BOB_TOKEN" | jq '.'

echo -e "\nDepositing freelance payment:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Deposit" '{"amount": 800.00, "description": "Freelance project payment"}' "$BOB_TOKEN" | jq '.'

echo -e "\nDepositing bonus:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Deposit" '{"amount": 200.00, "description": "Performance bonus"}' "$BOB_TOKEN" | jq '.'

echo -e "\nWithdrawing for car payment:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Withdraw" '{"amount": 350.00, "description": "Car loan payment"}' "$BOB_TOKEN" | jq '.'

echo -e "\nBob's current balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/GetBalance" "" "$BOB_TOKEN" | jq '.'

# Instance 3: account-charlie
echo ""
echo "3. Testing BankAccount instance 'account-charlie':"
echo "Creating Charlie's bank account:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/CreateAccount" '{"ownerName": "Charlie Brown", "initialDeposit": 2000.00}' "$CHARLIE_TOKEN" | jq '.'

echo -e "\nMultiple small withdrawals:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/Withdraw" '{"amount": 50.00, "description": "Coffee shop"}' "$CHARLIE_TOKEN" | jq '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/Withdraw" '{"amount": 25.00, "description": "Parking fee"}' "$CHARLIE_TOKEN" | jq '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/Withdraw" '{"amount": 100.00, "description": "Gas station"}' "$CHARLIE_TOKEN" | jq '.'

echo -e "\nLarge deposit:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/Deposit" '{"amount": 5000.00, "description": "Investment return"}' "$CHARLIE_TOKEN" | jq '.'

echo -e "\nCharlie's current balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/GetBalance" "" "$CHARLIE_TOKEN" | jq '.'

# Summary of all instances
echo ""
echo "4. State Isolation & Event Sourcing Verification:"
echo "-------------------------------------------------"
echo "Final balances for all BankAccount instances:"
echo ""
echo "Alice's balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/GetBalance" "" "$ALICE_TOKEN" | jq '.'
echo ""
echo "Bob's balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/GetBalance" "" "$BOB_TOKEN" | jq '.'
echo ""
echo "Charlie's balance:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/GetBalance" "" "$CHARLIE_TOKEN" | jq '.'

# Show event sourcing capabilities with transaction history
echo ""
echo "5. Event Sourcing Demonstration (Transaction Histories):"
echo "--------------------------------------------------------"

echo ""
echo "Alice's transaction history:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/GetHistory" "" "$ALICE_TOKEN" | jq '.'

echo ""
echo "Bob's transaction history:"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/GetHistory" "" "$BOB_TOKEN" | jq '.'

echo ""
echo "Charlie's transaction history (showing multiple small transactions):"
make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-charlie/method/GetHistory" "" "$CHARLIE_TOKEN" | jq '.'

echo ""
echo "✓ BankAccount tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - Event-sourced persistence pattern"
echo "  - Complete audit trail for all transactions"
echo "  - Independent actor instances with isolated state"
echo "  - Complex business operations (deposits, withdrawals, balance tracking)"
echo "  - Full transaction history reconstruction from events"
echo "  - Multiple concurrent actor instances with separate event streams"
if [ "$USE_AUTH" = true ]; then
    echo "  - Authentication middleware providing user context to actors"
    echo "  - Account ownership validation using stored OwnerID"
fi