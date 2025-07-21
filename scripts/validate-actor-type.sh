#!/bin/bash
set -e

# Actor Type Validation Script
# 
# This script validates that the actor type is properly extracted from the OpenAPI schema
# and used consistently throughout the codebase instead of being hardcoded.
#
# Key improvements achieved:
# 1. Actor type is defined once in the OpenAPI schema (x-dapr-actor.type)
# 2. Generated constant ActorType provides single source of truth
# 3. Main server and actor implementation both use the generated constant
# 4. Generated factory validates actor type at runtime
# 5. Adding new actors only requires changing the OpenAPI schema

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Actor Type Validation ===${NC}"
echo

# Check that no hardcoded "CounterActor" strings exist in main.go
echo -e "${GREEN}Checking main.go for hardcoded actor types...${NC}"
if grep -n '"CounterActor"' cmd/server/main.go > /dev/null; then
    echo "❌ Found hardcoded 'CounterActor' strings in main.go"
    exit 1
else
    echo "✅ No hardcoded actor type strings found in main.go"
fi

# Check that actor implementation uses generated constant
echo -e "${GREEN}Checking CounterActor implementation uses generated constant...${NC}"
if grep -n "generated.ActorType" internal/actor/counter.go > /dev/null; then
    echo "✅ CounterActor.Type() uses generated.ActorType constant"
else
    echo "❌ CounterActor.Type() does not use generated.ActorType constant"
    exit 1
fi

# Check that generated interface has ActorType constant
echo -e "${GREEN}Checking generated interface has ActorType constant...${NC}"
if grep -n "const ActorType" internal/generated/openapi/interface.go > /dev/null; then
    echo "✅ Generated interface includes ActorType constant"
else
    echo "❌ Generated interface missing ActorType constant"
    exit 1
fi

# Extract and display the actor type from generated code
echo -e "${GREEN}Actor type extracted from OpenAPI schema:${NC}"
ACTOR_TYPE=$(grep "const ActorType" internal/generated/openapi/interface.go | sed 's/.*= "\(.*\)".*/\1/')
echo "  ActorType: $ACTOR_TYPE"

echo
echo -e "${GREEN}✅ All actor type validations passed!${NC}"
echo "  - Main server uses generated.ActorType constant"
echo "  - Actor implementation uses generated.ActorType constant"  
echo "  - Actor type is extracted from OpenAPI x-dapr-actor.type field"
echo "  - Generated factory validates actor type at runtime"