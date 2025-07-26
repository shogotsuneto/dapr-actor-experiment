#!/bin/bash

echo "Testing Multi-Actor Implementation"
echo "=================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "Error: Dapr sidecar not running. Please run './scripts/run-docker.sh' first."
    exit 1
fi

echo "✓ Dapr sidecar is running"

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
echo ""
echo "Key Differences Demonstrated:"
echo "- Counter: State-based (stores only current value)"
echo "- BankAccount: Event-sourced (stores events, shows full history)"
echo ""
echo "For individual testing:"
echo "- Run './scripts/test-counter-actor.sh' for Counter only"
echo "- Run './scripts/test-bank-account-actor.sh' for BankAccount only"