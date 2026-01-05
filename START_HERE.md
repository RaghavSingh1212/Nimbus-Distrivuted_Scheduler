# 🚀 Quick Start Guide

## Step 1: Start the Master (Terminal 1)

```bash
cd /Users/raghavsingh/Desktop/DistributedSch
./bin/master
```

Leave this running. You should see:
```
Master server listening on :50051
```

## Step 2: Start a Worker (Terminal 2 - NEW TERMINAL)

Open a **new terminal window** and run:

```bash
cd /Users/raghavsingh/Desktop/DistributedSch
./bin/worker -master localhost:50051
```

Leave this running. You should see:
```
Worker worker-XXXXX started, connected to master at localhost:50051
```

## Step 3: Submit Tasks (Terminal 3 - NEW TERMINAL)

Open **another new terminal window** and run:

```bash
cd /Users/raghavsingh/Desktop/DistributedSch

# Submit a simple task
./bin/client submit echo "Hello, Nimbus!"

# List all tasks
./bin/client list

# Check status (replace with actual task ID from above)
./bin/client status task-1234567890
```

## Example Commands

```bash
# Submit tasks with different priorities
./bin/client submit echo "High priority" --priority=100
./bin/client submit sleep 3 --priority=50
./bin/client submit echo "Low priority" --priority=10

# List tasks
./bin/client list

# Filter by state
./bin/client list --state=PENDING
./bin/client list --state=SUCCEEDED
```

## Troubleshooting

**If you see "connection refused":**
- Make sure master is running first (Step 1)
- Wait a few seconds after starting master before starting worker

**If commands hang:**
- Press Ctrl+C to cancel
- Make sure master and worker are both running

