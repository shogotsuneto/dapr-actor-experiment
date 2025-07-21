#!/bin/bash

# Test script for multiple actors
echo "Testing Multi-Actor Implementation"
echo "=================================="

# Test CounterActor (state-based)
echo ""
echo "1. Testing CounterActor (State-based pattern):"
echo "----------------------------------------------"

echo "Getting initial counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/get | jq '.'

echo -e "\nIncrementing counter:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/increment | jq '.'

echo -e "\nSetting counter to 42:"
curl -s -X POST http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/set \
  -H "Content-Type: application/json" \
  -d '{"value": 42}' | jq '.'

echo -e "\nGetting final counter value:"
curl -s http://localhost:3500/v1.0/actors/CounterActor/test-counter/method/get | jq '.'

# Test BankAccountActor (event-sourced)
echo ""
echo "2. Testing BankAccountActor (Event-sourced pattern):"
echo "---------------------------------------------------"

echo "Creating bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/createAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "John Doe", "initialDeposit": 1000.00}' | jq '.'

echo -e "\nDepositing money:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 250.00, "description": "Salary deposit"}' | jq '.'

echo -e "\nWithdrawing money:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 50.00, "description": "ATM withdrawal"}' | jq '.'

echo -e "\nGetting current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/getBalance | jq '.'

echo -e "\nGetting transaction history (Event Sourcing!):"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-123/method/getHistory | jq '.'

# Test service status
echo ""
echo "3. Service Status:"
echo "-----------------"
curl -s http://localhost:8080/status | jq '.'

echo ""
echo "Test completed!"
echo ""
echo "Key Differences Demonstrated:"
echo "- CounterActor: State-based (stores only current value)"
echo "- BankAccountActor: Event-sourced (stores events, shows full history)"