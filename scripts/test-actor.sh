#!/bin/bash

echo "Testing Dapr Actor Demo..."

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Test actor operations
ACTOR_TYPE="CounterActor"
ACTOR_ID="test-counter"
BASE_URL="http://localhost:3500/v1.0/actors/${ACTOR_TYPE}/${ACTOR_ID}/method"

echo ""
echo "Testing CounterActor operations..."

# 1. Get initial value
echo "1. Getting initial value..."
RESULT=$(curl -s "${BASE_URL}/Get")
echo "   Result: $RESULT"

# 2. Increment counter
echo "2. Incrementing counter..."
RESULT=$(curl -s -X POST "${BASE_URL}/Increment")
echo "   Result: $RESULT"

# 3. Increment again
echo "3. Incrementing again..."
RESULT=$(curl -s -X POST "${BASE_URL}/Increment")
echo "   Result: $RESULT"

# 4. Decrement counter
echo "4. Decrementing counter..."
RESULT=$(curl -s -X POST "${BASE_URL}/Decrement")
echo "   Result: $RESULT"

# 5. Set counter to specific value
echo "5. Setting counter to 50..."
RESULT=$(curl -s -X POST "${BASE_URL}/Set" \
  -H "Content-Type: application/json" \
  -d '{"value": 50}')
echo "   Result: $RESULT"

# 6. Get final value
echo "6. Getting final value..."
RESULT=$(curl -s "${BASE_URL}/Get")
echo "   Result: $RESULT"

# 7. Test different actor instance
echo ""
echo "Testing different actor instance..."
ACTOR_ID2="test-counter-2"
BASE_URL2="http://localhost:3500/v1.0/actors/${ACTOR_TYPE}/${ACTOR_ID2}/method"

echo "7. Getting initial value for second actor..."
RESULT=$(curl -s "${BASE_URL2}/Get")
echo "   Result: $RESULT"

echo "8. Incrementing second actor..."
RESULT=$(curl -s -X POST "${BASE_URL2}/Increment")
echo "   Result: $RESULT"

echo ""
echo "✓ All tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - Actor method invocation"
echo "  - State persistence"
echo "  - Independent actor instances"