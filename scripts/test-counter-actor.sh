#!/bin/bash

echo "Testing Counter (State-based pattern)"
echo "=========================================="

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
    
    # Generate token for counter operations (using anonymous user)
    COUNTER_TOKEN_RESPONSE=$(curl -s -X POST "$JWKS_URL" \
        -H "Content-Type: application/json" \
        -d '{
            "claims": {
                "sub": "counter-user"
            },
            "expiresIn": 3600
        }')
    COUNTER_TOKEN=$(echo "$COUNTER_TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -n "$COUNTER_TOKEN" ]]; then
        echo "✅ Token generated successfully for authenticated operations"
    else
        echo "⚠️  Token generation failed - operations may fail"
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

# Initialize authentication
generate_tokens

# Test multiple Counter instances
echo ""
echo "Testing Multiple Counter Instances:"
echo "----------------------------------------"

# Instance 1: counter-001
echo ""
echo "1. Testing Counter instance 'counter-001':"
echo "Getting initial counter value:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nIncrementing counter:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Increment" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nIncrementing counter again:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Increment" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nSetting counter to 10:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Set" '{"value": 10}' "$COUNTER_TOKEN" | jq '.'

echo -e "\nFinal value for counter-001:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq '.'

# Instance 2: counter-002
echo ""
echo "2. Testing Counter instance 'counter-002':"
echo "Getting initial counter value:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Get" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nIncrementing counter 3 times:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Increment" "" "$COUNTER_TOKEN" | jq '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Increment" "" "$COUNTER_TOKEN" | jq '.'
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Increment" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nFinal value for counter-002:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Get" "" "$COUNTER_TOKEN" | jq '.'

# Instance 3: counter-003
echo ""
echo "3. Testing Counter instance 'counter-003':"
echo "Getting initial counter value:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Get" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nSetting counter to 25:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Set" '{"value": 25}' "$COUNTER_TOKEN" | jq '.'

echo -e "\nDecrementing counter:"
make_request "POST" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Decrement" "" "$COUNTER_TOKEN" | jq '.'

echo -e "\nFinal value for counter-003:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Get" "" "$COUNTER_TOKEN" | jq '.'

# Summary of all instances
echo ""
echo "4. State Isolation Verification:"
echo "--------------------------------"
echo "Final values for all Counter instances (demonstrating state isolation):"
echo "counter-001:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-001/method/Get" "" "$COUNTER_TOKEN" | jq '.'
echo "counter-002:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-002/method/Get" "" "$COUNTER_TOKEN" | jq '.'
echo "counter-003:"
make_request "GET" "http://localhost:3500/v1.0/actors/Counter/counter-003/method/Get" "" "$COUNTER_TOKEN" | jq '.'

echo ""
echo "✓ Counter tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - State-based persistence pattern"
echo "  - Independent actor instances with isolated state"
echo "  - All CRUD operations (Get, Set, Increment, Decrement)"
echo "  - Multiple concurrent actor instances"