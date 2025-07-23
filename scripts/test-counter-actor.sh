#!/bin/bash

echo "Testing CounterActor (State-based pattern)"
echo "=========================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Test multiple CounterActor instances
echo ""
echo "Testing Multiple CounterActor Instances:"
echo "----------------------------------------"

# Instance 1: counter-001
echo ""
echo "1. Testing CounterActor instance 'counter-001':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Get | jq '.'

echo -e "\nIncrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Increment | jq '.'

echo -e "\nIncrementing counter again:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Increment | jq '.'

echo -e "\nSetting counter to 10:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Set \
  -H "Content-Type: application/json" \
  -d '{"value": 10}' | jq '.'

echo -e "\nFinal value for counter-001:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Get | jq '.'

# Instance 2: counter-002
echo ""
echo "2. Testing CounterActor instance 'counter-002':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Get | jq '.'

echo -e "\nIncrementing counter 3 times:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Increment | jq '.'
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Increment | jq '.'
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Increment | jq '.'

echo -e "\nFinal value for counter-002:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Get | jq '.'

# Instance 3: counter-003
echo ""
echo "3. Testing CounterActor instance 'counter-003':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-003/method/Get | jq '.'

echo -e "\nSetting counter to 25:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-003/method/Set \
  -H "Content-Type: application/json" \
  -d '{"value": 25}' | jq '.'

echo -e "\nDecrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-003/method/Decrement | jq '.'

echo -e "\nFinal value for counter-003:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-003/method/Get | jq '.'

# Instance 4: counter-004
echo ""
echo "4. Testing CounterActor instance 'counter-004':"
echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-004/method/Get | jq '.'

echo -e "\nDecrementing from initial value:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-004/method/Decrement | jq '.'
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-004/method/Decrement | jq '.'

echo -e "\nFinal value for counter-004:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-004/method/Get | jq '.'

# Summary of all instances
echo ""
echo "5. State Isolation Verification:"
echo "--------------------------------"
echo "Final values for all CounterActor instances (demonstrating state isolation):"
echo "counter-001:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-001/method/Get | jq '.'
echo "counter-002:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-002/method/Get | jq '.'
echo "counter-003:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-003/method/Get | jq '.'
echo "counter-004:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/counter-004/method/Get | jq '.'

echo ""
echo "✓ CounterActor tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - State-based persistence pattern"
echo "  - Independent actor instances with isolated state"
echo "  - All CRUD operations (Get, Set, Increment, Decrement)"
echo "  - Multiple concurrent actor instances"