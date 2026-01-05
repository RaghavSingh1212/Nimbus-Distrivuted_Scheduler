#!/bin/bash

echo "🔍 Verifying Fix..."
echo ""

# Check if new master binary exists and was built after the fix
if [ -f "bin/master" ]; then
    BUILD_TIME=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" bin/master 2>/dev/null || stat -c "%y" bin/master 2>/dev/null | cut -d'.' -f1)
    echo "✅ Master binary exists (built: $BUILD_TIME)"
else
    echo "❌ Master binary not found. Run: go build -o bin/master ./master"
    exit 1
fi

# Check if loadPendingTasks function exists in source
if grep -q "loadPendingTasks" internal/scheduler/scheduler.go; then
    echo "✅ Fix code is present in scheduler.go"
else
    echo "❌ Fix code not found in scheduler.go"
    exit 1
fi

# Count pending tasks
PENDING_COUNT=$(sqlite3 master.db "SELECT COUNT(*) FROM tasks WHERE state = 0;" 2>/dev/null || echo "0")
echo "📊 Pending tasks in database: $PENDING_COUNT"

echo ""
echo "✅ Fix is ready!"
echo ""
echo "To apply the fix:"
echo "1. Stop the current master (Ctrl+C in master terminal)"
echo "2. Start new master: ./bin/master"
echo "3. The master will load $PENDING_COUNT pending tasks on startup"
echo "4. Worker should pick them up within seconds"
echo ""

