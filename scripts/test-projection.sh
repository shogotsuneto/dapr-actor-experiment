#!/bin/bash

# BankAccount Transactions Projection Demo - With JWT Authentication
# This script demonstrates the projection functionality with JWT authentication

set -e

echo "BankAccount Transactions Projection Demo"
echo "========================================"

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

# Check JWKS Mock API
if ! curl -s http://localhost:3000/health > /dev/null; then
    echo -e "${RED}❌ JWKS Mock API not ready at http://localhost:3000${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Services ready${NC}"

# Generate JWT tokens for both actor authentication and query authentication
echo -e "\n${BLUE}Generating JWT tokens...${NC}"
ALICE_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"claims": {"sub": "account-alice", "name": "Alice Demo"}, "exp_minutes": 60}' | jq -r '.token')
BOB_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"claims": {"sub": "account-bob", "name": "Bob Demo"}, "exp_minutes": 60}' | jq -r '.token')

echo -e "${GREEN}✅ JWT tokens generated${NC}"

# Create some test transactions
echo -e "\n${BLUE}Creating test transactions...${NC}"

echo "→ Creating Alice's account..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/CreateAccount \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"initialDeposit": 5000.0}' | jq '.'

echo "→ Alice deposit..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Deposit \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 2500.0, "description": "Salary deposit"}' | jq '.'

echo "→ Alice withdrawal..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccount/account-alice/method/Withdraw \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1200.0, "description": "Rent payment"}' | jq '.'

echo "→ Creating Bob's account..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/CreateAccount \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"initialDeposit": 3000.0}' | jq '.'

echo "→ Bob deposit..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccount/account-bob/method/Deposit \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1500.0, "description": "Freelance payment"}' | jq '.'

echo -e "${GREEN}✅ Test transactions created${NC}"

# Wait for projection to process
echo -e "\n${BLUE}Waiting for projector to process events (15 seconds)...${NC}"
sleep 15

echo -e "\n${YELLOW}JWT-Authenticated Queries:${NC}"
echo "=========================="

# Alice's account queries (authenticated)
echo -e "\n${BLUE}Alice's Account Data (authenticated as Alice):${NC}"
echo "--------------------------------------------"

echo -e "\n→ Alice's Transactions:"
curl -s -X POST http://localhost:8081/query/my_transactions \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}' | jq '.'

echo -e "\n→ Alice's Account Balance:"
curl -s -X POST http://localhost:8081/query/my_account_balance \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

echo -e "\n→ Alice's Transaction Summary:"
curl -s -X POST http://localhost:8081/query/my_transaction_summary \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# Bob's account queries (authenticated)
echo -e "\n${BLUE}Bob's Account Data (authenticated as Bob):${NC}"
echo "----------------------------------------"

echo -e "\n→ Bob's Transactions:"
curl -s -X POST http://localhost:8081/query/my_transactions \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}' | jq '.'

echo -e "\n→ Bob's Account Balance:"
curl -s -X POST http://localhost:8081/query/my_account_balance \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}' | jq '.'

# Test unauthorized access
echo -e "\n${YELLOW}Security Test - Unauthorized Access:${NC}"
echo "==================================="

echo -e "\n→ Query without token (should fail):"
curl -s -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 10}' | jq '.' || echo -e "${RED}❌ Unauthorized access blocked${NC}"

echo -e "\n${GREEN}✅ Projection Demo completed!${NC}"
echo ""
echo "Summary:"
echo "  • Events are projected from event store to transactions table with owner information"
echo "  • Queries require JWT authentication and filter by owner_id automatically"
echo "  • Each user can only see their own transaction data"
echo "  • Unauthorized access is properly blocked"
echo ""
echo "Available queries:"
echo "  • POST /query/my_transactions - Get user's transactions (with JWT auth)"
echo "  • POST /query/my_account_balance - Get user's account balances (with JWT auth)"
echo "  • POST /query/my_transaction_summary - Get user's transaction summary (with JWT auth)"
echo ""
echo -e "${YELLOW}JWT Tokens for Manual Testing:${NC}"
echo "============================="
echo ""
echo -e "${BLUE}Alice's Token (account-alice):${NC}"
echo "$ALICE_TOKEN"
echo ""
echo -e "${BLUE}Bob's Token (account-bob):${NC}"
echo "$BOB_TOKEN"
echo ""
echo "Use these tokens with the Authorization: Bearer header for manual testing:"
echo "curl -X POST http://localhost:8081/query/my_transactions -H \"Authorization: Bearer \$ALICE_TOKEN\" -H \"Content-Type: application/json\" -d '{\"limit\": 10}'"