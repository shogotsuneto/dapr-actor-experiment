# Dapr Actor Experiment

Dapr Actor Experiment is a Go-based demonstration application showcasing Dapr actors with Docker-based deployment, schema-first development, and comprehensive testing.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap, Build and Test the Repository
- **Go Version Required**: Go 1.24.4 (specified in go.mod)
- **Dependencies**: Docker and Docker Compose are required for deployment and testing
- **Build Command**: `make build` - takes 25 seconds. NEVER CANCEL. Set timeout to 60+ seconds.
- **Unit Tests**: `make test-unit` - takes 5 seconds. NEVER CANCEL. Set timeout to 30+ seconds.
- **Integration Tests**: `make test-integration` - takes 25 seconds. NEVER CANCEL. Set timeout to 90+ seconds.
- **Full Test Suite**: `make test` - runs both unit and integration tests. NEVER CANCEL. Set timeout to 120+ seconds.

### Docker-Based Deployment (Primary Method)
- **ALWAYS** use Docker Compose for deployment - no Dapr CLI required
- **Start Services**: `./scripts/run-docker.sh` - takes 100 seconds. NEVER CANCEL. Set timeout to 180+ seconds.
- **Alternative Start**: `docker compose up -d --build` - takes 100 seconds. NEVER CANCEL. Set timeout to 180+ seconds.
- **Stop Services**: `docker compose down`
- **CRITICAL**: Docker build includes multi-stage compilation, so no pre-compilation needed

### Code Generation (Advanced)
- **Install Generator**: `make generate-install` - takes 1 second. Downloads Docker image for code generation.
- **Generate Code**: `make generate` - takes 1 second. NEVER CANCEL. Set timeout to 30+ seconds.
- **Clean Generated Code**: `make generate-clean` - preserves implementation files, removes generated files

## Validation

### Manual Testing Commands
ALWAYS run comprehensive test scripts after making changes:
- **Test All Actors**: `./scripts/test-multi-actors.sh` - takes 1 second. Tests both CounterActor and BankAccountActor.
- **Test Counter Only**: `./scripts/test-counter-actor.sh` - tests state-based actor pattern
- **Test BankAccount Only**: `./scripts/test-bank-account-actor.sh` - tests event-sourced actor pattern

### Health Check Commands
- **Actor Service Health**: `curl http://localhost:8080/health` - should return "OK"
- **Dapr Sidecar Health**: `curl http://localhost:3500/v1.0/healthz` - returns "Unauthorized" (expected due to JWT auth)

### Required Validation Steps
ALWAYS run these commands after making changes:
1. `make build` - ensure code compiles
2. `make test-unit` - run unit tests
3. `docker compose up -d --build` - deploy services (WAIT for completion)
4. `./scripts/test-multi-actors.sh` - exercise full functionality
5. `make test-integration` - run comprehensive integration tests
6. `docker compose down` - clean up

## Common Tasks

### Repository Structure
```
├── cmd/                    # Main applications (server, client)
├── internal/               # Private code (auth, counter, bankaccount)
├── schemas/openapi/        # OpenAPI specs for code generation
├── configs/dapr/           # Dapr components and configuration
├── scripts/                # Build, test, and deployment scripts
├── test/integration/       # Comprehensive integration tests
├── k8s/                    # Kubernetes manifests for Kind deployment
├── docs/                   # Architecture and concept documentation
└── Makefile               # Build automation with proper timeouts
```

### Key Files to Know
- **Makefile**: All build commands with appropriate timeouts
- **docker-compose.yml**: Main service orchestration
- **test/integration/docker-compose.test.yml**: Dedicated test environment
- **schemas/openapi/multi-actors.yaml**: API specification for code generation
- **scripts/test-multi-actors.sh**: Comprehensive test script

### Environment Requirements
- **Docker**: Required for all deployment and testing
- **Docker Compose**: Required for service orchestration
- **Ports Used**: 3500 (Dapr), 6379 (Redis), 8080 (Actor Service), 3000 (JWKS Mock API)
- **Go 1.24.4**: Required for local development (optional for Docker-only usage)

## Authentication & Security
- **JWT Authentication**: All actor endpoints require valid JWT tokens
- **JWKS Mock API**: Provides JWT generation and validation at port 3000
- **Bearer Middleware**: Validates tokens and provides user context to actors
- **Account Ownership**: BankAccount actors enforce ownership validation via JWT claims

## Critical Timing Information
- **Build Time**: 25 seconds - NEVER CANCEL, set timeout 60+ seconds
- **Docker Startup**: 100 seconds - NEVER CANCEL, set timeout 180+ seconds
- **Unit Tests**: 5 seconds - NEVER CANCEL, set timeout 30+ seconds
- **Integration Tests**: 25 seconds - NEVER CANCEL, set timeout 90+ seconds
- **Test Scripts**: 1 second - quick validation of running services
- **Code Generation**: 1 second - fast Docker-based generation

## Actor Types Demonstrated
- **CounterActor**: State-based persistence with Get, Set, Increment, Decrement operations
- **BankAccountActor**: Event-sourced persistence with CreateAccount, Deposit, Withdraw, GetBalance, GetHistory operations
- **Multiple Instances**: Each actor type supports multiple independent instances with isolated state
- **State Isolation**: Demonstrated through comprehensive test scenarios

## Troubleshooting
- **Port Conflicts**: Ensure ports 3500, 6379, 8080, 3000 are available before starting services
- **Service Startup**: Always wait for health checks - services need 15+ seconds to initialize
- **Integration Test Failures**: Stop existing Docker services with `docker compose down` before running tests
- **JWT Authentication**: BankAccount operations require JWKS Mock API to be running for token generation

## Documentation References
The repository includes comprehensive documentation:
- **[Multiple Actors](docs/multiple-actors.md)**: Complete guide to actor patterns
- **[Authentication Middleware](docs/authentication-middleware.md)**: JWT setup and user context
- **[Event Sourcing](docs/event-sourcing.md)**: Event-sourced vs state-based patterns
- **[Kubernetes](docs/kubernetes.md)**: Kind setup for local K8s development
- **[Integration Tests](test/integration/README.md)**: Detailed testing documentation

## Alternative Deployment (Kubernetes)
For production-like environments:
- **Setup Kind Cluster**: `./scripts/kind-setup.sh` - takes 60 seconds. NEVER CANCEL. Set timeout 120+ seconds.
- **Deploy to Kind**: `./scripts/kind-deploy.sh` - takes 30 seconds. NEVER CANCEL. Set timeout 90+ seconds.
- **Test on Kind**: `./scripts/kind-test.sh` - smoke tests
- **Integration Tests on Kind**: `make test-integration-kind` - comprehensive tests
- **Cleanup Kind**: `./scripts/kind-cleanup.sh`

## CI/CD Integration
- **GitHub Actions**: `.github/workflows/pr-tests.yml` runs on pull requests
- **CI Commands**: `make build`, `make test-unit`, `make test-integration`
- **Timeout Settings**: CI uses 15-minute timeout for entire workflow
- **Always Successful**: All commands above are validated to work in CI environment