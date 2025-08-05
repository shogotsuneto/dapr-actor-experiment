#!/bin/bash
# Clean up Kind cluster and resources

set -e

CLUSTER_NAME="dapr-actor-dev"

echo "Cleaning up Kind cluster: ${CLUSTER_NAME}..."

# Delete the cluster
if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    echo "Deleting Kind cluster..."
    kind delete cluster --name="${CLUSTER_NAME}"
    echo "✅ Cluster deleted"
else
    echo "Cluster ${CLUSTER_NAME} not found, nothing to delete"
fi

# Clean up any leftover Docker images (optional)
if docker images | grep -q "dapr-actor-experiment:local"; then
    echo "Removing local Docker image..."
    docker rmi dapr-actor-experiment:local || true
fi

echo "✅ Cleanup complete!"