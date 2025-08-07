#!/bin/bash
# Generate a single Kubernetes manifest file from all components

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
OUTPUT_FILE="${PROJECT_ROOT}/k8s/all-in-one.yaml"

echo "Generating all-in-one Kubernetes manifest..."

# Create the output file with header
cat > "${OUTPUT_FILE}" << 'EOF'
# All-in-one Kubernetes manifest for Dapr Actor Experiment
# Generated from individual manifest files
# Apply with: kubectl apply -f k8s/all-in-one.yaml

EOF

# Combine all manifest files
for file in "${PROJECT_ROOT}"/k8s/local/*.yaml; do
    # Skip the all-in-one file itself and kind-config
    filename=$(basename "$file")
    if [[ "$filename" != "all-in-one.yaml" && "$filename" != "kind-config.yaml" ]]; then
        echo "# From: $filename" >> "${OUTPUT_FILE}"
        cat "$file" >> "${OUTPUT_FILE}"
        echo "" >> "${OUTPUT_FILE}"
        echo "---" >> "${OUTPUT_FILE}"
        echo "" >> "${OUTPUT_FILE}"
    fi
done

# Remove the trailing separator
sed -i '$ { /^---$/d; /^$/d; }' "${OUTPUT_FILE}"

echo "✅ All-in-one manifest generated: ${OUTPUT_FILE}"
echo ""
echo "Usage:"
echo "  kubectl apply -f k8s/all-in-one.yaml"