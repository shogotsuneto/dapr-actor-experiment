#!/bin/bash
# Helper script to set up port forwarding for Kubernetes testing

set -e

CLUSTER_NAME="dapr-actor-dev"

echo "Setting up port forwarding for Kubernetes testing..."

# Check if cluster exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "ERROR: Kind cluster '${CLUSTER_NAME}' not found. Run 'make kind-setup' first."
    exit 1
fi

# Switch to correct context
kubectl config use-context "kind-${CLUSTER_NAME}"

# Start port forwarding
echo "Starting port forwarding..."
kubectl -n dapr-actor-experiment port-forward svc/actor-service 3500:3500 &
PF_PID_DAPR=$!

kubectl -n dapr-actor-experiment port-forward svc/jwks-mock-api 3000:3000 &
PF_PID_JWKS=$!

# Function to cleanup port forwarding
cleanup() {
    echo "Cleaning up port forwarding..."
    kill $PF_PID_DAPR $PF_PID_JWKS 2>/dev/null || true
}

# Set up signal handlers
trap cleanup EXIT INT TERM

# Wait for port forwarding to be ready
sleep 5

echo "Port forwarding active:"
echo "  Dapr HTTP API: http://localhost:3500"
echo "  JWKS Mock API: http://localhost:3000"
echo ""
echo "Press Ctrl+C to stop port forwarding"

# Keep the script running
wait