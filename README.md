# Dapr Actor Experiment

A minimal demo of Dapr actors application in Go, demonstrating actor state management and method invocation patterns.

## Overview

This project showcases:
- **Dapr Actor Pattern**: Stateful actor implementation with persistent state
- **Counter Actor**: Simple counter with increment, decrement, get, and set operations
- **Client Interaction**: Example client demonstrating actor method invocation
- **Docker Compose Setup**: Complete local testing environment with Dapr sidecar

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
- Docker and Docker Compose
- Git (for cloning)

### Running the Demo

1. **Clone and navigate to the repository**:
   ```bash
   git clone https://github.com/shogotsuneto/dapr-actor-experiment.git
   cd dapr-actor-experiment
   ```

2. **Start the actor service**:
   ```bash
   ./run-server.sh
   ```
   This will:
   - Start Redis (state store)
   - Build and start the actor service
   - Start Dapr sidecar
   - Verify all services are healthy

3. **Run the demo client** (in a new terminal):
   ```bash
   ./run-client.sh
   ```

4. **Stop the services**:
   ```bash
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
├── main.go              # Actor service implementation
├── client/
│   ├── main.go          # Demo client application
│   └── Dockerfile       # Client container
├── dapr/
│   ├── statestore.yaml  # Redis state store configuration
│   └── config.yaml      # Dapr configuration
├── docker-compose.yml   # Complete setup with Dapr
├── Dockerfile           # Actor service container
├── run-server.sh        # Server startup script
├── run-client.sh        # Client demo script
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