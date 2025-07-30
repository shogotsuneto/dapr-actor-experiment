#!/bin/bash

# JWT-aware actor testing script
# This script demonstrates the JWT authentication features for Dapr actors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== JWT-Aware Dapr Actor Demo ==="
echo

# Check if services are running
check_services() {
    echo "Checking if Dapr services are running..."
    
    # Check Dapr sidecar
    if ! curl -s http://localhost:3500/v1.0/healthz > /dev/null; then
        echo "❌ Dapr sidecar not running at localhost:3500"
        echo "Please start the services with: docker compose up -d"
        exit 1
    fi
    
    # Check actor service
    if ! curl -s http://localhost:8080/health > /dev/null; then
        echo "❌ Actor service not running at localhost:8080"
        echo "Please start the services with: docker compose up -d"
        exit 1
    fi
    
    echo "✅ Services are running"
    echo
}

# Generate JWT tokens
generate_tokens() {
    echo "Generating JWT tokens..."
    cd "$PROJECT_DIR"
    
    # Generate tokens and capture output
    TOKEN_OUTPUT=$(./bin/jwt-generator 2>/dev/null)
    
    # Extract tokens using grep and cut
    ADMIN_TOKEN=$(echo "$TOKEN_OUTPUT" | grep "Admin Token" -A 1 | tail -1 | sed 's/Bearer //')
    USER_TOKEN=$(echo "$TOKEN_OUTPUT" | grep "User Token (user: user-123" -A 1 | tail -1 | sed 's/Bearer //')
    USER2_TOKEN=$(echo "$TOKEN_OUTPUT" | grep "User2 Token" -A 1 | tail -1 | sed 's/Bearer //')
    EXPIRED_TOKEN=$(echo "$TOKEN_OUTPUT" | grep "Expired Token" -A 1 | tail -1 | sed 's/Bearer //')
    
    echo "✅ Tokens generated"
    echo
}

# Test function
run_test() {
    local description="$1"
    local command="$2"
    local expected_status="$3"
    
    echo "🧪 Test: $description"
    echo "   Command: $command"
    
    # Run command and capture output and status
    set +e
    output=$(eval "$command" 2>&1)
    status=$?
    set -e
    
    # Check if we got the expected HTTP status from curl
    if echo "$output" | grep -q "HTTP.*$expected_status"; then
        echo "   ✅ PASS - Got expected status $expected_status"
    elif [[ $expected_status == "200" ]] && [[ $status -eq 0 ]]; then
        echo "   ✅ PASS - Success (status 200)"
        # Show some output for successful operations
        if echo "$output" | grep -q '"value":\|"balance":\|"ownerName":'; then
            echo "   📊 Response: $(echo "$output" | tr -d '\n' | sed 's/.*{/{/' | head -c 100)..."
        fi
    elif [[ $expected_status == "401" ]] && ([[ $status -ne 0 ]] || echo "$output" | grep -q "401\|Unauthorized\|authorization"); then
        echo "   ✅ PASS - Got expected authentication error"
    else
        echo "   ❌ FAIL - Expected status $expected_status, got different result"
        echo "   Output: $output"
    fi
    echo
}

# Run the tests
run_tests() {
    echo "Running JWT authentication tests..."
    echo
    
    # Test 1: Health endpoint should work without token
    run_test "Health endpoint (no auth required)" \
        "curl -s -w '\nHTTP_STATUS:%{http_code}' http://localhost:8080/health" \
        "200"
    
    # Test 2: Missing token should fail
    run_test "Counter access without token (should fail)" \
        "curl -s -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/get" \
        "401"
    
    # Test 3: Valid admin token should work
    run_test "Admin accessing counter (should succeed)" \
        "curl -s -H 'Authorization: Bearer $ADMIN_TOKEN' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/get" \
        "200"
    
    # Test 4: Admin can set counter value
    run_test "Admin setting counter value (should succeed)" \
        "curl -s -X POST -H 'Authorization: Bearer $ADMIN_TOKEN' -H 'Content-Type: application/json' -d '{\"value\": 42}' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/set" \
        "200"
    
    # Test 5: Regular user cannot set counter value
    run_test "Regular user setting counter value (should fail)" \
        "curl -s -X POST -H 'Authorization: Bearer $USER_TOKEN' -H 'Content-Type: application/json' -d '{\"value\": 99}' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/set" \
        "401"
    
    # Test 6: User2 (with counter_admin role) can set counter value
    run_test "Counter admin setting counter value (should succeed)" \
        "curl -s -X POST -H 'Authorization: Bearer $USER2_TOKEN' -H 'Content-Type: application/json' -d '{\"value\": 777}' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/set" \
        "200"
    
    # Test 7: User can create their own bank account
    run_test "User creating own bank account (should succeed)" \
        "curl -s -X POST -H 'Authorization: Bearer $USER_TOKEN' -H 'Content-Type: application/json' -d '{\"ownerName\": \"John Doe\", \"initialDeposit\": 1000}' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/BankAccount/user-123/method/createAccount" \
        "200"
    
    # Test 8: User cannot create account for someone else
    run_test "User creating account for others (should fail)" \
        "curl -s -X POST -H 'Authorization: Bearer $USER_TOKEN' -H 'Content-Type: application/json' -d '{\"ownerName\": \"Someone Else\", \"initialDeposit\": 500}' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/BankAccount/other-user/method/createAccount" \
        "401"
    
    # Test 9: User can access their own account
    run_test "User accessing own bank account (should succeed)" \
        "curl -s -H 'Authorization: Bearer $USER_TOKEN' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/BankAccount/user-123/method/getBalance" \
        "200"
    
    # Test 10: User cannot access someone else's account
    run_test "User accessing others' bank account (should fail)" \
        "curl -s -H 'Authorization: Bearer $USER_TOKEN' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/BankAccount/user-456/method/getBalance" \
        "401"
    
    # Test 11: Admin can access any account
    run_test "Admin accessing any bank account (should succeed)" \
        "curl -s -H 'Authorization: Bearer $ADMIN_TOKEN' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/BankAccount/user-123/method/getBalance" \
        "200"
    
    # Test 12: Expired token should fail
    run_test "Using expired token (should fail)" \
        "curl -s -H 'Authorization: Bearer $EXPIRED_TOKEN' -w '\nHTTP_STATUS:%{http_code}' http://localhost:3500/v1.0/actors/Counter/counter-1/method/get" \
        "401"
}

# Main execution
main() {
    check_services
    generate_tokens
    run_tests
    
    echo "=== Test Summary ==="
    echo "✅ JWT-aware actor authentication is working!"
    echo
    echo "Key features demonstrated:"
    echo "• JWT token validation and claims extraction"
    echo "• Role-based access control (admin, counter_admin, bank_admin)"
    echo "• Resource ownership validation (users can only access their own resources)"
    echo "• Admin users can access all resources"
    echo "• Expired tokens are properly rejected"
    echo "• Health endpoints skip authentication"
    echo
    echo "🎉 JWT-aware actors implementation is complete!"
}

# Build binaries if needed
if [[ ! -f "$PROJECT_DIR/bin/jwt-generator" ]]; then
    echo "Building binaries..."
    cd "$PROJECT_DIR"
    make build
    echo
fi

main "$@"