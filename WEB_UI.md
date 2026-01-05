# 🌐 Nimbus Web UI

A beautiful, modern web interface for the Nimbus Distributed Task Scheduler.

## Features

- 📊 **Real-time Dashboard**: Live statistics of task states
- 📝 **Task Submission**: Easy-to-use form to submit new tasks
- 📋 **Task List**: View all tasks with their current status
- 🔄 **Auto-refresh**: Automatically updates every 2 seconds
- 🎨 **Modern UI**: Beautiful gradient design with smooth animations

## Quick Start

### 1. Start the Master (if not already running)
```bash
./bin/master
```

### 2. Start the Web UI
```bash
./start_web.sh
```

Or manually:
```bash
./bin/web -master localhost:50051 -port 8080
```

### 3. Open in Browser
Navigate to: **http://localhost:8080**

## Usage

### Submit a Task
1. Fill in the form:
   - **Command**: e.g., `echo`, `sleep`, `python`
   - **Arguments**: Comma-separated (e.g., `Hello World, 5`)
   - **Priority**: Higher number = higher priority (0-1000)
   - **Max Retries**: Number of retry attempts (0-10)

2. Click "Submit Task"

### View Tasks
- Tasks are automatically displayed in the task list
- Each task shows:
  - Task ID
  - Current state (color-coded)
  - Command and arguments
  - Priority and attempt count
  - Creation time
  - Output/error (if available)

### Statistics
The dashboard shows real-time counts:
- **Pending**: Tasks waiting to be assigned
- **Assigned**: Tasks assigned to workers
- **Running**: Tasks currently executing
- **Succeeded**: Completed tasks
- **Failed**: Failed tasks

## API Endpoints

The web server also exposes REST API endpoints:

- `GET /api/tasks` - Get list of all tasks
- `POST /api/submit` - Submit a new task
  ```json
  {
    "command": "echo",
    "args": ["Hello"],
    "priority": 10,
    "max_retries": 3
  }
  ```

## Configuration

```bash
# Custom master address
./bin/web -master localhost:50051 -port 8080

# Different port
./bin/web -master localhost:50051 -port 3000
```

## Troubleshooting

**"Connection refused"**
- Make sure master is running on port 50051
- Check: `lsof -i :50051`

**Tasks not updating**
- Check browser console for errors
- Verify master is responding: `./bin/client list`

**Web server won't start**
- Check if port 8080 is already in use
- Try a different port: `-port 3000`

