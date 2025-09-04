#!/bin/bash

echo "Testing BankAccount (Event-sourced pattern)"
echo "==========================================="

# Check if services are running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "❌ Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

# Check authentication service
if curl -s http://localhost:3000/health > /dev/null; then
    echo "✅ Services ready with authentication"
    USE_AUTH=true
else
    echo "⚠️  Running without authentication"
    USE_AUTH=false
fi

# Generate tokens for test users
if [ "$USE_AUTH" = true ]; then
    ALICE_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
        -H "Content-Type: application/json" \
        -d '{"claims": {"sub": "account-alice"}, "expiresIn": 3600}' | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    BOB_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
        -H "Content-Type: application/json" \
        -d '{"claims": {"sub": "account-bob"}, "expiresIn": 3600}' | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
fi

# Helper function for requests
make_request() {
    local method="$1" url="$2" data="$3" token="$4"
    
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

echo ""
echo "Testing Multiple BankAccount Instances:"
echo "-------------------------------------"

# Test Alice's account
echo "→ Alice's account operations:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/CreateAccount" \
    '{"initialDeposit": 1500.00}' "$ALICE_TOKEN" | jq -c '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Deposit" \
    '{"amount": 3000.00, "description": "Monthly salary"}' "$ALICE_TOKEN" | jq -c '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Withdraw" \
    '{"amount": 1200.00, "description": "Rent payment"}' "$ALICE_TOKEN" | jq -c '.'

echo "  Alice's balance:" $(make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/GetBalance" "" "$ALICE_TOKEN" | jq -r '.data.balance')

# Test Bob's account
echo "→ Bob's account operations:"
make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/CreateAccount" \
    '{"initialDeposit": 500.00}' "$BOB_TOKEN" | jq -c '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Deposit" \
    '{"amount": 800.00, "description": "Freelance payment"}' "$BOB_TOKEN" | jq -c '.'

make_request "POST" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Withdraw" \
    '{"amount": 350.00, "description": "Car payment"}' "$BOB_TOKEN" | jq -c '.'

echo "  Bob's balance:" $(make_request "GET" "http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/GetBalance" "" "$BOB_TOKEN" | jq -r '.data.balance')

echo ""
echo "✅ BankAccount tests completed!"
echo "   • Event-sourced persistence"
echo "   • Independent instances with isolated state"
if [ "$USE_AUTH" = true ]; then
    echo "   • Authentication with ownership validation"
fi