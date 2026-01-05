#!/bin/bash

echo "🔄 Restarting Master with Fix..."
echo ""

# Find and kill old master
MASTER_PID=$(ps aux | grep "./bin/master" | grep -v grep | awk '{print $2}')

if [ ! -z "$MASTER_PID" ]; then
    echo "Stopping old master (PID: $MASTER_PID)..."
    kill $MASTER_PID
    sleep 2
    echo "✅ Old master stopped"
else
    echo "No master process found"
fi

echo ""
echo "Starting new master..."
echo "The master will now load pending tasks from the database!"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Start new master
./bin/master

