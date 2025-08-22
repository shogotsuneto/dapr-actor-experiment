#!/bin/bash

echo "Testing Counter (State-based pattern)"
echo "===================================="

# Check if services are running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "❌ Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

# Check authentication service
if curl -s http://localhost:3000/health > /dev/null; then
    echo "✅ Services ready with authentication"
    USE_AUTH=true
    COUNTER_TOKEN=$(curl -s -X POST http://localhost:3000/generate-token \
        -H "Content-Type: application/json" \
        -d '{"claims": {"sub": "counter-user"}, "expiresIn": 3600}' | \
        grep -o '"token":"[^"]*"' | cut -d'"' -f4)
else
    echo "⚠️  Running without authentication"
    USE_AUTH=false
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
echo "Testing Multiple Counter Instances:"
echo "---------------------------------"

# Test counter-001
echo "→ counter-001 operations:"
echo "  Get initial value:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')

make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Increment" "" "$COUNTER_TOKEN" | jq -c '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Increment" "" "$COUNTER_TOKEN" | jq -c '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Increment" "" "$COUNTER_TOKEN" | jq -c '.'

echo "  Final value:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')

# Test counter-002  
echo "→ counter-002 operations:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Set" '{"value": 10}' "$COUNTER_TOKEN" | jq -c '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Decrement" "" "$COUNTER_TOKEN" | jq -c '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Decrement" "" "$COUNTER_TOKEN" | jq -c '.'

echo "  Final value:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')

# Test counter-003
echo "→ counter-003 operations:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Set" '{"value": 25}' "$COUNTER_TOKEN" | jq -c '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Decrement" "" "$COUNTER_TOKEN" | jq -c '.'

echo "  Final value:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')

echo ""
echo "State Isolation Verification:"
echo "----------------------------"
echo "counter-001:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')
echo "counter-002:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')
echo "counter-003:" $(make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Get" "" "$COUNTER_TOKEN" | jq -r '.data.value')

echo ""
echo "✅ Counter tests completed!"
echo "   • State-based persistence pattern"
echo "   • Independent instances with isolated state"
echo "   • All CRUD operations (Get, Set, Increment, Decrement)"
if [ "$USE_AUTH" = true ]; then
    echo "   • Authentication with JWT tokens"
fi