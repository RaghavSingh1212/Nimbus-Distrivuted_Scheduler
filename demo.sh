#!/bin/bash

set -e

echo "🎬 Nimbus Distributed Task Scheduler Demo"
echo "=========================================="
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if master is running
if ! pgrep -f "./bin/master" > /dev/null; then
    echo -e "${YELLOW}⚠️  Master is not running. Starting master in background...${NC}"
    ./bin/master -port 50051 -db ./demo_master.db &
    MASTER_PID=$!
    sleep 2
    echo -e "${GREEN}✅ Master started (PID: $MASTER_PID)${NC}"
    echo ""
fi

# Start workers
echo -e "${BLUE}Starting workers...${NC}"
./bin/worker -master localhost:50051 -addr localhost:50052 > /dev/null 2>&1 &
WORKER1_PID=$!
sleep 1

./bin/worker -master localhost:50051 -addr localhost:50053 > /dev/null 2>&1 &
WORKER2_PID=$!
sleep 1

./bin/worker -master localhost:50051 -addr localhost:50054 > /dev/null 2>&1 &
WORKER3_PID=$!
sleep 2

echo -e "${GREEN}✅ 3 workers started${NC}"
echo ""

# Submit tasks
echo -e "${BLUE}Submitting 10 tasks...${NC}"
TASK_IDS=()

for i in {1..10}; do
    TASK_ID=$(./bin/client submit echo "Task $i completed" --priority=$((i % 5)) 2>/dev/null | grep -o 'task-[0-9]*')
    TASK_IDS+=($TASK_ID)
    echo "  Submitted: $TASK_ID"
done

echo ""
echo -e "${GREEN}✅ All tasks submitted${NC}"
echo ""

# Wait a bit
echo -e "${BLUE}Waiting for tasks to execute...${NC}"
sleep 5

# Show status
echo ""
echo -e "${BLUE}Task Status:${NC}"
for TASK_ID in "${TASK_IDS[@]}"; do
    STATUS=$(./bin/client status "$TASK_ID" 2>/dev/null | grep "State:" | awk '{print $2}')
    echo "  $TASK_ID: $STATUS"
done

echo ""
echo -e "${BLUE}Listing all tasks:${NC}"
./bin/client list --limit=10

echo ""
echo -e "${YELLOW}💡 Demo complete!${NC}"
echo ""
echo "To clean up, run:"
echo "  pkill -f './bin/master'"
echo "  pkill -f './bin/worker'"
echo "  rm -f demo_master.db"

