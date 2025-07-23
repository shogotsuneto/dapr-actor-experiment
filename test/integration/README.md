# Integration Tests

This directory contains integration tests that replace the shell scripts in `./scripts/test*.sh`. The tests use a dedicated Docker Compose setup with actual Dapr endpoints and include snapshot testing capabilities for fast execution.

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
├── snapshot.go                    # Snapshot testing utilities
├── counter_test.go                # CounterActor integration tests
├── bankaccount_test.go             # BankAccountActor integration tests
├── multi_test.go                   # Multi-actor integration tests
├── snapshot_test.go                # Snapshot testing demonstrations
├── quick_test.go                   # Fast tests for running services
├── snapshot_simple_test.go         # Simple snapshot examples
└── testdata/
    └── snapshots/                  # Stored test snapshots
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
- **Dapr sidecar**: `http://localhost:3500/v1.0/healthz`
- **Actor service**: `http://localhost:8080/health`

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
# Verify services are healthy
curl http://localhost:3500/v1.0/healthz
curl http://localhost:8080/health

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

## Snapshot Testing

The integration tests include snapshot testing capabilities for fast test execution:

### Creating Snapshots
On first run, snapshots are automatically created for response data.

### Updating Snapshots
```bash
UPDATE_SNAPSHOTS=true go test -v ./test/integration -run TestActorSnapshotIntegration
```

### Benefits of Snapshot Testing
- Fast execution by comparing JSON responses against stored snapshots
- Easy to detect unexpected changes in API responses
- Ideal for regression testing
- Reduces test maintenance overhead

## Comparison with Shell Scripts

| Shell Scripts | Integration Tests |
|---------------|------------------|
| `test-counter-actor.sh` | `counter_test.go` |
| `test-bank-account-actor.sh` | `bankaccount_test.go` |
| `test-multi-actors.sh` | `multi_test.go` |

### Advantages of Go Integration Tests

1. **Better Error Handling**: Detailed error messages and stack traces
2. **Type Safety**: Compile-time validation of API interactions
3. **Parallel Execution**: Tests can run concurrently
4. **CI/CD Integration**: Standard Go testing tools work out of the box
5. **Maintainability**: Structured, reusable test code
6. **Assertion Library**: Rich assertions with `testify`
7. **Snapshot Testing**: Fast regression testing capabilities

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
    daprClient := NewDaprClient("http://localhost:3500")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    require.NoError(t, dockerManager.StartServices(ctx))
    defer dockerManager.StopServices(context.Background())

    // Test implementation
    // ...
}
```

### Snapshot Testing Pattern
```go
WithSnapshotTesting(t, func(t *testing.T, snapshotter *SnapshotTester) {
    resp, err := client.InvokeActorMethod(ctx, ActorMethodRequest{
        ActorType: "MyActor",
        ActorID:   "test-id",
        Method:    "MyMethod",
    })
    require.NoError(t, err)
    snapshotter.MatchJSONSnapshot(t, "my_method_response", resp.Body)
})
```

## Future Enhancements

- **Parallel Test Execution**: Run test suites concurrently
- **Test Data Management**: Shared test data fixtures
- **Performance Testing**: Latency and throughput measurements
- **Chaos Testing**: Service failure scenarios
- **API Contract Testing**: Schema validation