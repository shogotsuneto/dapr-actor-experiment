# Dapr Actor Experiment

A minimal demo of Dapr actors application in Go, demonstrating actor state management and method invocation patterns.

## Overview

This project showcases:
- **Dapr Actor Pattern**: Stateful actor implementation with persistent state
- **Counter Actor**: Simple counter with increment, decrement, get, and set operations
- **Client Interaction**: Example client demonstrating actor method invocation
- **Go Standard Project Layout**: Organized codebase following Go community conventions
- **Docker-Only Setup**: Simple deployment using Docker Compose, no Dapr CLI required

## Quick Start

### Prerequisites
- Docker and Docker Compose (required)
- Go 1.19+ (optional, for local development)

### Run the Demo

The simplest way to run the demo:

```bash
# Clone and navigate to the repository
git clone https://github.com/shogotsuneto/dapr-actor-experiment.git
cd dapr-actor-experiment

# Start all services using Docker Compose
./scripts/run-docker.sh

# Test the service
./scripts/test-actor.sh

# Cleanup when done
docker compose down
```

This approach:
- Uses Docker Compose for declarative service management
- Builds Go applications from source automatically inside containers
- Uses Redis state store and Dapr sidecar containers
- Requires only Docker and Docker Compose

### Alternative Commands

You can also run Docker Compose commands directly:

```bash
# Start services (builds from source automatically)
docker compose up -d

# Test the service  
./scripts/test-actor.sh

# Stop services
docker compose down
```

## Building

The project includes a Makefile for common tasks:

```bash
# Build all binaries
make build

# Run tests  
make test

# Clean build artifacts
make clean

# Show help
make help
```

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
└── docker-compose.yml     # Docker Compose setup (supports both server-only and client modes)
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

## Development

### For Local Development

If you want to modify the code and test changes:

1. **Make your changes** to the Go code in `cmd/server/` or `cmd/client/`

2. **Rebuild and test**:
   ```bash
   # Rebuild containers and restart
   docker compose build
   docker compose up -d redis actor-service actor-service-dapr
   
   # Test your changes
   ./scripts/test-actor.sh
   ```

3. **View logs** for debugging:
   ```bash
   # Actor service logs
   docker compose logs -f actor-service
   
   # Dapr sidecar logs
   docker compose logs -f actor-service-dapr
   ```

### Running Client Demo

To run the client demo:

```bash
# Start the client with Docker Compose
docker compose --profile client up client client-dapr

# Or run locally (requires server to be running)
go run ./cmd/client
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

## Documentation

This repository includes detailed documentation on various aspects of Dapr actors:

### Architecture and Concepts
- **[Client vs Curl](docs/client-vs-curl.md)** - Understand the difference between using the Go client (Dapr SDK) vs direct HTTP calls with curl
- **[Event Sourcing](docs/event-sourcing.md)** - Learn whether this implementation uses event sourcing and understand the state-based approach
- **[Akka Comparison](docs/akka-comparison.md)** - Compare Dapr actors with Akka actors, including mailbox concepts and architectural differences

### Key Insights
- **Does this use event sourcing?** No - this is a state-based implementation. See [Event Sourcing documentation](docs/event-sourcing.md) for details.
- **How does it compare to Akka?** Both implement the actor model but serve different use cases. See [Akka Comparison](docs/akka-comparison.md) for a detailed analysis.
- **Client vs curl difference?** Both send identical HTTP requests to Dapr sidecar, but the Go client provides type safety and better error handling. See [Client vs Curl](docs/client-vs-curl.md) for details.

## Learning Resources

- [Dapr Actors Documentation](https://docs.dapr.io/developing-applications/building-blocks/actors/)
- [Dapr Go SDK](https://docs.dapr.io/developing-applications/sdks/go/)
- [Actor Pattern Overview](https://docs.dapr.io/concepts/actor-pattern/)

## Appendix: Dapr CLI Usage (Optional)

For advanced users who prefer using the Dapr CLI directly:

<details>
<summary>Click to expand Dapr CLI instructions</summary>

### Prerequisites
- Dapr CLI: [Installation Guide](https://docs.dapr.io/getting-started/install-dapr-cli/)

### Setup
1. **Initialize Dapr**:
   ```bash
   dapr init
   ```

2. **Start Redis**:
   ```bash
   docker run -d --name redis-dapr -p 6379:6379 redis:7-alpine
   ```

3. **Run actor service**:
   ```bash
   make build
   dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \
     --components-path ./configs/dapr --config ./configs/dapr/config.yaml -- ./bin/server
   ```

4. **Run client** (in new terminal):
   ```bash
   dapr run --app-id client --dapr-http-port 3501 -- ./bin/client
   ```

5. **Clean up**:
   ```bash
   docker stop redis-dapr && docker rm redis-dapr
   ```

</details>

## License

This project is provided as-is for educational and demonstration purposes.