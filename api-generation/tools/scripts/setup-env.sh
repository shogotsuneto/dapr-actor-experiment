#!/bin/bash
# Source this file to add API generation tools to your PATH
export PATH="/home/runner/work/dapr-actor-experiment/dapr-actor-experiment/api-generation/tools/bin:$PATH"
echo "API generation tools added to PATH"
echo "Available tools:"
ls -1 "/home/runner/work/dapr-actor-experiment/dapr-actor-experiment/api-generation/tools/bin" | sed 's/^/  - /'
