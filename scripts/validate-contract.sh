#!/bin/bash
set -e

# Script to validate that the actor implementation matches the OpenAPI contract
# This demonstrates contract enforcement by testing compilation

echo "=== Contract Validation Test ==="
echo "Testing that CounterActor implementation matches OpenAPI schema..."

cd /home/runner/work/dapr-actor-experiment/dapr-actor-experiment

# First, generate code from current schema
echo "1. Generating code from OpenAPI schema..."
cd api-generation && ./tools/scripts/generate.sh openapi schemas/openapi/counter-actor.yaml
cd ..

# Then, try to build - this should succeed if contract is satisfied
echo "2. Testing compilation (this should succeed)..."
if go build ./...; then
    echo "✓ Contract validation PASSED: Implementation matches OpenAPI schema"
else
    echo "✗ Contract validation FAILED: Implementation does not match OpenAPI schema"
    exit 1
fi

echo "3. The CounterActor implements the generated CounterActorContract interface"
echo "   If you modify the OpenAPI schema and regenerate, compilation will fail"
echo "   unless the implementation is updated to match."

echo ""
echo "=== Contract Enforcement Demonstration ==="
echo "The following enforces contract compliance:"
echo "- CounterActorContract interface is generated from OpenAPI schema"  
echo "- CounterActor must implement this interface (compile-time check)"
echo "- If OpenAPI schema changes, interface changes, and code must be updated"
echo "- This ensures implementation always matches the contract"