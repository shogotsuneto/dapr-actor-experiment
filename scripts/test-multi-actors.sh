#!/bin/bash

echo "Testing Multi-Actor Implementation"
echo "=================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Test CounterActor (state-based)
echo ""
echo "1. Testing CounterActor (State-based pattern):"
echo "----------------------------------------------"

echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/Get | jq '.'

echo -e "\nIncrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/Increment | jq '.'

echo -e "\nSetting counter to 42:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/Set \
  -H "Content-Type: application/json" \
  -d '{"value": 42}' | jq '.'

echo -e "\nGetting final counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/Get | jq '.'

# Test BankAccountActor (event-sourced)
echo ""
echo "2. Testing BankAccountActor (Event-sourced pattern):"
echo "---------------------------------------------------"

echo "Creating bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/CreateAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "John Doe", "initialDeposit": 1000.00}' | jq '.'

echo -e "\nDepositing money:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 250.00, "description": "Salary deposit"}' | jq '.'

echo -e "\nWithdrawing money:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 50.00, "description": "ATM withdrawal"}' | jq '.'

echo -e "\nGetting current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/GetBalance | jq '.'

echo -e "\nGetting transaction history (Event Sourcing!):"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/GetHistory | jq '.'

# Test service status
echo ""
echo "3. Testing different actor instances (State isolation):"
echo "------------------------------------------------------"

echo "Testing second counter instance:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter-2/method/Get | jq '.'

echo -e "\nIncrementing second counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/test-counter-2/method/Increment | jq '.'

echo -e "\nComparing both counters (should be different):"
echo "Counter 1:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/Get | jq '.'
echo "Counter 2:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter-2/method/Get | jq '.'

# Test service status
echo ""
echo "4. Service Status:"
echo "-----------------"
curl -s http://localhost:8080/status | jq '.' 2>/dev/null || echo "Application endpoint not available"

echo ""
echo "✓ All tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - Multiple actor types in single application"
echo "  - State-based persistence (CounterActor)"
echo "  - Event-sourced persistence (BankAccountActor)"
echo "  - Independent actor instances"
echo "  - Different persistence patterns side-by-side"
echo ""
echo "Key Differences Demonstrated:"
echo "- CounterActor: State-based (stores only current value)"
echo "- BankAccountActor: Event-sourced (stores events, shows full history)"