#!/bin/bash
# Quick test script

echo "Testing Nimbus connection..."
echo ""

# Test 1: Check if master is listening
if lsof -i :50051 > /dev/null 2>&1; then
    echo "✅ Master is listening on port 50051"
else
    echo "❌ Master is NOT listening on port 50051"
    echo "   Start master with: ./bin/master"
    exit 1
fi

# Test 2: Submit a simple task
echo ""
echo "Submitting test task..."
TASK_ID=$(./bin/client submit echo "Test" 2>&1 | grep -o 'task-[0-9]*' | head -1)

if [ -z "$TASK_ID" ]; then
    echo "❌ Failed to submit task"
    exit 1
fi

echo "✅ Task submitted: $TASK_ID"
echo ""
echo "Waiting 5 seconds for worker to pick it up..."
sleep 5

# Test 3: Check task status
echo ""
echo "Checking task status..."
./bin/client status "$TASK_ID" 2>&1 | grep -E "(State:|Task ID:)" || echo "Failed to get status"

echo ""
echo "Listing all tasks:"
./bin/client list 2>&1

