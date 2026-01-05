#!/bin/bash

echo "🛑 Stopping Nimbus System..."

pkill -f "./bin/master"
pkill -f "./bin/worker"
pkill -f "./bin/web"

sleep 1

echo "✅ All processes stopped"

