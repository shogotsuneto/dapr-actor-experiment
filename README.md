# Dapr Actor Experiment

A minimal demo of Dapr actors application in Go, demonstrating actor state management and method invocation patterns.

## Overview

This project showcases:
- **Dapr Actor Pattern**: Stateful actor implementation with persistent state
- **Counter Actor**: Simple counter with increment, decrement, get, and set operations
- **Client Interaction**: Example client demonstrating actor method invocation
- **Go Standard Project Layout**: Organized codebase following Go community conventions
- **Multiple Deployment Options**: Local development, Docker Compose, and container setups

## Project Structure

Following the [Go Standard Project Layout](https://github.com/golang-standards/project-layout):

```
├── cmd/                    # Main applications
│   ├── server/            # Actor service application
│   └── client/            # Demo client application
├── internal/              # Private application code
│   └── actor/             # Actor implementations
├── configs/               # Configuration files
│   └── dapr/              # Dapr components and config
├── scripts/               # Build and deployment scripts
├── Makefile               # Build automation
├── Dockerfile             # Container build
└── docker-compose.yml     # Multi-container setup
```

## Building

The project includes a Makefile for common tasks:

```bash
# Build all binaries
make build

# Run tests  
make test

# Run server with Dapr
make run-server

# Run client with Dapr
make run-client

# Clean build artifacts
make clean

# Show help
make help
```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│                 │    │                 │    │                 │
│     Client      │───▶│  Dapr Sidecar   │───▶│  Actor Service  │
│                 │    │   (HTTP API)    │    │   (CounterActor)│
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                │                        │
                                ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │                 │    │                 │
                       │      Redis      │◀───│  State Manager  │
                       │  (State Store)  │    │                 │
                       └─────────────────┘    └─────────────────┘
```

## Features

### Actor Implementation
- **CounterActor**: Stateful actor with persistent counter value
- **Operations**: `get`, `increment`, `decrement`, `set`
- **State Persistence**: Automatic state management via Dapr state store
- **Multiple Instances**: Each actor ID maintains independent state

### Demo Client
- Demonstrates all actor operations
- Shows state persistence across operations
- Tests multiple actor instances
- Comprehensive logging of operations

## Quick Start

### Prerequisites
- Go 1.19+ (for local development)
- Docker (for Redis and optional containerization)
- Dapr CLI (for local development): [Installation Guide](https://docs.dapr.io/getting-started/install-dapr-cli/)

### Option 1: Local Development (Recommended)

This is the most reliable way to test the demo:

1. **Clone and navigate to the repository**:
   ```bash
   git clone https://github.com/shogotsuneto/dapr-actor-experiment.git
   cd dapr-actor-experiment
   ```

2. **Start Redis for state storage**:
   ```bash
   docker run -d --name redis-dapr -p 6379:6379 redis:7-alpine
   ```

3. **Install and initialize Dapr** (if not already done):
   ```bash
   curl -fsSL https://raw.githubusercontent.com/dapr/cli/master/install/install.sh | /bin/bash
   dapr init
   ```

4. **Build and run the actor service**:
   ```bash
   # Using Makefile (recommended)
   make run-server
   
   # Or manually
   go mod tidy
   make build
   dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \
     --components-path ./configs/dapr --config ./configs/dapr/config.yaml -- ./bin/server
   ```

5. **Test with curl** (in a new terminal):
   ```bash
   ./scripts/test-actor.sh
   ```

6. **Or run the demo client** (in a new terminal):
   ```bash
   # Using Makefile (recommended)
   make run-client
   
   # Or manually
   dapr run --app-id client --dapr-http-port 3501 -- ./bin/client
   ```

7. **Clean up**:
   ```bash
   docker stop redis-dapr && docker rm redis-dapr
   ```

### Option 2: Quick Local Test

For a guided experience, use the local testing script:

```bash
./scripts/run-local.sh
```

This script will:
- Check prerequisites
- Start Redis
- Provide instructions for testing
- Optionally run the demo immediately

### Option 3: Docker Compose (Advanced)

For containerized deployment (requires network access for package downloads):

```bash
./scripts/run-server.sh
./scripts/test-actor.sh
docker compose down
```

## Manual Testing

### Using curl

Once the server is running, you can test the actor directly:

```bash
# Get current counter value
curl http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/get

# Increment counter
curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/increment

# Decrement counter  
curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/decrement

# Set counter to specific value
curl -X POST http://localhost:3500/v1.0/actors/CounterActor/counter-1/method/set \
  -H "Content-Type: application/json" \
  -d '{"value": 42}'

# Test different actor instance
curl http://localhost:3500/v1.0/actors/CounterActor/counter-2/method/get
```

### Health Checks

```bash
# Check Dapr sidecar health
curl http://localhost:3500/v1.0/healthz

# Check actor service health
curl http://localhost:8080/health

# Get service status
curl http://localhost:8080/status
```

## Development

### Project Structure
```
├── main.go                    # Actor service implementation
├── client/
│   ├── main.go                # Demo client application
│   └── Dockerfile             # Client container
├── dapr/
│   ├── statestore.yaml        # Redis state store configuration
│   └── config.yaml            # Dapr configuration
├── docker-compose.yml         # Full setup with Dapr (requires network access)
├── docker-compose.simple.yml  # Simplified Docker setup
├── Dockerfile                 # Actor service container
├── run-server.sh              # Server startup script
├── run-client.sh              # Client demo script
├── run-local.sh               # Local development helper
├── test-actor.sh              # Simple curl-based testing
└── README.md
```

### Local Development

To run locally without Docker:

1. **Install Dapr CLI**: Follow [Dapr documentation](https://docs.dapr.io/getting-started/install-dapr-cli/)

2. **Initialize Dapr**: 
   ```bash
   dapr init
   ```

3. **Start Redis**:
   ```bash
   docker run -d -p 6379:6379 redis:7-alpine
   ```

4. **Run actor service**:
   ```bash
   go mod tidy
   dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \
     --components-path ./dapr --config ./dapr/config.yaml -- go run main.go
   ```

5. **Run client** (in new terminal):
   ```bash
   cd client
   dapr run --app-id client --dapr-http-port 3501 -- go run main.go
   ```

## Actor API Reference

### CounterActor Methods

| Method    | Description              | Request Body      | Response         |
|-----------|--------------------------|-------------------|------------------|
| `get`     | Get current value        | None              | `{"value": int}` |
| `increment` | Increment by 1         | None              | `{"value": int}` |
| `decrement` | Decrement by 1         | None              | `{"value": int}` |
| `set`     | Set to specific value    | `{"value": int}`  | `{"value": int}` |

### Actor State

Each actor instance maintains its own counter state:
- **Actor Type**: `CounterActor`
- **Actor ID**: User-defined (e.g., `counter-1`, `counter-2`)
- **State**: Persistent integer counter value
- **Default**: Counter starts at 0 for new instances

## Configuration

### Dapr Configuration
- **State Store**: Redis with actor state store enabled
- **API**: HTTP API v1 enabled for actors
- **Timeouts**: 1h idle timeout, 30s scan interval
- **Tracing**: Full sampling for development

### Docker Configuration
- **Base Images**: Alpine Linux for minimal size
- **Health Checks**: Built-in health monitoring
- **Networking**: Bridge network with service communication

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3500, 6379, 8080 are available
2. **Docker permissions**: Ensure user has Docker permissions
3. **Service startup**: Wait for health checks before testing

### Logs

View service logs:
```bash
# Actor service logs
docker compose logs -f actor-service

# Dapr sidecar logs  
docker compose logs -f actor-service-dapr

# Redis logs
docker compose logs -f redis
```

## Learning Resources

- [Dapr Actors Documentation](https://docs.dapr.io/developing-applications/building-blocks/actors/)
- [Dapr Go SDK](https://docs.dapr.io/developing-applications/sdks/go/)
- [Actor Pattern Overview](https://docs.dapr.io/concepts/actor-pattern/)

## License

This project is provided as-is for educational and demonstration purposes.