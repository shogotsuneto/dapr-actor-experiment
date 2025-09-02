#!/bin/bash

echo "Quick Actor Demo"
echo "==============="

# Quick health check
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "❌ Services not running. Start with: docker compose up -d"
    exit 1
fi

echo "✅ Services ready"

# Generate tokens
if curl -s http://localhost:3000/health > /dev/null; then
    COUNTER_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
        -H "Content-Type: application/json" \
        -d '{"claims": {"sub": "demo-user"}, "expiresIn": 3600}' | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    ALICE_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
        -H "Content-Type: application/json" \
        -d '{"claims": {"sub": "account-alice"}, "expiresIn": 3600}' | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    AUTH_HEADER_COUNTER="Authorization: Bearer $COUNTER_TOKEN"
    AUTH_HEADER_ALICE="Authorization: Bearer $ALICE_TOKEN"
    echo "🔐 Authentication enabled"
else
    AUTH_HEADER_COUNTER=""
    AUTH_HEADER_ALICE=""
    echo "⚠️  Running without authentication"
fi

echo ""
echo "Counter Demo:"
echo "============"
COUNTER_VALUE=$(curl -s -H "$AUTH_HEADER_COUNTER" http://localhost:3500/v1.0/actors/Counter/demo/method/Get | jq -r '.data.value // 0')
echo "Current value: $COUNTER_VALUE"

curl -s -X POST -H "$AUTH_HEADER_COUNTER" http://localhost:3500/v1.0/actors/Counter/demo/method/Increment | jq -c '.'
COUNTER_VALUE=$(curl -s -H "$AUTH_HEADER_COUNTER" http://localhost:3500/v1.0/actors/Counter/demo/method/Get | jq -r '.data.value')
echo "After increment: $COUNTER_VALUE"

echo ""
echo "BankAccount Demo:"
echo "================="
BALANCE=$(curl -s -H "$AUTH_HEADER_ALICE" http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/GetBalance 2>/dev/null | jq -r '.data.balance // "New Account"')
if [ "$BALANCE" = "New Account" ]; then
    echo "Creating new account..."
    curl -s -X POST -H "$AUTH_HEADER_ALICE" \
        -H "Content-Type: application/json" \
        -d '{"initialDeposit": 1000}' \
        http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/CreateAccount | jq -c '.'
    BALANCE=1000
fi
echo "Current balance: $BALANCE"

echo ""
echo "✅ Quick demo completed!"
echo "   Run the individual test scripts for comprehensive examples:"
echo "   • ./scripts/test-counter-actor.sh"
echo "   • ./scripts/test-bank-account-actor.sh"