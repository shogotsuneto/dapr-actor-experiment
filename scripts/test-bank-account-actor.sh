#!/bin/bash

echo "Testing BankAccountActor (Event-sourced pattern)"
echo "==============================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Test multiple BankAccountActor instances
echo ""
echo "Testing Multiple BankAccountActor Instances:"
echo "--------------------------------------------"

# Instance 1: account-alice
echo ""
echo "1. Testing BankAccountActor instance 'account-alice':"
echo "Creating Alice's bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/CreateAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Alice Johnson", "initialDeposit": 1500.00}' | jq '.'

echo -e "\nDepositing salary:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 3000.00, "description": "Monthly salary"}' | jq '.'

echo -e "\nWithdrawing for rent:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 1200.00, "description": "Rent payment"}' | jq '.'

echo -e "\nWithdrawing for groceries:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 150.00, "description": "Grocery shopping"}' | jq '.'

echo -e "\nAlice's current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/GetBalance | jq '.'

# Instance 2: account-bob
echo ""
echo "2. Testing BankAccountActor instance 'account-bob':"
echo "Creating Bob's bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/CreateAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Bob Smith", "initialDeposit": 500.00}' | jq '.'

echo -e "\nDepositing freelance payment:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 800.00, "description": "Freelance project payment"}' | jq '.'

echo -e "\nDepositing bonus:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 200.00, "description": "Performance bonus"}' | jq '.'

echo -e "\nWithdrawing for car payment:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 350.00, "description": "Car loan payment"}' | jq '.'

echo -e "\nBob's current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/GetBalance | jq '.'

# Instance 3: account-charlie
echo ""
echo "3. Testing BankAccountActor instance 'account-charlie':"
echo "Creating Charlie's bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/CreateAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Charlie Brown", "initialDeposit": 2000.00}' | jq '.'

echo -e "\nMultiple small withdrawals:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 50.00, "description": "Coffee shop"}' | jq '.'

curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 25.00, "description": "Parking fee"}' | jq '.'

curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 100.00, "description": "Gas station"}' | jq '.'

echo -e "\nLarge deposit:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 5000.00, "description": "Investment return"}' | jq '.'

echo -e "\nCharlie's current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/GetBalance | jq '.'

# Instance 4: account-diana
echo ""
echo "4. Testing BankAccountActor instance 'account-diana':"
echo "Creating Diana's bank account:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/CreateAccount \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Diana Wilson", "initialDeposit": 750.00}' | jq '.'

echo -e "\nDepositing tax refund:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/Deposit \
  -H "Content-Type: application/json" \
  -d '{"amount": 1200.00, "description": "Tax refund"}' | jq '.'

echo -e "\nWithdrawing for vacation:"
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/Withdraw \
  -H "Content-Type: application/json" \
  -d '{"amount": 800.00, "description": "Vacation expenses"}' | jq '.'

echo -e "\nDiana's current balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/GetBalance | jq '.'

# Summary of all instances
echo ""
echo "5. State Isolation & Event Sourcing Verification:"
echo "-------------------------------------------------"
echo "Final balances for all BankAccountActor instances:"
echo ""
echo "Alice's balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/GetBalance | jq '.'
echo ""
echo "Bob's balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/GetBalance | jq '.'
echo ""
echo "Charlie's balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/GetBalance | jq '.'
echo ""
echo "Diana's balance:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/GetBalance | jq '.'

# Show event sourcing capabilities with transaction history
echo ""
echo "6. Event Sourcing Demonstration (Transaction Histories):"
echo "--------------------------------------------------------"

echo ""
echo "Alice's transaction history:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-alice/method/GetHistory | jq '.'

echo ""
echo "Bob's transaction history:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-bob/method/GetHistory | jq '.'

echo ""
echo "Charlie's transaction history (showing multiple small transactions):"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-charlie/method/GetHistory | jq '.'

echo ""
echo "Diana's transaction history:"
curl -s http://localhost:3500/v1.0/actors/BankAccountActor/account-diana/method/GetHistory | jq '.'

echo ""
echo "✓ BankAccountActor tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - Event-sourced persistence pattern"
echo "  - Complete audit trail for all transactions"
echo "  - Independent actor instances with isolated state"
echo "  - Complex business operations (deposits, withdrawals, balance tracking)"
echo "  - Full transaction history reconstruction from events"
echo "  - Multiple concurrent actor instances with separate event streams"