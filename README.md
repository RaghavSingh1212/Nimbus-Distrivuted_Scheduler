# Nimbus - Mini-Kubernetes Distributed Task Scheduler

A production-ready distributed task scheduler inspired by Kubernetes, built with Go and gRPC. This system demonstrates core distributed systems concepts, including fault tolerance, lease-based task assignment, automatic failure recovery, and priority-based scheduling.

<img width="1365" height="792" alt="Screenshot 2026-06-19 at 3 43 41 PM" src="https://github.com/user-attachments/assets/2a55cb22-dbc9-4a16-a539-301860ed611e" />

## 🏗️ Architecture

```
┌─────────┐         ┌──────────┐         ┌─────────┐
│ Client  │────────▶│  Master  │◀────────│ Worker1 │
└─────────┘         │(Scheduler)│         └─────────┘
                    │           │         ┌─────────┐
                    │  ┌─────┐  │◀────────│ Worker2 │
                    │  │ DB  │  │         └─────────┘
                    │  └─────┘  │         ┌─────────┐
                    │           │◀────────│ Worker3 │
                    └───────────┘         └─────────┘
```

### Components

- **Master (Scheduler)**: Central coordinator that accepts tasks, assigns them to workers, tracks state, and handles failures
- **Workers**: Execute tasks, send heartbeats, and report results
- **Client**: CLI tool for submitting tasks and checking status
- **Database**: SQLite for persistent state storage

## ✨ Features

### Core Features
- ✅ **Task Submission & Scheduling**: Priority-based task queue with worker-pull model
- ✅ **Lease-based Assignment**: Tasks are assigned with time-bound leases for automatic recovery
- ✅ **Failure Detection**: Heartbeat-based worker health monitoring
- ✅ **Automatic Recovery**: Expired leases trigger task requeuing
- ✅ **Retry Logic**: Exponential backoff retries for failed tasks
- ✅ **Resource-aware Scheduling**: CPU and memory constraints
- ✅ **Idempotency**: Exactly-once semantics via idempotency keys
- ✅ **gRPC API**: Type-safe, high-performance RPC

### Task States
- `PENDING`: Waiting to be assigned
- `ASSIGNED`: Assigned to a worker with active lease
- `RUNNING`: Currently executing (optional)
- `SUCCEEDED`: Completed successfully
- `FAILED`: Failed after max retries
- `TIMED_OUT`: Worker died, lease expired

## 🌐 Web UI

A beautiful web dashboard is available! See [WEB_UI.md](WEB_UI.md) for details.

**Quick start:**
```bash
./start_web.sh
# Then open http://localhost:8080 in your browser
```

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- Go plugins: `protoc-gen-go`, `protoc-gen-go-grpc`

### Installation

```bash
# Install protobuf compiler (macOS)
brew install protobuf

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install dependencies
go mod download
```

### Build

```bash
# Generate protobuf code and build all components
make build

# Or build individually
make master
make worker
make client
```

### Run Locally

**Terminal 1 - Master:**
```bash
./bin/master -port 50051 -db ./master.db
```

**Terminal 2 - Worker 1:**
```bash
./bin/worker -master localhost:50051 -addr localhost:50052
```

**Terminal 3 - Worker 2:**
```bash
./bin/worker -master localhost:50051 -addr localhost:50053
```

**Terminal 4 - Client:**
```bash
# Submit a task
./bin/client submit sleep 5 --priority=10

# Check status
./bin/client status task-1234567890

# List all tasks
./bin/client list

# List pending tasks only
./bin/client list --state=PENDING
```

### Run with Docker Compose

```bash
# Start master + 3 workers
docker-compose up --build

# In another terminal, submit tasks
docker-compose exec master /app/client submit echo "Hello, World!"
```

## 📖 Usage Examples

### Submit Tasks

```bash
# Simple command
./bin/client submit echo "Hello from Nimbus"

# With priority and retries
./bin/client submit python script.py --priority=100 --retries=5

# Long-running task
./bin/client submit sleep 30 --priority=50

# Resource-constrained task (requires 2 CPU cores, 1GB RAM)
# Note: Resource constraints are checked but not enforced in current worker implementation
```

### Monitor Tasks

```bash
# Get task status
./bin/client status task-1234567890

# List all tasks
./bin/client list --limit=20

# Filter by state
./bin/client list --state=SUCCEEDED
./bin/client list --state=FAILED
./bin/client list --state=PENDING
```

## 🔧 Configuration

### Master Options
- `-port`: gRPC server port (default: 50051)
- `-db`: Database file path (default: ./master.db)

### Worker Options
- `-master`: Master server address (default: localhost:50051)
- `-addr`: Worker address (default: localhost:50052)

## 🧪 Failure Scenarios & Recovery

### Scenario 1: Worker Dies During Task Execution
1. Worker stops sending heartbeats
2. Master detects worker as DEAD after 12 seconds
3. Master finds tasks with expired leases
4. Tasks are automatically requeued as PENDING
5. Another worker picks up and executes the task

### Scenario 2: Network Partition
1. Worker loses connection to master
2. Heartbeats fail
3. Master marks worker as SUSPECTED (6s) then DEAD (12s)
4. Tasks reassigned to healthy workers

### Scenario 3: Task Failure
1. Worker reports task failure
2. Master increments attempt count
3. If attempts < max_retries: task requeued with exponential backoff
4. If attempts >= max_retries: task marked as FAILED

## 🏛️ Architecture Details

### Lease Mechanism
- Each task assignment includes a **lease expiry time** (default: 10 seconds)
- Workers must complete tasks before lease expires
- If lease expires, master automatically requeues the task
- This prevents tasks from being "stuck forever" if a worker dies

### Priority Queue
- Tasks are ordered by priority (higher = more important)
- Within same priority, FIFO ordering
- In-memory queue for fast access, SQLite for durability

### Failure Detector
- Runs every 2 seconds
- Checks worker last heartbeat time
- States: ALIVE → SUSPECTED (6s) → DEAD (12s)
- Automatically recovers expired task leases

### Retry Logic
- Exponential backoff: `2^attempts` seconds
- Max retries configurable per task
- Failed tasks are requeued with incremented attempt count

## 📊 State Machine

```
Task Lifecycle:
PENDING → ASSIGNED → RUNNING → SUCCEEDED
                ↓
             FAILED (if attempts < max_retries → PENDING)
                ↓
             FAILED (if attempts >= max_retries)

Worker Lifecycle:
ALIVE → SUSPECTED (6s no heartbeat) → DEAD (12s no heartbeat)
```

## 🔐 Idempotency

Tasks can include an `idempotency_key` to ensure exactly-once execution:
- If a task with the same key already completed, returns existing result
- Prevents duplicate execution of the same logical task
- Useful for idempotent operations (e.g., payment processing)

## 🐳 Docker Deployment

The system includes Docker Compose configuration for easy multi-node deployment:

```bash
# Build and start
docker-compose up --build

# Scale workers
docker-compose up --scale worker=5

# View logs
docker-compose logs -f master
docker-compose logs -f worker1
```

## 📝 API Reference

### gRPC Services

#### MasterService
- `RegisterWorker`: Register a new worker
- `Heartbeat`: Worker heartbeat (every 2s)
- `PollTask`: Worker requests a task (pull model)
- `ReportTaskResult`: Worker reports task completion
- `SubmitTask`: Client submits a new task
- `GetTaskStatus`: Get task status by ID
- `ListTasks`: List tasks with optional filtering

See `proto/scheduler.proto` for complete API definitions.

## 🧩 Project Structure

```
nimbus/
├── proto/              # Protocol Buffer definitions
├── master/             # Master (Scheduler) server
├── worker/             # Worker implementation
├── client/             # CLI client
├── internal/
│   ├── db/            # Database layer (SQLite)
│   ├── scheduler/     # Core scheduling logic
│   ├── queue/         # Priority queue
│   └── leases/        # Lease management
├── docker-compose.yml # Multi-node deployment
├── Makefile           # Build automation
└── README.md          # This file
```

## 🎯 Design Decisions

1. **Worker-Pull Model**: Workers request tasks rather than master pushing. Better for scalability and worker autonomy.

2. **SQLite for Persistence**: Simple, embedded database. Easy to replace with PostgreSQL/MySQL for production.

3. **Lease-based Assignment**: Prevents tasks from being stuck if workers die. Industry-standard approach (used by Kubernetes, Mesos).

4. **Priority Queue**: In-memory for speed, SQLite for durability. Trade-off between performance and consistency.

5. **gRPC over REST**: Better performance, type safety, streaming support.

## 🚧 Future Enhancements

- [ ] Preemption: High-priority tasks can interrupt low-priority ones
- [ ] Resource bin-packing: Better resource utilization
- [ ] Task dependencies: DAG-based scheduling
- [ ] Metrics & Observability: Prometheus integration
- [ ] Web UI: Dashboard for monitoring
- [ ] Multi-master: Leader election for high availability
- [ ] Task affinity: Schedule tasks on specific workers

## 📄 License

MIT License - feel free to use this for learning, projects, or production!

## 🙏 Acknowledgments

Inspired by Kubernetes scheduler, Apache Mesos, and distributed systems research.

---

**Built with ❤️ using Go, gRPC, and distributed systems principles**

