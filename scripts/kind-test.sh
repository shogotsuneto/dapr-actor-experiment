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

# Start port forwarding in background
echo "Setting up port forwarding..."
kubectl -n dapr-actor-experiment port-forward svc/actor-service 3500:3500 &
PF_PID_DAPR=$!
kubectl -n dapr-actor-experiment port-forward svc/jwks-mock-api 3000:3000 &
PF_PID_JWKS=$!

# Function to cleanup port forwarding
cleanup() {
    echo "Cleaning up port forwarding..."
    kill $PF_PID_DAPR $PF_PID_JWKS 2>/dev/null || true
}
trap cleanup EXIT

# Function to wait for port forwarding to be ready
wait_for_port() {
    local port=$1
    local service_name=$2
    local max_attempts=30
    local attempt=1
    
    echo "Waiting for $service_name port forwarding on :$port to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if curl -s --connect-timeout 1 --max-time 2 http://localhost:$port >/dev/null 2>&1; then
            echo "✓ Port forwarding to $service_name is ready"
            return 0
        fi
        echo "  Attempt $attempt/$max_attempts - waiting for port $port..."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    echo "ERROR: Port forwarding to $service_name on :$port failed to become ready after $max_attempts attempts"
    return 1
}

# Wait for port forwarding to be ready
if ! wait_for_port 3500 "Dapr sidecar"; then
    exit 1
fi

if ! wait_for_port 3000 "JWKS Mock API"; then
    exit 1
fi

# Test health endpoints with proper error handling
echo "Testing health endpoints..."

# For Dapr health check, accept both 200 (success) and 401 (requires auth but running)
DAPR_HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 http://localhost:3500/v1.0/healthz)
if [[ "$DAPR_HEALTH_CODE" == "200" || "$DAPR_HEALTH_CODE" == "401" ]]; then
    echo "✓ Dapr sidecar is running (HTTP $DAPR_HEALTH_CODE)"
elif [[ "$DAPR_HEALTH_CODE" == "000" ]]; then
    echo "ERROR: Cannot connect to Dapr sidecar (connection failed)"
    exit 1
else
    echo "ERROR: Dapr health check failed (HTTP $DAPR_HEALTH_CODE)"
    exit 1
fi

# JWKS Mock API should respond with 200
JWKS_HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 5 --max-time 10 http://localhost:3000/health)
if [[ "$JWKS_HEALTH_CODE" == "200" ]]; then
    echo "✓ JWKS Mock API is running (HTTP $JWKS_HEALTH_CODE)"
elif [[ "$JWKS_HEALTH_CODE" == "000" ]]; then
    echo "ERROR: Cannot connect to JWKS Mock API (connection failed)"
    exit 1
else
    echo "ERROR: JWKS Mock API health check failed (HTTP $JWKS_HEALTH_CODE)"
    exit 1
fi

# Run the existing test scripts (they should work with port forwarding)
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