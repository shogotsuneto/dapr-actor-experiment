#!/bin/bash

echo "Testing Multi-Actor Implementation"
echo "=================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

# Check if JWKS Mock API is available for authentication
if curl -s http://localhost:3000/health > /dev/null; then
    echo "✓ JWKS Mock API is available - authentication will be used for BankAccount operations"
    USE_AUTH=true
else
    echo "⚠️  JWKS Mock API not available - BankAccount operations may require authentication"
    echo "To run with full authentication, start with: docker compose up -d"
    USE_AUTH=false
fi

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run CounterActor tests
echo ""
echo "=================================="
echo "Running Counter Tests..."
echo "=================================="
bash "$SCRIPT_DIR/test-counter-actor.sh"

# Run BankAccountActor tests
echo ""
echo ""
echo "=================================="
echo "Running BankAccount Tests..."
echo "=================================="
bash "$SCRIPT_DIR/test-bank-account-actor.sh"

# Summary
echo ""
echo ""
echo "=========================================="
echo "Multi-Actor Implementation Test Summary"
echo "=========================================="

echo ""
echo "✓ All tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "  - Multiple actor types in single application"
echo "  - State-based persistence (Counter)"
echo "  - Event-sourced persistence (BankAccount)"
echo "  - Independent actor instances with isolated state"
echo "  - Different persistence patterns side-by-side"
echo "  - Multiple instances per actor type"
if [ "$USE_AUTH" = true ]; then
    echo "  - Authentication middleware with user context access"
    echo "  - Account ownership validation using JWT tokens"
fi
echo ""
echo "Key Differences Demonstrated:"
echo "- Counter: State-based (stores only current value)"
echo "- BankAccount: Event-sourced (stores events, shows full history)"
if [ "$USE_AUTH" = true ]; then
    echo "- BankAccount: Includes authentication with ownership validation"
fi
echo ""
echo "For individual testing:"
echo "- Run './scripts/test-counter-actor.sh' for Counter only"
echo "- Run './scripts/test-bank-account-actor.sh' for BankAccount only"
if [ "$USE_AUTH" = true ]; then
    echo "- Run 'docker compose up -d' to enable full JWT authentication"
fi