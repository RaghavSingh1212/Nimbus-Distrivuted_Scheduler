#!/bin/bash

echo "🌐 Starting Nimbus Web UI..."
echo ""

# Check if master is running
if ! lsof -i :50051 > /dev/null 2>&1; then
    echo "⚠️  Warning: Master is not running on port 50051"
    echo "   Start master first with: ./bin/master"
    echo ""
fi

# Start web server
echo "Starting web server on http://localhost:8080"
echo "Press Ctrl+C to stop"
echo ""

./bin/web -master localhost:50051 -port 8080

