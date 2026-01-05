# Quick Start Guide

## Prerequisites

1. **Install Go** (1.21+): https://golang.org/dl/
2. **Install Protocol Buffers**:
   ```bash
   # macOS
   brew install protobuf
   
   # Linux (Ubuntu/Debian)
   sudo apt-get install protobuf-compiler
   
   # Or download from: https://grpc.io/docs/protoc-installation/
   ```

3. **Install Go plugins**:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

## Setup (One-time)

```bash
# Run setup script
./setup.sh

# Or manually:
make build
```

## Running the System

### Option 1: Manual (3 terminals)

**Terminal 1 - Master:**
```bash
./bin/master
```

**Terminal 2 - Worker:**
```bash
./bin/worker -master localhost:50051
```

**Terminal 3 - Client:**
```bash
# Submit a task
./bin/client submit echo "Hello, Nimbus!"

# Check status (replace with actual task ID)
./bin/client status task-1234567890

# List tasks
./bin/client list
```

### Option 2: Docker Compose

```bash
# Start master + 3 workers
docker-compose up --build

# In another terminal, use the client
# (You'll need to build the client locally or use docker exec)
```

### Option 3: Demo Script

```bash
./demo.sh
```

## Example Workflow

1. **Start Master:**
   ```bash
   ./bin/master -port 50051
   ```

2. **Start Multiple Workers:**
   ```bash
   # Terminal 2
   ./bin/worker -master localhost:50051 -addr localhost:50052
   
   # Terminal 3
   ./bin/worker -master localhost:50051 -addr localhost:50053
   ```

3. **Submit Tasks:**
   ```bash
   ./bin/client submit sleep 5 --priority=10
   ./bin/client submit echo "Task 2" --priority=5
   ./bin/client submit python -c "print('Hello')" --priority=8
   ```

4. **Monitor Tasks:**
   ```bash
   # List all tasks
   ./bin/client list
   
   # List pending tasks
   ./bin/client list --state=PENDING
   
   # Check specific task
   ./bin/client status task-1234567890
   ```

5. **Test Failure Recovery:**
   - Submit a long-running task: `./bin/client submit sleep 30`
   - Kill the worker executing it (Ctrl+C)
   - Watch the master automatically reassign the task to another worker

## Common Commands

```bash
# Submit task with priority
./bin/client submit <command> [args...] --priority=N --retries=N

# Get task status
./bin/client status <task_id>

# List tasks
./bin/client list [--state=STATE] [--limit=N]

# Available states: PENDING, ASSIGNED, RUNNING, SUCCEEDED, FAILED
```

## Troubleshooting

**"command not found: protoc"**
- Install protobuf compiler (see Prerequisites)

**"connection refused"**
- Make sure master is running before starting workers
- Check the port (default: 50051)

**"task stuck in ASSIGNED state"**
- Worker may have died - wait 10-12 seconds for lease expiry
- Master will automatically requeue the task

**Build errors**
- Run `go mod tidy` to fix dependencies
- Make sure protobuf code is generated: `make proto`

## Next Steps

- Read the full [README.md](README.md) for architecture details
- Explore the code in `internal/scheduler/` to understand the scheduling logic
- Try the demo script: `./demo.sh`
- Experiment with failure scenarios (kill workers, network partitions)

