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

# Wait for all dapr-system deployments to be available with better error handling
echo "Checking Dapr deployments..."
DAPR_DEPLOYMENTS=()
while IFS= read -r deployment; do
    DAPR_DEPLOYMENTS+=("$deployment")
done < <(kubectl -n dapr-system get deployments -o name 2>/dev/null | sed 's/deployment.apps\///')

if [ ${#DAPR_DEPLOYMENTS[@]} -eq 0 ]; then
    echo "ERROR: No Dapr deployments found in dapr-system namespace"
    exit 1
fi

echo "Found Dapr deployments: ${DAPR_DEPLOYMENTS[*]}"

# Wait for each deployment to be available
for deployment in "${DAPR_DEPLOYMENTS[@]}"; do
    echo "Waiting for deployment/$deployment to be available..."
    if kubectl -n dapr-system wait --for=condition=available --timeout=300s "deployment/$deployment"; then
        echo "✓ $deployment is ready"
    else
        echo "⚠ Warning: $deployment failed to become available, but continuing..."
    fi
done

# Final verification using dapr status
echo "Verifying Dapr installation..."
if dapr status -k >/dev/null 2>&1; then
    echo "✓ Dapr system is healthy"
else
    echo "⚠ Warning: dapr status check failed, but proceeding with setup..."
fi

# Build and load the actor service image
echo "Building and loading actor service image..."
cd "${PROJECT_ROOT}"
docker build -t dapr-actor-experiment:local .
kind load docker-image dapr-actor-experiment:local --name="${CLUSTER_NAME}"

echo "✅ Kind cluster setup complete!"
echo ""
echo "Next steps:"
echo "  1. Deploy the application: ./scripts/kind-deploy.sh"
echo "  2. Test the application: ./scripts/kind-test.sh"
echo "  3. Cleanup when done: ./scripts/kind-cleanup.sh"
echo ""
echo "Cluster info:"
echo "  Cluster name: ${CLUSTER_NAME}"
echo "  Kubectl context: kind-${CLUSTER_NAME}"
echo "  Access services directly via NodePort on localhost"
echo "  Dashboard will be available at: http://localhost:9080 (after deployment)"