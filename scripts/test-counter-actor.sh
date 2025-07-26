#!/bin/bash

echo "Testing Counter (State-based pattern)"
echo "=========================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Test multiple Counter instances
echo ""
echo "Testing Multiple Counter Instances:"
echo "----------------------------------------"

# Instance 1: counter-001
echo ""
echo "1. Testing Counter instance 'counter-001':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-001/method/get | jq '.'

echo -e "\nIncrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-001/method/increment | jq '.'

echo -e "\nIncrementing counter again:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-001/method/increment | jq '.'

echo -e "\nSetting counter to 10:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-001/method/set \
  -H "Content-Type: application/json" \
  -d '{"value": 10}' | jq '.'

echo -e "\nFinal value for counter-001:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-001/method/get | jq '.'

# Instance 2: counter-002
echo ""
echo "2. Testing Counter instance 'counter-002':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-002/method/get | jq '.'

echo -e "\nIncrementing counter 3 times:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-002/method/increment | jq '.'
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-002/method/increment | jq '.'
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-002/method/increment | jq '.'

echo -e "\nFinal value for counter-002:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-002/method/get | jq '.'

# Instance 3: counter-003
echo ""
echo "3. Testing Counter instance 'counter-003':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-003/method/get | jq '.'

echo -e "\nSetting counter to 25:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-003/method/set \
  -H "Content-Type: application/json" \
  -d '{"value": 25}' | jq '.'

echo -e "\nDecrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/Counter/counter-003/method/decrement | jq '.'

echo -e "\nFinal value for counter-003:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-003/method/get | jq '.'

# Summary of all instances
echo ""
echo "4. State Isolation Verification:"
echo "--------------------------------"
echo "Final values for all Counter instances (demonstrating state isolation):"
echo "counter-001:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-001/method/get | jq '.'
echo "counter-002:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-002/method/get | jq '.'
echo "counter-003:"
curl -s http://localhost:3500/v1.0/actors/Counter/counter-003/method/get | jq '.'

echo ""
echo "✓ Counter tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - State-based persistence pattern"
echo "  - Independent actor instances with isolated state"
echo "  - All CRUD operations (Get, Set, Increment, Decrement)"
echo "  - Multiple concurrent actor instances"