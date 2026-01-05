#!/bin/bash

echo "🚀 Starting Complete Nimbus System"
echo "===================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if binaries exist
if [ ! -f "bin/master" ] || [ ! -f "bin/worker" ]; then
    echo "❌ Binaries not found. Building..."
    ./run.sh
fi

# Kill any existing processes
echo "Cleaning up old processes..."
pkill -f "./bin/master" 2>/dev/null
pkill -f "./bin/worker" 2>/dev/null
sleep 1

# Start master
echo -e "${GREEN}Starting Master...${NC}"
./bin/master -port 50051 -db ./master.db > master.log 2>&1 &
MASTER_PID=$!
sleep 2

if ps -p $MASTER_PID > /dev/null; then
    echo -e "${GREEN}✅ Master started (PID: $MASTER_PID)${NC}"
else
    echo "❌ Master failed to start. Check master.log"
    exit 1
fi

# Start worker
echo -e "${GREEN}Starting Worker...${NC}"
./bin/worker -master localhost:50051 -addr localhost:50052 > worker.log 2>&1 &
WORKER_PID=$!
sleep 2

if ps -p $WORKER_PID > /dev/null; then
    echo -e "${GREEN}✅ Worker started (PID: $WORKER_PID)${NC}"
else
    echo "❌ Worker failed to start. Check worker.log"
    kill $MASTER_PID 2>/dev/null
    exit 1
fi

echo ""
echo -e "${YELLOW}System Status:${NC}"
echo "  Master: http://localhost:50051 (PID: $MASTER_PID)"
echo "  Worker: Connected (PID: $WORKER_PID)"
echo ""
echo "Logs:"
echo "  Master: tail -f master.log"
echo "  Worker: tail -f worker.log"
echo ""
echo "To stop: ./stop_all.sh"
echo ""
echo "Testing system..."
sleep 2

# Test submission
TASK_ID=$(./bin/client submit echo "Test task" 2>&1 | grep -o 'task-[0-9]*' | head -1)

if [ ! -z "$TASK_ID" ]; then
    echo -e "${GREEN}✅ Task submitted: $TASK_ID${NC}"
    echo "Waiting 3 seconds for execution..."
    sleep 3
    
    STATUS=$(./bin/client status "$TASK_ID" 2>&1 | grep "State:" | awk '{print $2}')
    echo -e "${GREEN}✅ Task status: $STATUS${NC}"
else
    echo -e "${YELLOW}⚠️  Task submission had issues, but system is running${NC}"
fi

echo ""
echo -e "${GREEN}✅ System is running!${NC}"
echo ""
echo "Next steps:"
echo "  1. View web UI: ./start_web.sh (then open http://localhost:8080)"
echo "  2. Submit tasks: ./bin/client submit echo 'Hello'"
echo "  3. Monitor: ./monitor.sh"
echo "  4. Stop: ./stop_all.sh"

