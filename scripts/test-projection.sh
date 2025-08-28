#!/bin/bash

# BankAccount Transactions Projection Demo
# This script demonstrates the projection functionality

set -e

echo "BankAccount Transactions Projection Demo"
echo "======================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if services are running
echo -e "${BLUE}Checking if services are ready...${NC}"

# Check actor service
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${RED}❌ Actor service not ready at http://localhost:8080${NC}"
    exit 1
fi

# Check query server  
if ! curl -s http://localhost:8081/health > /dev/null; then
    echo -e "${RED}❌ Query server not ready at http://localhost:8081${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Services ready${NC}"

# Generate JWT tokens for different users
echo -e "\n${BLUE}Generating JWT tokens...${NC}"
ALICE_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"sub": "account-demo-alice", "name": "Alice Demo", "exp_minutes": 60}' | jq -r '.token')
BOB_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"sub": "account-demo-bob", "name": "Bob Demo", "exp_minutes": 60}' | jq -r '.token')

echo -e "${GREEN}✅ JWT tokens generated${NC}"

# Create some test transactions
echo -e "\n${BLUE}Creating test transactions...${NC}"

echo "→ Creating Alice's account..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-alice/method/createAccount \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Alice Demo", "initialDeposit": 5000.0}' | jq '.'

echo "→ Alice deposit..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-alice/method/deposit \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 2500.0, "description": "Salary deposit"}' | jq '.'

echo "→ Alice withdrawal..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-alice/method/withdraw \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1200.0, "description": "Rent payment"}' | jq '.'

echo "→ Creating Bob's account..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-bob/method/createAccount \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ownerName": "Bob Demo", "initialDeposit": 3000.0}' | jq '.'

echo "→ Bob deposit..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-bob/method/deposit \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1500.0, "description": "Freelance payment"}' | jq '.'

echo -e "${GREEN}✅ Test transactions created${NC}"

# Wait for projection to process
echo -e "\n${BLUE}Waiting for projector to process events (15 seconds)...${NC}"
sleep 15

echo -e "\n${YELLOW}Demo Queries:${NC}"
echo "============="

# Query 1: Show all transactions
echo -e "\n${BLUE}1. All Transactions:${NC}"
curl -s -X POST http://localhost:8081/query/all_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 50}' | jq '.'

# Query 2: Transaction summary by type
echo -e "\n${BLUE}2. Transaction Summary by Type:${NC}"
curl -s -X POST http://localhost:8081/query/transaction_type_summary \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# Query 3: Account balances
echo -e "\n${BLUE}3. Account Balances (calculated from transactions):${NC}"
curl -s -X POST http://localhost:8081/query/account_balances \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# Query 4: Recent activity
echo -e "\n${BLUE}4. Recent Activity (last 5 transactions):${NC}"
curl -s -X POST http://localhost:8081/query/recent_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 5}' | jq '.'

echo -e "\n${GREEN}✅ Projection demo completed!${NC}"
echo ""
echo "You can now:"
echo "  • Use the REST API endpoints like POST /query/recent_transactions"
echo "  • Check GET /queries endpoint for available query names"
echo "  • Example: curl -X POST http://localhost:8081/query/account_balances -d '{}'"
echo ""
echo "The projection continuously monitors bankaccount events and updates the transactions table."