# How to Run Nimbus

## Quick Start (Easiest Way)

```bash
# Run the setup script (one-time setup)
./run.sh

# Then in separate terminals:
# Terminal 1: Master
./bin/master

# Terminal 2: Worker  
./bin/worker -master localhost:50051

# Terminal 3: Client
./bin/client submit echo "Hello, World!"
./bin/client list
```

## Step-by-Step Instructions

### Step 1: Install Prerequisites

**Install Go** (if not already installed):
- Download from: https://golang.org/dl/
- Or on macOS: `brew install go`

**Install Protocol Buffers**:
```bash
# macOS
brew install protobuf

# Linux (Ubuntu/Debian)
sudo apt-get install protobuf-compiler

# Verify installation
protoc --version
```

**Install Go protobuf plugins**:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Step 2: Build the Project

**Option A: Use the run script (recommended)**
```bash
./run.sh
```

**Option B: Manual build**
```bash
# Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/scheduler.proto

# Download dependencies
go mod download

# Build all components
go build -o bin/master ./master
go build -o bin/worker ./worker
go build -o bin/client ./client
```

**Option C: Use Makefile**
```bash
make build
```

### Step 3: Run the System

You need **3 terminal windows**:

#### Terminal 1: Start Master
```bash
./bin/master
```

You should see:
```
Master server listening on :50051
```

#### Terminal 2: Start Worker
```bash
./bin/worker -master localhost:50051
```

You should see:
```
Worker <worker-id> started, connected to master at localhost:50051
```

#### Terminal 3: Use Client
```bash
# Submit a task
./bin/client submit echo "Hello, Nimbus!"

# Check task status (replace with actual task ID from above)
./bin/client status task-1234567890

# List all tasks
./bin/client list

# List only pending tasks
./bin/client list --state=PENDING
```

## Example Session

```bash
# Terminal 1
$ ./bin/master
Master server listening on :50051

# Terminal 2  
$ ./bin/worker -master localhost:50051
Worker worker-1234567890 started, connected to master at localhost:50051
Executing task task-1234567890: echo [Hello, Nimbus!]
Task task-1234567890 succeeded

# Terminal 3
$ ./bin/client submit echo "Hello, Nimbus!" --priority=10
Task submitted: task-1234567890

$ ./bin/client status task-1234567890
Task ID: task-1234567890
State: SUCCEEDED
Command: echo [Hello, Nimbus!]
Priority: 10
Attempts: 0/3
Output:
Hello, Nimbus!
```

## Running Multiple Workers

You can start multiple workers for parallel task execution:

```bash
# Terminal 2: Worker 1
./bin/worker -master localhost:50051 -addr localhost:50052

# Terminal 3: Worker 2
./bin/worker -master localhost:50051 -addr localhost:50053

# Terminal 4: Worker 3
./bin/worker -master localhost:50051 -addr localhost:50054
```

## Using Docker Compose

If you prefer Docker:

```bash
# Build and start master + 3 workers
docker-compose up --build

# In another terminal, you can exec into master to use client
# (Note: client binary needs to be built separately or use docker exec)
```

## Demo Script

For an automated demo:

```bash
./demo.sh
```

This will:
- Start master and 3 workers
- Submit 10 tasks
- Show their status
- Demonstrate the system working

## Common Commands

### Submit Tasks
```bash
# Simple command
./bin/client submit echo "Hello"

# With priority (higher = more important)
./bin/client submit sleep 5 --priority=100

# With custom retries
./bin/client submit python script.py --retries=5

# Complex command
./bin/client submit python -c "print('Hello from Python')" --priority=50
```

### Check Status
```bash
# Get detailed status
./bin/client status <task_id>

# List all tasks
./bin/client list

# List with limit
./bin/client list --limit=20

# Filter by state
./bin/client list --state=SUCCEEDED
./bin/client list --state=FAILED
./bin/client list --state=PENDING
```

## Testing Failure Recovery

1. Start master and worker as above
2. Submit a long-running task:
   ```bash
   ./bin/client submit sleep 30 --priority=10
   ```
3. Kill the worker (Ctrl+C in worker terminal)
4. Wait 10-12 seconds
5. Start a new worker
6. Check task status - it should be reassigned and completed

## Troubleshooting

**"command not found: protoc"**
- Install protobuf: `brew install protobuf` (macOS) or `sudo apt-get install protobuf-compiler` (Linux)

**"connection refused"**
- Make sure master is running before starting workers
- Check the port (default: 50051)

**"cannot find package" errors**
- Run: `go mod download` and `go mod tidy`

**Build errors about missing protobuf files**
- Generate protobuf code: `make proto` or run `./run.sh`

**Task stuck in ASSIGNED state**
- Worker may have died - wait 10-12 seconds for lease expiry
- Master will automatically requeue the task

## Next Steps

- Read [README.md](README.md) for architecture details
- Try different task types and priorities
- Experiment with failure scenarios
- Check out the code to understand how it works!

