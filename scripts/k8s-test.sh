#!/bin/bash
# Test the Dapr Actor Experiment on Kubernetes

set -e

CLUSTER_NAME="dapr-actor-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Testing Dapr Actor Experiment on Kubernetes..."

# Check if cluster exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "ERROR: Kind cluster '${CLUSTER_NAME}' not found. Run 'make k8s-setup' first."
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

# Wait for port forwarding to be ready
sleep 5

# Function to cleanup port forwarding
cleanup() {
    echo "Cleaning up port forwarding..."
    kill $PF_PID_DAPR $PF_PID_JWKS 2>/dev/null || true
}
trap cleanup EXIT

# Test health endpoints
echo "Testing health endpoints..."
curl -f http://localhost:3500/v1.0/healthz || { echo "ERROR: Dapr health check failed"; exit 1; }
curl -f http://localhost:3000/health || { echo "ERROR: JWKS Mock API health check failed"; exit 1; }

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