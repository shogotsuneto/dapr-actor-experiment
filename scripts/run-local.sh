#!/bin/bash

echo "=== Local Testing Script for Dapr Actor Demo ==="
echo ""
echo "This script demonstrates how to test the Dapr actor demo locally."
echo "Make sure you have Dapr CLI installed: https://docs.dapr.io/getting-started/install-dapr-cli/"
echo ""

# Check if dapr is available
if ! command -v dapr &> /dev/null; then
    echo "❌ Dapr CLI not found. Please install it first:"
    echo "   curl -fsSL https://raw.githubusercontent.com/dapr/cli/master/install/install.sh | /bin/bash"
    echo "   dapr init"
    exit 1
fi

echo "✓ Dapr CLI found"

# Check if redis is running
if ! docker ps | grep redis > /dev/null; then
    echo "Starting Redis container..."
    docker run -d --name redis-dapr -p 6379:6379 redis:7-alpine
    echo "✓ Redis started"
else
    echo "✓ Redis already running"
fi

echo ""
echo "To test the demo:"
echo ""
echo "1. Start the actor service in one terminal:"
echo "   dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \\"
echo "     --components-path ./configs/dapr --config ./configs/dapr/config.yaml -- go run ./cmd/server"
echo ""
echo "2. Test with curl in another terminal:"
echo "   # Get initial value"
echo "   curl http://localhost:3500/v1.0/actors/CounterActor/test-1/method/get"
echo ""
echo "   # Increment counter"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/test-1/method/increment"
echo ""
echo "   # Set counter to specific value"
echo "   curl -X POST http://localhost:3500/v1.0/actors/CounterActor/test-1/method/set \\"
echo "     -H 'Content-Type: application/json' -d '{\"value\": 42}'"
echo ""
echo "3. Or run the client demo:"
echo "   dapr run --app-id client --dapr-http-port 3501 -- go run ./cmd/client"
echo ""
echo "4. To stop:"
echo "   docker stop redis-dapr && docker rm redis-dapr"
echo ""
echo "=== Manual Demo ==="
echo ""
echo "If you want to test right now with a pre-built binary:"

# Build the application if needed
if [ ! -f "./bin/server" ]; then
    echo "Building application..."
    go build -o bin/server ./cmd/server
fi

echo ""
echo "Starting actor service with Dapr..."
echo "Press Ctrl+C to stop when done testing."
echo ""

# Run the application with Dapr
exec dapr run --app-id actor-service --app-port 8080 --dapr-http-port 3500 \
  --components-path ./configs/dapr --config ./configs/dapr/config.yaml -- ./bin/server