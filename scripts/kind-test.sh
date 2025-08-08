#!/bin/bash
# Test the Dapr Actor Experiment on Kubernetes

set -e

CLUSTER_NAME="dapr-actor-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Testing Dapr Actor Experiment on Kubernetes..."

# Check if cluster exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "ERROR: Kind cluster '${CLUSTER_NAME}' not found. Run './scripts/kind-setup.sh' first."
    exit 1
fi

# Switch to correct context
kubectl config use-context "kind-${CLUSTER_NAME}"

# Check if services are running
echo "Checking service status..."
kubectl -n dapr-actor-experiment get pods

# Wait for services to be ready
echo "Waiting for services to be ready..."
kubectl -n dapr-actor-experiment wait --for=condition=ready pod -l app=actor-service --timeout=300s
kubectl -n dapr-actor-experiment wait --for=condition=ready pod -l app=jwks-mock-api --timeout=60s
kubectl -n dapr-actor-experiment wait --for=condition=ready pod -l app=redis --timeout=60s

# Test health endpoints using NodePort services (no port forwarding needed)
echo "Testing health endpoints..."

# For Dapr health check, accept both 200 (success) and 401 (requires auth but running)
echo "Checking Dapr sidecar health..."
DAPR_HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 http://localhost:3500/v1.0/healthz 2>/dev/null || echo "000")
if [[ "$DAPR_HEALTH_CODE" == "200" || "$DAPR_HEALTH_CODE" == "401" ]]; then
    echo "✓ Dapr sidecar is running (HTTP $DAPR_HEALTH_CODE)"
elif [[ "$DAPR_HEALTH_CODE" == "000" ]]; then
    echo "ERROR: Cannot connect to Dapr sidecar (connection failed)"
    echo "DEBUG: Checking NodePort service..."
    kubectl -n dapr-actor-experiment get svc actor-service
    exit 1
else
    echo "ERROR: Dapr health check failed (HTTP $DAPR_HEALTH_CODE)"
    exit 1
fi

# JWKS Mock API should respond with 200
echo "Checking JWKS Mock API health..."
JWKS_HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 http://localhost:3000/health 2>/dev/null || echo "000")
if [[ "$JWKS_HEALTH_CODE" == "200" ]]; then
    echo "✓ JWKS Mock API is running (HTTP $JWKS_HEALTH_CODE)"
elif [[ "$JWKS_HEALTH_CODE" == "000" ]]; then
    echo "ERROR: Cannot connect to JWKS Mock API (connection failed)"
    echo "DEBUG: Checking NodePort service..."
    kubectl -n dapr-actor-experiment get svc jwks-mock-api
    exit 1
else
    echo "ERROR: JWKS Mock API health check failed (HTTP $JWKS_HEALTH_CODE)"
    exit 1
fi

# Run the existing test scripts (they should work with direct access)
echo "Running integration tests..."
export DAPR_HTTP_ENDPOINT="http://localhost:3500"
export JWKS_GENERATE_URL="http://localhost:3000/generate-token"
export JWT_ISSUER="http://localhost:3000"
export KUBERNETES_TEST="true"

# Test counter actor
echo "Testing CounterActor..."
"${PROJECT_ROOT}/scripts/test-counter-actor.sh"

# Test bank account actor
echo "Testing BankAccountActor..."
"${PROJECT_ROOT}/scripts/test-bank-account-actor.sh"

# Test multi-actor scenarios
echo "Testing multi-actor scenarios..."
"${PROJECT_ROOT}/scripts/test-multi-actors.sh"

echo "✅ All Kubernetes tests passed!"
echo ""
echo "Deployment info:"
kubectl -n dapr-actor-experiment get pods -o wide
echo ""
echo "Services:"
kubectl -n dapr-actor-experiment get services