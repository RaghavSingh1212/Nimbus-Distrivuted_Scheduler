#!/bin/bash

# Quick run script for Nimbus Distributed Scheduler

set -e

echo "🚀 Nimbus Distributed Task Scheduler"
echo "======================================"
echo ""

# Check prerequisites
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ from https://golang.org"
    exit 1
fi

if ! command -v protoc &> /dev/null; then
    echo "❌ protoc is not installed."
    echo "Install with: brew install protobuf (macOS) or apt-get install protobuf-compiler (Linux)"
    exit 1
fi

# Check if protobuf code exists
if [ ! -f "proto/scheduler.pb.go" ]; then
    echo "📦 Generating protobuf code..."
    
    # Install protoc plugins if needed
    if ! command -v protoc-gen-go &> /dev/null; then
        echo "Installing protoc-gen-go..."
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    fi
    
    if ! command -v protoc-gen-go-grpc &> /dev/null; then
        echo "Installing protoc-gen-go-grpc..."
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    fi
    
    # Generate code
    mkdir -p bin
    protoc --go_out=. --go_opt=paths=source_relative \
        --go-grpc_out=. --go-grpc_opt=paths=source_relative \
        proto/scheduler.proto
    
    echo "✅ Protobuf code generated"
fi

# Download dependencies
echo "📥 Downloading dependencies..."
go mod download

# Build binaries
echo "🔨 Building binaries..."
go build -o bin/master ./master
go build -o bin/worker ./worker
go build -o bin/client ./client

echo "✅ Build complete!"
echo ""
echo "📋 To run the system:"
echo ""
echo "1. Start Master (Terminal 1):"
echo "   ./bin/master"
echo ""
echo "2. Start Worker (Terminal 2):"
echo "   ./bin/worker -master localhost:50051"
echo ""
echo "3. Submit tasks (Terminal 3):"
echo "   ./bin/client submit echo 'Hello, Nimbus!'"
echo "   ./bin/client list"
echo ""
echo "Or use the demo script: ./demo.sh"

