#!/bin/bash
# Setup Kind cluster for local Kubernetes development

set -e

CLUSTER_NAME="dapr-actor-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "Setting up Kind cluster for Dapr Actor Experiment..."

# Check if required tools are installed
command -v kind >/dev/null 2>&1 || { echo "ERROR: kind is required but not installed. Please install it first: https://kind.sigs.k8s.io/docs/user/quick-start/"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl is required but not installed. Please install it first"; exit 1; }
command -v dapr >/dev/null 2>&1 || { echo "ERROR: dapr CLI is required but not installed. Please install it first: https://docs.dapr.io/getting-started/install-dapr-cli/"; exit 1; }

# Create Kind cluster
echo "Creating Kind cluster: ${CLUSTER_NAME}..."
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Cluster ${CLUSTER_NAME} already exists. Skipping creation."
else
    kind create cluster --config="${PROJECT_ROOT}/k8s/kind-config.yaml" --name="${CLUSTER_NAME}"
fi

# Wait for cluster to be ready
echo "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# Install Dapr on the cluster
echo "Installing Dapr on the cluster..."
dapr init -k --wait --timeout 600

# Wait for Dapr system to be ready
echo "Waiting for Dapr system to be ready..."
kubectl -n dapr-system wait --for=condition=available --timeout=300s deployment/dapr-operator
kubectl -n dapr-system wait --for=condition=available --timeout=300s deployment/dapr-placement-server
kubectl -n dapr-system wait --for=condition=available --timeout=300s deployment/dapr-sidecar-injector
kubectl -n dapr-system wait --for=condition=available --timeout=300s deployment/dapr-sentry

# Build and load the actor service image
echo "Building and loading actor service image..."
cd "${PROJECT_ROOT}"
docker build -t dapr-actor-experiment:local .
kind load docker-image dapr-actor-experiment:local --name="${CLUSTER_NAME}"

echo "✅ Kind cluster setup complete!"
echo ""
echo "Next steps:"
echo "  1. Deploy the application: make k8s-deploy"
echo "  2. Test the application: make k8s-test"
echo "  3. Cleanup when done: make k8s-cleanup"
echo ""
echo "Cluster info:"
echo "  Cluster name: ${CLUSTER_NAME}"
echo "  Kubectl context: kind-${CLUSTER_NAME}"
echo "  Access services via: kubectl port-forward"