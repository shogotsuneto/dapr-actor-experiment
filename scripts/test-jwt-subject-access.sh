#!/bin/bash

# Test script to demonstrate JWT validation and subject access patterns in Dapr actors
# Shows what works today and what patterns are available for JWT subject access

set -e

echo "🔐 JWT Subject Access in Dapr Actors - Demo"
echo "=========================================="

# Configuration
JWKS_URL="http://localhost:3001"
DAPR_URL="http://localhost:3500"

echo "📋 Test Configuration:"
echo "  - JWKS Server: ${JWKS_URL}"
echo "  - Dapr Sidecar: ${DAPR_URL}"
echo ""

# Check if services are running
echo "🔍 Checking Services..."
echo -n "  JWKS Server... "
if curl -s -f "${JWKS_URL}/health" > /dev/null; then
    echo "✅"
else
    echo "❌ JWKS Server not responding"
    exit 1
fi

echo -n "  Dapr Sidecar... "
if curl -s -f "${DAPR_URL}/v1.0/healthz" > /dev/null; then
    echo "✅"
else
    echo "❌ Dapr Sidecar not responding"
    exit 1
fi

# Generate JWT token with test user
echo ""
echo "🎫 Generating JWT Token..."
TOKEN_RESPONSE=$(curl -s -X POST "${JWKS_URL}/generate-token" \
    -H "Content-Type: application/json" \
    -d '{"claims": {"sub": "demo-user-123", "role": "admin"}, "expiresIn": 3600}')

VALID_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$VALID_TOKEN" ]; then
    echo "  ✅ JWT token generated for user: demo-user-123"
else
    echo "  ❌ Failed to generate JWT token"
    exit 1
fi

echo ""
echo "🎯 Testing JWT Validation and Access Patterns..."

# Test 1: Direct actor call without JWT (should fail)
echo ""
echo "❌ Test 1: Direct actor call without JWT"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "${DAPR_URL}/v1.0/actors/CounterActor/jwt-demo-counter/method/get")

if [ "$HTTP_CODE" = "401" ]; then
    echo "   ✅ Correctly rejected without JWT (HTTP $HTTP_CODE)"
else
    echo "   ❌ Expected rejection, got HTTP $HTTP_CODE"
fi

# Test 2: Direct actor call with JWT (should work, but no subject access)
echo ""
echo "🎭 Test 2: Direct actor call with JWT (Bearer middleware validates)"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    "${DAPR_URL}/v1.0/actors/CounterActor/jwt-demo-counter/method/get")

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    echo "   ✅ Actor method called successfully (HTTP $HTTP_CODE)"
    echo "   ✅ JWT validation works!"
    echo "   💡 Check server logs - JWT subject NOT directly accessible in actor"
else
    echo "   ❌ Actor method failed (HTTP $HTTP_CODE)"
fi

# Test 3: Service-level pattern demo
echo ""
echo "📡 Test 3: Service handler pattern (demonstrates JWT access approach)"
RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    "${DAPR_URL}/v1.0/invoke/actor-service/method/jwt-demo")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "   ✅ Service handler accessed successfully"
    echo "   📋 Response: $BODY"
    echo "   💡 This demonstrates where JWT extraction would happen"
else
    echo "   ❌ Service handler failed (HTTP $HTTP_CODE)"
fi

echo ""
echo "📝 Current Implementation Status:"
echo ""
echo "✅ **Working Today:**"
echo "   - JWT validation for all actor calls (Bearer middleware)"
echo "   - Access control (401 for invalid/missing JWT)"
echo "   - Actor method execution with valid JWT"
echo "   - Service handlers that could extract JWT headers"
echo ""
echo "🔧 **Patterns for JWT Subject Access:**"
echo "   1. Service handlers extract JWT info from headers"
echo "   2. Service handlers call actors with user info as parameters"
echo "   3. Actors implement JWT-aware methods (e.g., GetWithUser)"
echo ""
echo "💡 **Key Insight:**"
echo "   - Direct actor calls: JWT validated ✅, subject not accessible ❌"
echo "   - Service-mediated calls: JWT extractable ✅, can pass to actors ✅"
echo ""
echo "🎉 JWT Bearer Middleware Demo Complete!"
echo ""
echo "📊 Check the server logs to see:"
echo "   - JWT validation working"
echo "   - Actor method calls"
echo "   - Service handler demonstrations"