#!/bin/bash

echo "👀 Monitoring Nimbus Task Scheduler"
echo "===================================="
echo ""
echo "Watching for task status changes..."
echo "Press Ctrl+C to stop"
echo ""

PREVIOUS_COUNT=0

while true; do
    # Get current task status
    OUTPUT=$(./bin/client list 2>&1)
    
    # Count tasks by state
    PENDING=$(echo "$OUTPUT" | grep -c "PENDING" || echo "0")
    ASSIGNED=$(echo "$OUTPUT" | grep -c "ASSIGNED" || echo "0")
    RUNNING=$(echo "$OUTPUT" | grep -c "RUNNING" || echo "0")
    SUCCEEDED=$(echo "$OUTPUT" | grep -c "SUCCEEDED" || echo "0")
    FAILED=$(echo "$OUTPUT" | grep -c "FAILED" || echo "0")
    
    TOTAL=$((PENDING + ASSIGNED + RUNNING + SUCCEEDED + FAILED))
    
    # Clear line and print status
    printf "\r📊 Tasks: PENDING=%d | ASSIGNED=%d | RUNNING=%d | SUCCEEDED=%d | FAILED=%d | TOTAL=%d" \
        "$PENDING" "$ASSIGNED" "$RUNNING" "$SUCCEEDED" "$FAILED" "$TOTAL"
    
    # If count changed, show details
    if [ "$TOTAL" != "$PREVIOUS_COUNT" ]; then
        echo ""
        echo "📋 Task Details:"
        echo "$OUTPUT" | head -10
        echo ""
        PREVIOUS_COUNT=$TOTAL
    fi
    
    sleep 2
done

