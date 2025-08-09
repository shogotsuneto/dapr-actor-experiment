# Local Kubernetes Development with Kind

This document explains how to set up and use a local Kubernetes environment for the Dapr Actor Experiment using [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker).

## Overview

The Kubernetes setup provides an isolated development environment with multiple actor service instances for testing distribution in a production-like setting.

## Prerequisites

Ensure you have the following tools installed:

- **Docker**: Container runtime (Docker Desktop or Docker Engine)
- **Kind**: `go install sigs.k8s.io/kind@v0.29.0` ([other methods](https://kind.sigs.k8s.io/docs/user/quick-start/))
- **kubectl**: Kubernetes CLI ([installation guide](https://kubernetes.io/docs/tasks/tools/install-kubectl/))
- **Dapr CLI**: ([installation guide](https://docs.dapr.io/getting-started/install-dapr-cli/))

## Quick Start

### 1. Setup and Deploy
```bash
# Create cluster with Dapr and build application image
./scripts/kind-setup.sh

# Deploy Redis, JWKS Mock API, and Actor services
./scripts/kind-deploy.sh
```

### 2. Testing

**Smoke Tests** (simplified validation):
```bash
# Run basic API validation tests
./scripts/kind-test.sh
```

**Integration Tests** (comprehensive validation):
```bash
# Automated approach (recommended)
make test-integration-kind

# Manual approach - services are accessible directly via NodePort
export DAPR_HTTP_ENDPOINT="http://localhost:3500"
go test -v ./test/integration/...
```

### 3. Monitor and Cleanup
```bash
# Check status
kubectl -n dapr-actor-experiment get pods

# View logs
kubectl -n dapr-actor-experiment logs -l app=actor-service -c actor-service

# Cleanup
./scripts/kind-cleanup.sh
```

## Dapr Dashboard

The Dapr Dashboard provides a web-based UI to monitor your Dapr applications, components, and system health.

### Accessing the Dashboard

After deployment, the dashboard can be accessed in two ways:

**Method 1: Direct access (recommended - requires cluster recreation)**
If you recreate the Kind cluster after updating to this version:
```
http://localhost:9080
```

**Method 2: Port forwarding (immediate access)**
For existing clusters or immediate access:
```bash
kubectl -n dapr-actor-experiment port-forward svc/dapr-dashboard 9080:8080
```
Then access: `http://localhost:9080`

### Dashboard Features

The dashboard allows you to:
- **Monitor Applications**: View running Dapr applications and their health
- **Inspect Components**: See configured Dapr components (state stores, pub/sub, etc.)
- **View Logs**: Access logs from Dapr sidecars and applications
- **Control Plane Status**: Monitor Dapr system components
- **Metrics**: View application and system metrics

### Dashboard Troubleshooting

If the dashboard is not accessible:

```bash
# Check dashboard pod status
kubectl -n dapr-actor-experiment get pods -l app=dapr-dashboard

# View dashboard logs
kubectl -n dapr-actor-experiment logs -l app=dapr-dashboard

# Verify service and port mapping
kubectl -n dapr-actor-experiment get svc dapr-dashboard

# Use port forwarding as alternative access method
kubectl -n dapr-actor-experiment port-forward svc/dapr-dashboard 9080:8080
```

## Architecture

The deployment creates a `dapr-actor-experiment` namespace with:
- **Redis**: State storage for Dapr actors
- **JWKS Mock API**: JWT authentication testing
- **Actor Service**: Multiple replicas (2 by default) with Dapr sidecars
- **Dapr Dashboard**: Web UI for monitoring Dapr applications and components
- **Dapr Components**: State store and JWT middleware configuration

Services communicate through Kubernetes service discovery, with NodePort services providing direct access from the host machine through Kind's port mapping configuration.

## Advanced Usage

### Manual Cluster Management
```bash
# Create cluster only
kind create cluster --config=k8s/kind-config.yaml --name=dapr-actor-dev

# Install Dapr
dapr init -k --wait --timeout 600

# Build and load image
docker build -t dapr-actor-experiment:local .
kind load docker-image dapr-actor-experiment:local --name=dapr-actor-dev

# Deploy manifests
kubectl apply -f k8s/local/
```

### Service Access and Scaling

The local configuration uses **NodePort services** with **mTLS enabled**:
```bash
# Services are directly accessible via NodePort (no port forwarding needed)
# Dapr sidecar: http://localhost:3500
# Dapr Dashboard: http://localhost:9080
# JWKS Mock API: http://localhost:3000  
# Redis: localhost:6379

# Scale actor service
kubectl -n dapr-actor-experiment scale deployment actor-service --replicas=3

# View logs
kubectl -n dapr-actor-experiment logs -l app=actor-service -c daprd
```

## Troubleshooting

### Common Issues

**Cluster creation fails**: Check Docker is running with `docker info`

**Services not starting**: Check pod status and events:
```bash
kubectl -n dapr-actor-experiment get pods
kubectl -n dapr-actor-experiment get events --sort-by=.metadata.creationTimestamp
```

**Image not found**: Rebuild and reload image:
```bash
docker build -t dapr-actor-experiment:local .
kind load docker-image dapr-actor-experiment:local --name=dapr-actor-dev
kubectl -n dapr-actor-experiment rollout restart deployment/actor-service
```

**Port conflicts**: Check port usage with `lsof -i :3500` and use different local ports if needed

### Debugging with BusyBox
For network debugging and connectivity testing within the cluster:
```bash
# Create a busybox pod for debugging
kubectl run busybox --image=busybox:1.28 --rm -it --restart=Never -- sh

# Inside the busybox pod, test connectivity:
# Test Redis connection
nc -zv redis.dapr-actor-experiment.svc.cluster.local 6379

# Test JWKS Mock API
wget -qO- http://jwks-mock-api.dapr-actor-experiment.svc.cluster.local:3000/health

# Test Actor service
wget -qO- http://actor-service.dapr-actor-experiment.svc.cluster.local:8080/health

# Exit busybox
exit
```

### Reset Environment
```bash
./scripts/kind-cleanup.sh
./scripts/kind-setup.sh
./scripts/kind-deploy.sh
```

## Comparison with Docker Compose

| Feature | Docker Compose | Kubernetes (Kind) |
|---------|----------------|-------------------|
| Setup Complexity | Simple | Moderate |
| Resource Usage | Lower | Higher |
| Multi-node Testing | No | Yes |
| Production Similarity | Limited | High |

## References

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Dapr on Kubernetes](https://docs.dapr.io/operations/hosting/kubernetes/)
- [Integration Test Documentation](../test/integration/README.md)