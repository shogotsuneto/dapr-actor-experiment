# Kubernetes Manifests

This directory contains Kubernetes manifests for deploying the Dapr Actor Experiment to a local Kind cluster.

## Files

- `kind-config.yaml` - Kind cluster configuration with port mappings
- `namespace.yaml` - Kubernetes namespace for the project
- `redis.yaml` - Redis state store deployment and service
- `jwks-mock-api.yaml` - JWKS Mock API for JWT authentication
- `dapr-components.yaml` - Dapr components and configuration as ConfigMaps
- `actor-service.yaml` - Actor service deployment with Dapr sidecar injection

## Quick Start

```bash
# Setup cluster
./scripts/kind-setup.sh

# Deploy all components
./scripts/kind-deploy.sh

# Test deployment
./scripts/kind-test.sh

# Cleanup
./scripts/kind-cleanup.sh
```

## Manual Deployment

If you prefer to apply manifests manually:

```bash
# Apply in order
kubectl apply -f namespace.yaml
kubectl apply -f dapr-components.yaml
kubectl apply -f redis.yaml
kubectl apply -f jwks-mock-api.yaml
kubectl apply -f actor-service.yaml
```

## Architecture

The deployment creates:
- 1x Redis instance (state store)
- 1x JWKS Mock API instance (JWT authentication)
- 2x Actor service instances (for multi-node testing)
- Dapr components for state management and authentication

All services run in the `dapr-actor-experiment` namespace with proper service discovery and networking.

```
┌─────────────────────────────────────────────────────────────┐
│                    Kind Cluster                             │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │            dapr-actor-experiment namespace              │ │
│  │                                                         │ │
│  │  ┌──────────────┐    ┌──────────────┐                  │ │
│  │  │ Actor Pod 1  │    │ Actor Pod 2  │                  │ │
│  │  │┌────────────┐│    │┌────────────┐│                  │ │
│  │  ││    App     ││    ││    App     ││                  │ │
│  │  ││  :8080     ││    ││  :8080     ││                  │ │
│  │  │└────────────┘│    │└────────────┘│                  │ │
│  │  │┌────────────┐│    │┌────────────┐│                  │ │
│  │  ││   Dapr     ││    ││   Dapr     ││                  │ │
│  │  ││  :3500     ││    ││  :3500     ││                  │ │
│  │  │└────────────┘│    │└────────────┘│                  │ │
│  │  └──────────────┘    └──────────────┘                  │ │
│  │                                                         │ │
│  │  ┌──────────────┐    ┌──────────────┐                  │ │
│  │  │    Redis     │    │ JWKS Mock API│                  │ │
│  │  │    :6379     │    │    :3000     │                  │ │
│  │  └──────────────┘    └──────────────┘                  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
      │                      │                        │
      └─ localhost:3500 ─────┘                        │
      └─ localhost:3000 ─────────────────────────────┘
```