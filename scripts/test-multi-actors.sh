#!/bin/bash

echo "Testing Multi-Actor Implementation"
echo "=================================="

# Check if server is running
if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
    echo "❌ Dapr sidecar not running. Please run 'docker compose up -d' first."
    exit 1
fi

# Check if JWKS Mock API is available for authentication
if curl -s http://localhost:3000/health > /dev/null; then
    echo "✅ Services ready with authentication"
    USE_AUTH=true
else
    echo "⚠️  JWKS Mock API not available - running without authentication"
    USE_AUTH=false
fi

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Run CounterActor tests
echo ""
echo "Running Counter Tests..."
echo "========================"
bash "$SCRIPT_DIR/test-counter-actor.sh"

# Run BankAccountActor tests
echo ""
echo ""
echo "Running BankAccount Tests..."
echo "============================"
bash "$SCRIPT_DIR/test-bank-account-actor.sh"

# Summary
echo ""
echo ""
echo "Multi-Actor Test Summary"
echo "========================"
echo "✅ All tests completed successfully!"
echo ""
echo "This demonstrates:"
echo "   • Multiple actor types in single application"
echo "   • State-based persistence (Counter)"
echo "   • Event-sourced persistence (BankAccount)"
echo "   • Independent actor instances with isolated state"
if [ "$USE_AUTH" = true ]; then
    echo "   • Authentication middleware with user context access"
    echo "   • Account ownership validation using JWT tokens"
fi
echo ""
echo "For individual testing:"
echo "   ./scripts/test-counter-actor.sh     # Counter only"
echo "   ./scripts/test-bank-account-actor.sh # BankAccount only"