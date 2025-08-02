#!/bin/bash
# Validation script to ensure all requirements are met

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "🧪 Validating Kubernetes support implementation..."
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success_count=0
total_count=0

check_requirement() {
    local description="$1"
    local test_command="$2"
    
    total_count=$((total_count + 1))
    printf "Checking: %-60s " "$description"
    
    if eval "$test_command" >/dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        success_count=$((success_count + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        return 1
    fi
}

check_file_exists() {
    local description="$1"
    local file_path="$2"
    
    check_requirement "$description" "test -f '$file_path'"
}

check_directory_exists() {
    local description="$1"
    local dir_path="$2"
    
    check_requirement "$description" "test -d '$dir_path'"
}

check_script_executable() {
    local description="$1"
    local script_path="$2"
    
    check_requirement "$description" "test -x '$script_path'"
}

echo -e "${YELLOW}1. Kind Configuration${NC}"
check_file_exists "Kind cluster configuration exists" "$PROJECT_ROOT/k8s/kind-config.yaml"
check_requirement "Kind config has port mappings" "grep -q 'extraPortMappings' '$PROJECT_ROOT/k8s/kind-config.yaml'"
check_requirement "Kind config maps Dapr port 3500" "grep -q '3500' '$PROJECT_ROOT/k8s/kind-config.yaml'"

echo ""
echo -e "${YELLOW}2. Kubernetes Manifests${NC}"
check_directory_exists "Kubernetes manifests directory exists" "$PROJECT_ROOT/k8s"
check_file_exists "Namespace manifest exists" "$PROJECT_ROOT/k8s/namespace.yaml"
check_file_exists "Redis manifest exists" "$PROJECT_ROOT/k8s/redis.yaml"
check_file_exists "JWKS Mock API manifest exists" "$PROJECT_ROOT/k8s/jwks-mock-api.yaml"
check_file_exists "Dapr components manifest exists" "$PROJECT_ROOT/k8s/dapr-components.yaml"
check_file_exists "Actor service manifest exists" "$PROJECT_ROOT/k8s/actor-service.yaml"

echo ""
echo -e "${YELLOW}3. Scripts for Quick Destroy & Clean Start${NC}"
check_script_executable "Setup script is executable" "$PROJECT_ROOT/scripts/k8s-setup.sh"
check_script_executable "Deploy script is executable" "$PROJECT_ROOT/scripts/k8s-deploy.sh"
check_script_executable "Test script is executable" "$PROJECT_ROOT/scripts/k8s-test.sh"
check_script_executable "Cleanup script is executable" "$PROJECT_ROOT/scripts/k8s-cleanup.sh"
check_requirement "Setup script uses Kind" "grep -q 'kind create cluster' '$PROJECT_ROOT/scripts/k8s-setup.sh'"
check_requirement "Cleanup script destroys cluster" "grep -q 'kind delete cluster' '$PROJECT_ROOT/scripts/k8s-cleanup.sh'"

echo ""
echo -e "${YELLOW}4. Makefile Integration${NC}"
check_requirement "Makefile has k8s-setup target" "grep -q 'k8s-setup:' '$PROJECT_ROOT/Makefile'"
check_requirement "Makefile has k8s-deploy target" "grep -q 'k8s-deploy:' '$PROJECT_ROOT/Makefile'"
check_requirement "Makefile has k8s-test target" "grep -q 'k8s-test:' '$PROJECT_ROOT/Makefile'"
check_requirement "Makefile has k8s-cleanup target" "grep -q 'k8s-cleanup:' '$PROJECT_ROOT/Makefile'"
check_requirement "Makefile has k8s-status target" "grep -q 'k8s-status:' '$PROJECT_ROOT/Makefile'"

echo ""
echo -e "${YELLOW}5. Multiple Application Nodes${NC}"
check_requirement "Actor service has multiple replicas" "grep -q 'replicas: 2' '$PROJECT_ROOT/k8s/actor-service.yaml'"
check_requirement "Dapr sidecar injection enabled" "grep -q 'dapr.io/enabled.*true' '$PROJECT_ROOT/k8s/actor-service.yaml'"
check_requirement "Dapr app-id configured" "grep -q 'dapr.io/app-id' '$PROJECT_ROOT/k8s/actor-service.yaml'"

echo ""
echo -e "${YELLOW}6. Integration Tests${NC}"
check_file_exists "Kubernetes integration tests exist" "$PROJECT_ROOT/test/integration/kubernetes_test.go"
check_requirement "Tests check multi-node deployment" "grep -q 'TestKubernetesMultiNodeDeployment' '$PROJECT_ROOT/test/integration/kubernetes_test.go'"
check_requirement "Tests check actor placement" "grep -q 'TestKubernetesActorPlacement' '$PROJECT_ROOT/test/integration/kubernetes_test.go'"
check_requirement "Tests handle multiple actor instances" "grep -q 'numActors.*=' '$PROJECT_ROOT/test/integration/kubernetes_test.go'"

echo ""
echo -e "${YELLOW}7. Documentation${NC}"
check_file_exists "Kubernetes documentation exists" "$PROJECT_ROOT/docs/kubernetes.md"
check_file_exists "K8s directory README exists" "$PROJECT_ROOT/k8s/README.md"
check_requirement "Main README mentions Kubernetes" "grep -q -i 'kubernetes' '$PROJECT_ROOT/README.md'"
check_requirement "Documentation mentions Kind" "grep -q -i 'kind' '$PROJECT_ROOT/docs/kubernetes.md'"
check_requirement "Documentation has setup instructions" "grep -q 'make k8s-setup' '$PROJECT_ROOT/docs/kubernetes.md'"

echo ""
echo -e "${YELLOW}8. Service Configuration${NC}"
check_requirement "Redis state store configured" "grep -q 'state.redis' '$PROJECT_ROOT/k8s/dapr-components.yaml'"
check_requirement "JWT middleware configured" "grep -q 'middleware.http.bearer' '$PROJECT_ROOT/k8s/dapr-components.yaml'"
check_requirement "JWKS URL configured" "grep -q 'jwksURL' '$PROJECT_ROOT/k8s/dapr-components.yaml'"
check_requirement "Actor service has health checks" "grep -q 'livenessProbe' '$PROJECT_ROOT/k8s/actor-service.yaml'"

echo ""
echo -e "${YELLOW}9. YAML Validation${NC}"
for file in "$PROJECT_ROOT"/k8s/*.yaml; do
    filename=$(basename "$file")
    if [[ "$filename" != "kind-config.yaml" ]]; then
        check_requirement "K8s manifest $filename is valid YAML" "python3 -c \"import yaml; list(yaml.safe_load_all(open('$file')))\""
    fi
done

echo ""
echo -e "${YELLOW}10. Code Compilation${NC}"
check_requirement "Kubernetes tests compile" "cd '$PROJECT_ROOT' && go build ./test/integration/..."
check_requirement "Main application compiles" "cd '$PROJECT_ROOT' && go build ./cmd/server"

echo ""
echo "=========================================="
echo -e "Results: ${GREEN}$success_count${NC}/$total_count requirements met"

if [ "$success_count" -eq "$total_count" ]; then
    echo -e "${GREEN}🎉 All requirements validated successfully!${NC}"
    echo ""
    echo "The Kubernetes support implementation includes:"
    echo "  ✓ Isolated project Kubernetes environment using Kind"
    echo "  ✓ Quick destroy & clean start capabilities"
    echo "  ✓ Configurations and templates for necessary resources"  
    echo "  ✓ Integration tests that involve multiple application nodes"
    echo ""
    echo "Next steps:"
    echo "  1. Run 'make k8s-setup' to create the cluster"
    echo "  2. Run 'make k8s-deploy' to deploy the application"
    echo "  3. Run 'make k8s-test' to test multi-node functionality"
    echo "  4. Run 'make k8s-cleanup' when done"
    exit 0
else
    echo -e "${RED}❌ Some requirements are not met. Please review the failed checks above.${NC}"
    exit 1
fi