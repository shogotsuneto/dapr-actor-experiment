#!/bin/bash

# BankAccount Transactions Projection Demo with JWT Authentication
# This script demonstrates the projection functionality with account holder authentication

set -e

echo "BankAccount Transactions Projection Demo with JWT Authentication"
echo "==============================================================="

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

# Generate JWT tokens for different users
echo -e "\n${BLUE}Generating JWT tokens...${NC}"
ALICE_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"claims": {"sub": "account-demo-alice", "name": "Alice Demo"}, "exp_minutes": 60}' | jq -r '.token')
BOB_TOKEN=$(curl -s http://localhost:3000/generate-token -H "Content-Type: application/json" -d '{"claims": {"sub": "account-demo-bob", "name": "Bob Demo"}, "exp_minutes": 60}' | jq -r '.token')

echo -e "${GREEN}✅ JWT tokens generated${NC}"

# Create some test transactions
echo -e "\n${BLUE}Creating test transactions...${NC}"

echo "→ Creating Alice's account..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-alice/method/createAccount \
  -H "Authorization: Bearer $ALICE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"initialDeposit": 5000.0}' | jq '.'

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
  -d '{"initialDeposit": 3000.0}' | jq '.'

echo "→ Bob deposit..."
curl -s -X POST http://localhost:3500/v1.0/actors/BankAccountActor/account-demo-bob/method/deposit \
  -H "Authorization: Bearer $BOB_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 1500.0, "description": "Freelance payment"}' | jq '.'

echo -e "${GREEN}✅ Test transactions created${NC}"

# Wait for projection to process
echo -e "\n${BLUE}Waiting for projector to process events (15 seconds)...${NC}"
sleep 15

echo -e "\n${YELLOW}Account Holder Queries (JWT Authentication Required):${NC}"
echo "======================================================"

# Alice's queries (using Alice's token)
echo -e "\n${BLUE}Alice's Account Data:${NC}"
echo "--------------------"

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

# Bob's queries (using Bob's token)
echo -e "\n${BLUE}Bob's Account Data:${NC}"
echo "------------------"

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

# Demonstrate security
echo -e "\n${YELLOW}Security Demonstration:${NC}"
echo "======================="

echo -e "\n→ Request without JWT token (should fail with 401):"
curl -s -X POST http://localhost:8081/query/my_transactions \
  -H "Content-Type: application/json" \
  -d '{"limit": 5}' || echo -e "${RED}❌ Unauthorized (expected)${NC}"

echo -e "\n${GREEN}✅ JWT Authentication and Security Demo completed!${NC}"
echo ""
echo "Summary:"
echo "  • Account holders can only query their own transaction data"
echo "  • JWT authentication is required for all query endpoints"
echo "  • The 'sub' claim from JWT is mapped to 'user_id' parameter in SQL"
echo "  • Cross-account access is prevented by user_id filtering"
echo ""
echo "Available authenticated queries:"
echo "  • POST /query/my_transactions - Get user's transactions"
echo "  • POST /query/my_account_balance - Get user's account balance"
echo "  • POST /query/my_transaction_summary - Get user's transaction summary"