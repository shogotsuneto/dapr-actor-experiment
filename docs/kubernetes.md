# Local Kubernetes Development with Kind

This document explains how to set up and use a local Kubernetes environment for developing and testing the Dapr Actor Experiment using [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker).

## Overview

The Kubernetes setup provides:
- **Isolated Development Environment**: Complete Kubernetes cluster running locally
- **Multiple Application Nodes**: Deploy multiple actor service instances for testing distribution
- **Quick Destroy & Clean Start**: One-command cluster setup and teardown
- **Integration Tests**: Test suites that validate multi-node deployment
- **Production-like Environment**: Similar to cloud Kubernetes deployments

## Prerequisites

Before setting up the local Kubernetes environment, ensure you have the following tools installed:

### Required Tools

1. **Docker**: Container runtime
   ```bash
   # Install Docker Desktop or Docker Engine
   # Ensure Docker is running and accessible
   docker --version
   ```

2. **Kind**: Kubernetes in Docker
   ```bash
   # Install Kind
   # Linux/macOS:
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
   chmod +x ./kind
   sudo mv ./kind /usr/local/bin/kind
   
   # Or using package manager:
   # macOS: brew install kind
   # Windows: choco install kind
   
   kind --version
   ```

3. **kubectl**: Kubernetes CLI
   ```bash
   # Install kubectl
   # Follow instructions at: https://kubernetes.io/docs/tasks/tools/install-kubectl/
   kubectl version --client
   ```

4. **Dapr CLI**: Dapr command-line interface
   ```bash
   # Install Dapr CLI
   # Follow instructions at: https://docs.dapr.io/getting-started/install-dapr-cli/
   dapr --version
   ```

## Quick Start

### 1. Setup Kubernetes Cluster

Create a Kind cluster with Dapr installed:

```bash
# Create cluster, install Dapr, and build application image
make k8s-setup
```

This command:
- Creates a Kind cluster named `dapr-actor-dev`
- Installs Dapr on the cluster
- Builds and loads the application Docker image
- Configures port mappings for external access

### 2. Deploy Application

Deploy all services to the cluster:

```bash
# Deploy Redis, JWKS Mock API, Dapr components, and Actor service
make k8s-deploy
```

This command:
- Creates the `dapr-actor-experiment` namespace
- Deploys Redis state store
- Deploys JWKS Mock API for JWT authentication
- Deploys Dapr components and configuration
- Deploys multiple instances of the actor service (2 replicas by default)

### 3. Run Tests

Test the application running on Kubernetes:

```bash
# Run integration tests against the Kubernetes deployment
make k8s-test
```

This command:
- Sets up port forwarding to access services
- Runs all existing test scripts against the Kubernetes deployment
- Tests multi-node actor distribution
- Verifies service health and functionality

### 4. Check Status

Monitor the deployment status:

```bash
# Check pods, services, and overall status
make k8s-status
```

### 5. Cleanup

Remove the cluster and cleanup resources:

```bash
# Delete the Kind cluster and associated resources
make k8s-cleanup
```

## Architecture

### Kubernetes Components

The Kubernetes deployment includes:

1. **Namespace**: `dapr-actor-experiment`
   - Isolates all resources for the project

2. **Redis Deployment & Service**
   - Provides state storage for Dapr actors
   - Single replica with persistent data (within cluster lifetime)

3. **JWKS Mock API Deployment & Service**
   - Provides JWT token generation and validation
   - Used for authentication testing

4. **Dapr Components ConfigMap**
   - Defines state store configuration
   - Defines JWT Bearer middleware configuration

5. **Actor Service Deployment & Service**
   - Multiple replicas (2 by default) for testing distribution
   - Dapr sidecar injection enabled
   - Health checks and resource limits configured

### Service Communication

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              dapr-actor-experiment namespace            │ │
│  │                                                         │ │
│  │  ┌──────────────┐    ┌──────────────┐                  │ │
│  │  │ Actor Service│    │ Actor Service│                  │ │
│  │  │   (Pod 1)    │    │   (Pod 2)    │                  │ │
│  │  │ ┌──────────┐ │    │ ┌──────────┐ │                  │ │
│  │  │ │   App    │ │    │ │   App    │ │                  │ │
│  │  │ │ :8080    │ │    │ │ :8080    │ │                  │ │
│  │  │ └──────────┘ │    │ └──────────┘ │                  │ │
│  │  │ ┌──────────┐ │    │ ┌──────────┐ │                  │ │
│  │  │ │  Dapr    │ │    │ │  Dapr    │ │                  │ │
│  │  │ │ :3500    │ │    │ │ :3500    │ │                  │ │
│  │  │ └──────────┘ │    │ └──────────┘ │                  │ │
│  │  └──────────────┘    └──────────────┘                  │ │
│  │                                                         │ │
│  │  ┌──────────────┐    ┌──────────────┐                  │ │
│  │  │    Redis     │    │ JWKS Mock API│                  │ │
│  │  │    :6379     │    │    :3000     │                  │ │
│  │  └──────────────┘    └──────────────┘                  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┐
│                        Port Forwarding                       │
│  localhost:3500 → Actor Service (Dapr)                      │
│  localhost:3000 → JWKS Mock API                             │
│  localhost:6379 → Redis (for debugging)                      │
└─────────────────────────────────────────────────────────────┘
```

## Advanced Usage

### Manual Cluster Management

You can manage the cluster manually for more control:

```bash
# Create cluster only
kind create cluster --config=k8s/kind-config.yaml --name=dapr-actor-dev

# Install Dapr
dapr init -k --wait --timeout 600

# Build and load image
docker build -t dapr-actor-experiment:local .
kind load docker-image dapr-actor-experiment:local --name=dapr-actor-dev

# Deploy application
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/dapr-components.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/jwks-mock-api.yaml
kubectl apply -f k8s/actor-service.yaml
```

### Scaling the Deployment

Adjust the number of actor service replicas:

```bash
# Scale to 3 replicas
kubectl -n dapr-actor-experiment scale deployment actor-service --replicas=3

# Check scaling status
kubectl -n dapr-actor-experiment get pods -l app=actor-service
```

### Access Services

Use port forwarding to access services from your local machine:

```bash
# Access Dapr HTTP API
kubectl -n dapr-actor-experiment port-forward svc/actor-service 3500:3500

# Access JWKS Mock API
kubectl -n dapr-actor-experiment port-forward svc/jwks-mock-api 3000:3000

# Access Redis (for debugging)
kubectl -n dapr-actor-experiment port-forward svc/redis 6379:6379
```

### View Logs

Monitor application and Dapr logs:

```bash
# Actor service logs
kubectl -n dapr-actor-experiment logs -l app=actor-service -c actor-service

# Dapr sidecar logs
kubectl -n dapr-actor-experiment logs -l app=actor-service -c daprd

# All logs from a specific pod
kubectl -n dapr-actor-experiment logs <pod-name> --all-containers
```

## Integration Testing

### Kubernetes-Specific Tests

The project includes specialized tests for Kubernetes deployments:

```bash
# Run Kubernetes-specific integration tests
KUBERNETES_TEST=true go test -v ./test/integration -run TestKubernetes
```

These tests verify:
- **Multi-node deployment**: Actors distributed across multiple pods
- **Cross-node communication**: Actors can interact across different nodes
- **State persistence**: Actor state is maintained across pod restarts
- **Placement service**: Dapr placement service works correctly
- **Service discovery**: Services can find and communicate with each other

### Load Testing

Test actor distribution and performance:

```bash
# Run placement tests
KUBERNETES_TEST=true go test -v ./test/integration -run TestKubernetesActorPlacement
```

## Configuration

### Cluster Configuration

The Kind cluster is configured in `k8s/kind-config.yaml`:
- Single control-plane node
- Port mappings for external access
- Resource limits suitable for local development

### Application Configuration

The actor service deployment (`k8s/actor-service.yaml`) includes:
- Multiple replicas for testing distribution
- Dapr sidecar injection with proper annotations
- Health checks and readiness probes
- Resource requests and limits

### Dapr Configuration

Dapr components are defined in `k8s/dapr-components.yaml`:
- Redis state store configuration
- JWT Bearer middleware configuration
- Access control and security settings

## Troubleshooting

### Common Issues

1. **Cluster creation fails**
   ```bash
   # Check Docker is running
   docker info
   
   # Check available resources
   docker stats
   
   # Try with different cluster name
   kind create cluster --name=test-cluster
   ```

2. **Services not starting**
   ```bash
   # Check pod status
   kubectl -n dapr-actor-experiment get pods
   
   # Describe problematic pods
   kubectl -n dapr-actor-experiment describe pod <pod-name>
   
   # Check events
   kubectl -n dapr-actor-experiment get events --sort-by=.metadata.creationTimestamp
   ```

3. **Image not found**
   ```bash
   # Rebuild and reload image
   docker build -t dapr-actor-experiment:local .
   kind load docker-image dapr-actor-experiment:local --name=dapr-actor-dev
   
   # Restart deployment
   kubectl -n dapr-actor-experiment rollout restart deployment/actor-service
   ```

4. **Port forwarding issues**
   ```bash
   # Check if ports are in use
   lsof -i :3500
   lsof -i :3000
   
   # Use different local ports
   kubectl -n dapr-actor-experiment port-forward svc/actor-service 3501:3500
   ```

### Reset Environment

If you encounter persistent issues:

```bash
# Complete cleanup and restart
make k8s-cleanup
make k8s-setup
make k8s-deploy
```

### Debugging

Enable debug logging:

```bash
# Edit the actor service deployment to use debug logging
kubectl -n dapr-actor-experiment edit deployment actor-service

# Add to Dapr annotations:
# dapr.io/log-level: "debug"

# View detailed logs
kubectl -n dapr-actor-experiment logs -l app=actor-service -c daprd --tail=100
```

## Comparison with Docker Compose

| Feature | Docker Compose | Kubernetes (Kind) |
|---------|----------------|-------------------|
| **Setup Complexity** | Simple | Moderate |
| **Resource Usage** | Lower | Higher |
| **Production Similarity** | Limited | High |
| **Multi-node Testing** | No | Yes |
| **Service Discovery** | DNS | Native K8s |
| **Scaling** | Manual | Automatic |
| **Health Checks** | Basic | Advanced |
| **Configuration** | Simple | Flexible |

## Next Steps

1. **Explore Dapr Features**: Try different Dapr components and configurations
2. **Add Monitoring**: Install Prometheus and Grafana for observability
3. **Service Mesh**: Experiment with Istio or Linkerd integration
4. **GitOps**: Set up ArgoCD or Flux for deployment automation
5. **CI/CD**: Integrate Kubernetes tests into your CI/CD pipeline

## References

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Dapr on Kubernetes](https://docs.dapr.io/operations/hosting/kubernetes/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [kubectl Reference](https://kubernetes.io/docs/reference/kubectl/)