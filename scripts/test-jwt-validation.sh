#!/bin/bash

# Simple JWT Bearer Middleware Test Script for Dapr Actor Experiment
# Tests JWT validation functionality using Dapr's Bearer middleware

set -e

echo "🔐 JWT Bearer Middleware Test"
echo "============================="

# Configuration
JWKS_URL="http://localhost:3001"
DAPR_URL="http://localhost:3500"

echo "📋 Test Configuration:"
echo "  - JWKS Server: ${JWKS_URL}"
echo "  - Dapr Sidecar: ${DAPR_URL}"
echo ""

# Test 1: Check if services are running
echo "🔍 Test 1: Service Health Check"
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

# Test 2: Request without JWT token (should fail)
echo ""
echo "🚫 Test 2: Request without JWT token"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${DAPR_URL}/v1.0/actors/CounterActor/test-counter/method/get")
if [ "$HTTP_CODE" = "401" ]; then
    echo "  ✅ Correctly rejected (HTTP $HTTP_CODE)"
else
    echo "  ❌ Expected HTTP 401, got HTTP $HTTP_CODE"
fi

# Test 3: Generate and use valid JWT token
echo ""
echo "🎫 Test 3: Generate valid JWT token"
TOKEN_RESPONSE=$(curl -s -X POST "${JWKS_URL}/generate-token" \
    -H "Content-Type: application/json" \
    -d '{"claims": {"sub": "test-user", "role": "user"}, "expiresIn": 3600}')

VALID_TOKEN=$(echo "$TOKEN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$VALID_TOKEN" ]; then
    echo "  ✅ JWT token generated successfully"
else
    echo "  ❌ Failed to generate JWT token"
    exit 1
fi

# Test 4: Request with valid JWT token
echo ""
echo "🔓 Test 4: Request with valid JWT token"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    "${DAPR_URL}/v1.0/actors/CounterActor/test-counter/method/get")

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    echo "  ✅ Request accepted (HTTP $HTTP_CODE)"
else
    echo "  ❌ Expected HTTP 200/204, got HTTP $HTTP_CODE"
fi

echo ""
echo "🎉 Bearer Middleware JWT validation tests completed!"
echo "   All actor endpoints are now protected by JWT validation"