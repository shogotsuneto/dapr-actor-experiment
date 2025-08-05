#!/bin/bash
# Deploy the Dapr Actor Experiment to Kind cluster

set -e

CLUSTER_NAME="dapr-actor-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Deploying Dapr Actor Experiment to Kubernetes..."

# Check if cluster exists
if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "ERROR: Kind cluster '${CLUSTER_NAME}' not found. Run 'make kind-setup' first."
    exit 1
fi

# Switch to correct context
kubectl config use-context "kind-${CLUSTER_NAME}"

# Apply namespace
echo "Creating namespace..."
kubectl apply -f "${PROJECT_ROOT}/k8s/namespace.yaml"

# Apply Dapr components and configuration
echo "Applying Dapr components and configuration..."
kubectl apply -f "${PROJECT_ROOT}/k8s/dapr-components.yaml"

# Apply Redis
echo "Deploying Redis..."
kubectl apply -f "${PROJECT_ROOT}/k8s/redis.yaml"

# Apply JWKS Mock API
echo "Deploying JWKS Mock API..."
kubectl apply -f "${PROJECT_ROOT}/k8s/jwks-mock-api.yaml"

# Wait for dependencies to be ready
echo "Waiting for Redis to be ready..."
kubectl -n dapr-actor-experiment wait --for=condition=available --timeout=300s deployment/redis

echo "Waiting for JWKS Mock API to be ready..."
kubectl -n dapr-actor-experiment wait --for=condition=available --timeout=300s deployment/jwks-mock-api

# Apply actor service
echo "Deploying Actor Service..."
kubectl apply -f "${PROJECT_ROOT}/k8s/actor-service.yaml"

# Wait for actor service to be ready
echo "Waiting for Actor Service to be ready..."
kubectl -n dapr-actor-experiment wait --for=condition=available --timeout=300s deployment/actor-service

echo "✅ Deployment complete!"
echo ""
echo "Check status:"
echo "  kubectl -n dapr-actor-experiment get pods"
echo "  kubectl -n dapr-actor-experiment get services"
echo ""
echo "Access services:"
echo "  # Actor service (Dapr HTTP API)"
echo "  kubectl -n dapr-actor-experiment port-forward svc/actor-service 3500:3500"
echo ""
echo "  # JWKS Mock API"
echo "  kubectl -n dapr-actor-experiment port-forward svc/jwks-mock-api 3000:3000"
echo ""
echo "  # Redis (for debugging)"
echo "  kubectl -n dapr-actor-experiment port-forward svc/redis 6379:6379"
echo ""
echo "Run tests:"
echo "  make kind-test"