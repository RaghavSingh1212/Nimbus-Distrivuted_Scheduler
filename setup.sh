#!/bin/bash

set -e

echo "🚀 Setting up Nimbus Distributed Task Scheduler..."

# Check for Go
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ from https://golang.org"
    exit 1
fi

# Check for protoc
if ! command -v protoc &> /dev/null; then
    echo "❌ protoc is not installed."
    echo "Install with:"
    echo "  macOS: brew install protobuf"
    echo "  Linux: apt-get install protobuf-compiler"
    echo "  Or download from: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Check for protoc-gen-go
if ! command -v protoc-gen-go &> /dev/null; then
    echo "📦 Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

# Check for protoc-gen-go-grpc
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "📦 Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Download dependencies
echo "📥 Downloading dependencies..."
go mod download

# Generate protobuf code
echo "🔨 Generating protobuf code..."
mkdir -p bin
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/scheduler.proto

# Build all components
echo "🔨 Building components..."
make build

echo "✅ Setup complete!"
echo ""
echo "To start the system:"
echo "  1. Terminal 1: ./bin/master"
echo "  2. Terminal 2: ./bin/worker -master localhost:50051"
echo "  3. Terminal 3: ./bin/client submit echo 'Hello, World!'"

