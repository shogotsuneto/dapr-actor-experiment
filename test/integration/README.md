# Integration Tests

This directory contains integration tests that use a dedicated Docker Compose setup with actual Dapr endpoints.

## Overview

The integration tests validate:
- **CounterActor** operations (Get, Increment, Decrement, Set)
- **BankAccountActor** operations (CreateAccount, Deposit, Withdraw, GetBalance, GetHistory)
- **Multi-actor** scenarios with both actor types
- **State isolation** between actor instances
- **Event sourcing** capabilities of BankAccountActor

## Simplified Architecture

The test setup uses a dedicated `docker-compose.test.yml` file that runs only the services needed for testing:
- **Redis** (state store)
- **Dapr placement service** (for actor distribution)
- **Actor service** (built from source)
- **Dapr sidecar** (for HTTP API access)

Tests run either on the host or can be containerized, while the services being tested run in containers with proper Dapr sidecar and placement service configuration.

## Test Structure

```
test/integration/
├── README.md                      # This file
├── docker-compose.test.yml        # Dedicated compose file for testing
├── client.go                      # Dapr HTTP client utilities
├── counter_test.go                # CounterActor integration tests
├── bankaccount_test.go             # BankAccountActor integration tests
└── multi_test.go                   # Multi-actor integration tests
```

## Running Tests

### Option 1: Automated (Recommended)
The Make target handles service lifecycle automatically:
```bash
make test-integration
```

### Option 2: Manual Control
Start services manually and run tests multiple times:
```bash
# Start test services
docker compose -f test/integration/docker-compose.test.yml up -d

# Run tests (can repeat multiple times)
make test-integration-quick

# Or run specific tests
go test -v ./test/integration -run TestCounterActor
go test -v ./test/integration -run TestBankAccountActor

# Stop services when done
docker compose -f test/integration/docker-compose.test.yml down
```

### Option 3: Run Tests in Docker Container
For environments with different Go versions or when you prefer full containerization:
```bash
make test-integration-docker
```

This option:
- Starts all required services (Redis, Dapr placement, actor service, Dapr sidecar)
- Runs the tests inside a Go 1.24 Docker container
- Automatically handles service lifecycle
- Useful when your local Go version differs from the project requirements

## Configuration

The integration tests support configurable endpoints to accommodate different deployment scenarios:

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DAPR_HTTP_ENDPOINT` | `http://localhost:3500` | Dapr sidecar HTTP endpoint |
| `ACTOR_SERVICE_ENDPOINT` | `http://localhost:8080` | Actor service endpoint |

### Usage Examples

**Default (localhost):**
```bash
make test-integration
```

**Custom endpoints:**
```bash
DAPR_HTTP_ENDPOINT=http://dapr-sidecar:3500 \
ACTOR_SERVICE_ENDPOINT=http://actor-service:8080 \
make test-integration
```

**For Docker-based tests:**
```bash
# Using service names from docker-compose.test.yml
DAPR_HTTP_ENDPOINT=http://actor-service-dapr:3500 \
ACTOR_SERVICE_ENDPOINT=http://actor-service:8080 \
make test-integration-docker
```

**For remote testing:**
```bash
DAPR_HTTP_ENDPOINT=https://staging-dapr.example.com \
ACTOR_SERVICE_ENDPOINT=https://staging-actors.example.com \
go test -v ./test/integration
```

## Benefits of Simplified Architecture

### Before (Complex Setup)
- ❌ Each test managed full Docker lifecycle
- ❌ Long test startup/teardown times
- ❌ Complex Docker management code in Go
- ❌ Resource intensive (starting/stopping services repeatedly)
- ❌ Tests tightly coupled to infrastructure

### After (Simplified Setup)
- ✅ Dedicated test compose file
- ✅ Services start once, tests run multiple times
- ✅ Clean separation between test code and infrastructure
- ✅ Fast test execution with manual service control
- ✅ Simple Docker configuration focused on testing needs

### Test Performance
- **Automated mode**: ~2-3 minutes (includes service startup/teardown)
- **Manual mode**: ~10-15 seconds (services already running)

## Service Requirements

For tests to pass, the following services must be healthy:
- **Dapr sidecar**: `${DAPR_HTTP_ENDPOINT}/v1.0/healthz` (default: `http://localhost:3500/v1.0/healthz`)
- **Actor service**: `${ACTOR_SERVICE_ENDPOINT}/health` (default: `http://localhost:8080/health`)

Tests automatically verify service availability and provide clear error messages if services are not running.

## Troubleshooting

### Services not starting
```bash
# Check service logs
docker compose -f test/integration/docker-compose.test.yml logs

# Check specific service
docker compose -f test/integration/docker-compose.test.yml logs actor-service-dapr
```

### Tests failing with connection errors
```bash
# Verify services are healthy (using default endpoints)
curl http://localhost:3500/v1.0/healthz
curl http://localhost:8080/health

# Or use custom endpoints if configured
curl ${DAPR_HTTP_ENDPOINT}/v1.0/healthz
curl ${ACTOR_SERVICE_ENDPOINT}/health

# Check if ports are available
netstat -tlnp | grep :3500
netstat -tlnp | grep :8080
```

### Reset test environment
```bash
# Clean restart
docker compose -f test/integration/docker-compose.test.yml down -v
docker compose -f test/integration/docker-compose.test.yml up -d --build
```

### Short Mode (Skip Integration Tests)
```bash
go test -short ./test/integration/...
```

## Advantages of Go Integration Tests

1. **Better Error Handling**: Detailed error messages and stack traces
2. **Type Safety**: Compile-time validation of API interactions
3. **Parallel Execution**: Tests can run concurrently
4. **CI/CD Integration**: Standard Go testing tools work out of the box
5. **Maintainability**: Structured, reusable test code
6. **Assertion Library**: Rich assertions with `testify`

## Docker Requirements

The integration tests require Docker and Docker Compose to be available:

- **Docker**: For container management
- **Docker Compose**: For service orchestration
- **Ports**: 3500 (Dapr), 6379 (Redis), 8080 (Actor Service)

### Automatic Service Management

The tests automatically:
1. Start required Docker Compose services
2. Wait for services to be healthy
3. Run test scenarios
4. Clean up services after tests

## Test Configuration

### Timeouts
- Overall test timeout: 5 minutes
- Service startup timeout: 30 retries (60 seconds)
- HTTP client timeout: 5 seconds

### Service Dependencies
- Redis (state store)
- Dapr placement service
- Actor service
- Dapr sidecar

## Debugging

### View Service Logs
```bash
# During test execution (in another terminal)
docker compose logs -f actor-service
docker compose logs -f actor-service-dapr
docker compose logs -f redis
```

### Manual Service Startup
```bash
# Start services manually for debugging
docker compose up -d redis placement actor-service actor-service-dapr

# Run tests against running services
go test -v ./test/integration/...

# Cleanup
docker compose down
```

### Common Issues

1. **Port Conflicts**: Ensure ports 3500, 6379, 8080 are available
2. **Docker Permissions**: Ensure user has Docker access
3. **Service Startup**: Services need time to initialize
4. **Network Issues**: Check Docker network connectivity

## Adding New Tests

### Test Function Pattern
```go
func TestNewFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup Docker services
    composeFile := filepath.Join("..", "..", "docker-compose.yml")
    dockerManager := NewDockerComposeManager(composeFile)
    daprClient := NewDaprClient(GetDaprEndpoint())

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    require.NoError(t, dockerManager.StartServices(ctx))
    defer dockerManager.StopServices(context.Background())

    // Test implementation
    // ...
}
```

## Future Enhancements

- **Parallel Test Execution**: Run test suites concurrently
- **Test Data Management**: Shared test data fixtures
- **Performance Testing**: Latency and throughput measurements
- **Chaos Testing**: Service failure scenarios
- **API Contract Testing**: Schema validation