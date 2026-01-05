package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	pb "github.com/distributed-scheduler/nimbus/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type WebServer struct {
	masterAddr string
	client     pb.MasterServiceClient
	conn       *grpc.ClientConn
}

func NewWebServer(masterAddr string) (*WebServer, error) {
	conn, err := grpc.Dial(masterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}

	return &WebServer{
		masterAddr: masterAddr,
		client:     pb.NewMasterServiceClient(conn),
		conn:       conn,
	}, nil
}

func (ws *WebServer) Close() {
	ws.conn.Close()
}

func (ws *WebServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nimbus - Distributed Task Scheduler</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        
        :root {
            --primary: #6366f1;
            --primary-dark: #4f46e5;
            --secondary: #8b5cf6;
            --success: #10b981;
            --warning: #f59e0b;
            --danger: #ef4444;
            --info: #3b82f6;
            --bg: #0f172a;
            --bg-card: #1e293b;
            --bg-hover: #334155;
            --text: #f1f5f9;
            --text-muted: #94a3b8;
            --border: #334155;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: var(--bg);
            color: var(--text);
            min-height: 100vh;
            padding: 20px;
            line-height: 1.6;
        }
        
        .container {
            max-width: 1600px;
            margin: 0 auto;
        }
        
        .header {
            background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
            border-radius: 16px;
            padding: 40px;
            margin-bottom: 30px;
            box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.3);
            animation: slideDown 0.5s ease-out;
        }
        
        @keyframes slideDown {
            from { transform: translateY(-20px); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
        }
        
        h1 {
            font-size: 3em;
            font-weight: 800;
            margin-bottom: 10px;
            text-shadow: 0 2px 4px rgba(0,0,0,0.2);
        }
        
        .subtitle {
            font-size: 1.2em;
            opacity: 0.9;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        
        .stat-card {
            background: var(--bg-card);
            border-radius: 12px;
            padding: 24px;
            border: 1px solid var(--border);
            transition: all 0.3s ease;
            animation: fadeIn 0.6s ease-out;
            animation-fill-mode: both;
        }
        
        .stat-card:nth-child(1) { animation-delay: 0.1s; }
        .stat-card:nth-child(2) { animation-delay: 0.2s; }
        .stat-card:nth-child(3) { animation-delay: 0.3s; }
        .stat-card:nth-child(4) { animation-delay: 0.4s; }
        .stat-card:nth-child(5) { animation-delay: 0.5s; }
        
        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }
        
        .stat-card:hover {
            transform: translateY(-4px);
            box-shadow: 0 10px 20px rgba(99, 102, 241, 0.2);
            border-color: var(--primary);
        }
        
        .stat-value {
            font-size: 2.5em;
            font-weight: 700;
            background: linear-gradient(135deg, var(--primary), var(--secondary));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 8px;
        }
        
        .stat-label {
            color: var(--text-muted);
            font-size: 0.9em;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            font-weight: 600;
        }
        
        .main-grid {
            display: grid;
            grid-template-columns: 400px 1fr;
            gap: 30px;
        }
        
        @media (max-width: 1200px) {
            .main-grid {
                grid-template-columns: 1fr;
            }
        }
        
        .panel {
            background: var(--bg-card);
            border-radius: 16px;
            padding: 30px;
            border: 1px solid var(--border);
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        
        .panel h2 {
            color: var(--text);
            margin-bottom: 24px;
            font-size: 1.5em;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        
        .task-form {
            display: flex;
            flex-direction: column;
            gap: 20px;
        }
        
        .form-group {
            display: flex;
            flex-direction: column;
        }
        
        label {
            color: var(--text-muted);
            margin-bottom: 8px;
            font-weight: 500;
            font-size: 0.9em;
        }
        
        input, select {
            padding: 12px 16px;
            background: var(--bg);
            border: 1px solid var(--border);
            border-radius: 8px;
            color: var(--text);
            font-size: 1em;
            transition: all 0.2s;
        }
        
        input:focus, select:focus {
            outline: none;
            border-color: var(--primary);
            box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
        }
        
        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }
        
        button {
            background: linear-gradient(135deg, var(--primary) 0%, var(--secondary) 100%);
            color: white;
            border: none;
            padding: 14px 28px;
            border-radius: 8px;
            font-size: 1em;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            box-shadow: 0 4px 6px rgba(99, 102, 241, 0.3);
        }
        
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 12px rgba(99, 102, 241, 0.4);
        }
        
        button:active {
            transform: translateY(0);
        }
        
        .refresh-btn {
            background: var(--success);
            margin-bottom: 15px;
            width: 100%;
        }
        
        .task-list {
            max-height: 700px;
            overflow-y: auto;
            padding-right: 10px;
        }
        
        .task-list::-webkit-scrollbar {
            width: 8px;
        }
        
        .task-list::-webkit-scrollbar-track {
            background: var(--bg);
            border-radius: 4px;
        }
        
        .task-list::-webkit-scrollbar-thumb {
            background: var(--border);
            border-radius: 4px;
        }
        
        .task-list::-webkit-scrollbar-thumb:hover {
            background: var(--primary);
        }
        
        .task-item {
            background: var(--bg);
            border-left: 4px solid var(--primary);
            padding: 20px;
            margin-bottom: 15px;
            border-radius: 12px;
            transition: all 0.3s;
            border: 1px solid var(--border);
            animation: slideIn 0.3s ease-out;
        }
        
        @keyframes slideIn {
            from { opacity: 0; transform: translateX(-10px); }
            to { opacity: 1; transform: translateX(0); }
        }
        
        .task-item:hover {
            transform: translateX(5px);
            border-color: var(--primary);
            box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2);
        }
        
        .task-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 12px;
        }
        
        .task-id {
            font-family: 'Monaco', 'Courier New', monospace;
            font-size: 0.85em;
            color: var(--text-muted);
            background: var(--bg-card);
            padding: 4px 8px;
            border-radius: 4px;
        }
        
        .task-state {
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
        
        .state-PENDING { background: rgba(245, 158, 11, 0.2); color: #fbbf24; border: 1px solid rgba(245, 158, 11, 0.3); }
        .state-ASSIGNED { background: rgba(59, 130, 246, 0.2); color: #60a5fa; border: 1px solid rgba(59, 130, 246, 0.3); }
        .state-RUNNING { background: rgba(16, 185, 129, 0.2); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
        .state-SUCCEEDED { background: rgba(16, 185, 129, 0.2); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.3); }
        .state-FAILED { background: rgba(239, 68, 68, 0.2); color: #f87171; border: 1px solid rgba(239, 68, 68, 0.3); }
        
        .task-command {
            font-family: 'Monaco', 'Courier New', monospace;
            color: var(--text);
            margin: 10px 0;
            font-size: 1.1em;
            font-weight: 500;
            background: var(--bg-card);
            padding: 12px;
            border-radius: 6px;
        }
        
        .task-meta {
            display: flex;
            gap: 20px;
            font-size: 0.85em;
            color: var(--text-muted);
            margin-top: 12px;
            flex-wrap: wrap;
        }
        
        .task-meta span {
            display: flex;
            align-items: center;
            gap: 5px;
        }
        
        .auto-refresh {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-top: 15px;
            padding: 12px;
            background: var(--bg);
            border-radius: 8px;
        }
        
        .auto-refresh input[type="checkbox"] {
            width: 20px;
            height: 20px;
            cursor: pointer;
            accent-color: var(--primary);
        }
        
        .loading {
            text-align: center;
            padding: 40px;
            color: var(--text-muted);
        }
        
        .spinner {
            border: 3px solid var(--border);
            border-top: 3px solid var(--primary);
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }
        
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        
        .empty-state {
            text-align: center;
            padding: 60px 20px;
            color: var(--text-muted);
        }
        
        .empty-state svg {
            width: 80px;
            height: 80px;
            margin-bottom: 20px;
            opacity: 0.5;
        }
        
        .output-box {
            margin-top: 12px;
            padding: 12px;
            background: var(--bg-card);
            border-radius: 6px;
            font-size: 0.9em;
            font-family: 'Monaco', 'Courier New', monospace;
            white-space: pre-wrap;
            word-break: break-all;
        }
        
        .output-box.success {
            border-left: 3px solid var(--success);
        }
        
        .output-box.error {
            border-left: 3px solid var(--danger);
            color: #f87171;
        }
        
        .filter-bar {
            display: flex;
            gap: 10px;
            margin-bottom: 20px;
            flex-wrap: wrap;
        }
        
        .filter-btn {
            padding: 8px 16px;
            background: var(--bg);
            border: 1px solid var(--border);
            color: var(--text-muted);
            border-radius: 20px;
            cursor: pointer;
            transition: all 0.2s;
            font-size: 0.9em;
        }
        
        .filter-btn.active {
            background: var(--primary);
            color: white;
            border-color: var(--primary);
        }
        
        .filter-btn:hover {
            border-color: var(--primary);
            color: var(--text);
        }
        
        .status-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            margin-right: 8px;
            animation: pulse 2s infinite;
        }
        
        .status-indicator.connected {
            background: var(--success);
        }
        
        .status-indicator.disconnected {
            background: var(--danger);
        }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚡ Nimbus</h1>
            <p class="subtitle">Distributed Task Scheduler Dashboard</p>
            <div style="margin-top: 15px;">
                <span class="status-indicator connected" id="statusIndicator"></span>
                <span id="statusText">Connected to Master</span>
            </div>
        </div>

        <div class="stats-grid" id="stats">
            <div class="stat-card">
                <div class="stat-value" id="stat-pending">-</div>
                <div class="stat-label">Pending</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-assigned">-</div>
                <div class="stat-label">Assigned</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-running">-</div>
                <div class="stat-label">Running</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-succeeded">-</div>
                <div class="stat-label">Succeeded</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="stat-failed">-</div>
                <div class="stat-label">Failed</div>
            </div>
        </div>

        <div class="main-grid">
            <div class="panel">
                <h2>📝 Submit Task</h2>
                <form class="task-form" id="taskForm" onsubmit="submitTask(event)">
                    <div class="form-group">
                        <label for="command">Command</label>
                        <input type="text" id="command" name="command" placeholder="e.g., echo, sleep, python" required>
                    </div>
                    <div class="form-group">
                        <label for="args">Arguments (comma-separated)</label>
                        <input type="text" id="args" name="args" placeholder="e.g., Hello World, 5">
                    </div>
                    <div class="form-row">
                        <div class="form-group">
                            <label for="priority">Priority</label>
                            <input type="number" id="priority" name="priority" value="0" min="0" max="1000">
                        </div>
                        <div class="form-group">
                            <label for="retries">Max Retries</label>
                            <input type="number" id="retries" name="retries" value="3" min="0" max="10">
                        </div>
                    </div>
                    <button type="submit">🚀 Submit Task</button>
                </form>
            </div>

            <div class="panel">
                <h2>📋 Tasks</h2>
                <div class="filter-bar">
                    <button class="filter-btn active" onclick="setFilter('ALL')">All</button>
                    <button class="filter-btn" onclick="setFilter('PENDING')">Pending</button>
                    <button class="filter-btn" onclick="setFilter('ASSIGNED')">Assigned</button>
                    <button class="filter-btn" onclick="setFilter('RUNNING')">Running</button>
                    <button class="filter-btn" onclick="setFilter('SUCCEEDED')">Succeeded</button>
                    <button class="filter-btn" onclick="setFilter('FAILED')">Failed</button>
                </div>
                <button class="refresh-btn" onclick="loadTasks()">🔄 Refresh</button>
                <div class="auto-refresh">
                    <input type="checkbox" id="autoRefresh" checked onchange="toggleAutoRefresh()">
                    <label for="autoRefresh">Auto-refresh (2s)</label>
                </div>
                <div class="task-list" id="taskList">
                    <div class="loading">
                        <div class="spinner"></div>
                        Loading tasks...
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        let autoRefreshInterval = null;
        let currentFilter = 'ALL';

        function setFilter(filter) {
            currentFilter = filter;
            document.querySelectorAll('.filter-btn').forEach(btn => {
                btn.classList.remove('active');
            });
            event.target.classList.add('active');
            loadTasks();
        }

        function toggleAutoRefresh() {
            const checkbox = document.getElementById('autoRefresh');
            if (checkbox.checked) {
                autoRefreshInterval = setInterval(loadTasks, 2000);
            } else {
                if (autoRefreshInterval) {
                    clearInterval(autoRefreshInterval);
                    autoRefreshInterval = null;
                }
            }
        }

        function updateStats(tasks) {
            const stats = {
                PENDING: 0,
                ASSIGNED: 0,
                RUNNING: 0,
                SUCCEEDED: 0,
                FAILED: 0
            };

            tasks.forEach(task => {
                const state = task.state || 'PENDING';
                if (stats[state] !== undefined) {
                    stats[state]++;
                }
            });

            document.getElementById('stat-pending').textContent = stats.PENDING;
            document.getElementById('stat-assigned').textContent = stats.ASSIGNED;
            document.getElementById('stat-running').textContent = stats.RUNNING;
            document.getElementById('stat-succeeded').textContent = stats.SUCCEEDED;
            document.getElementById('stat-failed').textContent = stats.FAILED;
        }

        function formatTask(task) {
            const state = task.state || 'PENDING';
            const args = task.spec?.args || [];
            const argsStr = args.length > 0 ? args.join(' ') : '';
            const command = task.spec?.command || 'unknown';
            const fullCommand = argsStr ? command + ' ' + argsStr : command;
            
            const createdAt = task.createdAt ? new Date(task.createdAt / 1000000).toLocaleString() : 'Unknown';
            
            let html = '<div class="task-item">';
            html += '<div class="task-header">';
            html += '<span class="task-id">' + (task.taskId || 'N/A') + '</span>';
            html += '<span class="task-state state-' + state + '">' + state + '</span>';
            html += '</div>';
            html += '<div class="task-command">' + fullCommand + '</div>';
            html += '<div class="task-meta">';
            html += '<span>🎯 Priority: ' + (task.spec?.priority || 0) + '</span>';
            html += '<span>🔄 Attempts: ' + (task.attempts || 0) + '/' + (task.spec?.maxRetries || 3) + '</span>';
            html += '<span>🕐 ' + createdAt + '</span>';
            if (task.assignedWorkerId) {
                html += '<span>👷 Worker: ' + task.assignedWorkerId.substring(0, 12) + '...</span>';
            }
            html += '</div>';
            if (task.output) {
                html += '<div class="output-box success"><strong>Output:</strong><br>' + escapeHtml(task.output) + '</div>';
            }
            if (task.error) {
                html += '<div class="output-box error"><strong>Error:</strong><br>' + escapeHtml(task.error) + '</div>';
            }
            html += '</div>';
            return html;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        async function loadTasks() {
            try {
                const response = await fetch('/api/tasks');
                if (!response.ok) throw new Error('Failed to fetch');
                
                const data = await response.json();
                
                if (data.tasks) {
                    updateStats(data.tasks);
                    
                    // Apply filter
                    let filteredTasks = data.tasks;
                    if (currentFilter !== 'ALL') {
                        filteredTasks = data.tasks.filter(t => (t.state || 'PENDING') === currentFilter);
                    }
                    
                    const taskList = document.getElementById('taskList');
                    if (filteredTasks.length === 0) {
                        taskList.innerHTML = '<div class="empty-state"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor"><path d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg><div>No tasks found</div></div>';
                    } else {
                        taskList.innerHTML = filteredTasks.map(formatTask).join('');
                    }
                    
                    // Update status
                    document.getElementById('statusIndicator').className = 'status-indicator connected';
                    document.getElementById('statusText').textContent = 'Connected to Master';
                }
            } catch (error) {
                console.error('Error loading tasks:', error);
                document.getElementById('taskList').innerHTML = 
                    '<div class="empty-state" style="color: #f87171;"><strong>Error loading tasks</strong><br>Make sure master is running</div>';
                document.getElementById('statusIndicator').className = 'status-indicator disconnected';
                document.getElementById('statusText').textContent = 'Disconnected';
            }
        }

        async function submitTask(event) {
            event.preventDefault();
            
            const formData = new FormData(event.target);
            const command = formData.get('command');
            const argsStr = formData.get('args') || '';
            const args = argsStr ? argsStr.split(',').map(s => s.trim()).filter(s => s) : [];
            const priority = parseInt(formData.get('priority')) || 0;
            const retries = parseInt(formData.get('retries')) || 3;

            const submitBtn = event.target.querySelector('button[type="submit"]');
            const originalText = submitBtn.textContent;
            submitBtn.textContent = '⏳ Submitting...';
            submitBtn.disabled = true;

            try {
                const response = await fetch('/api/submit', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        command: command,
                        args: args,
                        priority: priority,
                        max_retries: retries
                    })
                });

                const data = await response.json();
                
                if (data.task_id) {
                    submitBtn.textContent = '✅ Submitted!';
                    setTimeout(() => {
                        submitBtn.textContent = originalText;
                        submitBtn.disabled = false;
                    }, 2000);
                    event.target.reset();
                    loadTasks();
                } else {
                    alert('Error: ' + (data.error || 'Failed to submit task'));
                    submitBtn.textContent = originalText;
                    submitBtn.disabled = false;
                }
            } catch (error) {
                alert('Error submitting task: ' + error.message);
                submitBtn.textContent = originalText;
                submitBtn.disabled = false;
            }
        }

        // Initialize
        loadTasks();
        toggleAutoRefresh();
    </script>
</body>
</html>`

	t, _ := template.New("index").Parse(tmpl)
	t.Execute(w, nil)
}

func (ws *WebServer) handleAPITasks(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := getContext(5 * time.Second)
	defer cancel()

	tasks, err := ws.client.ListTasks(ctx, &pb.ListTasksRequest{
		Limit: 100,
	})

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks.Tasks,
	})
}

func (ws *WebServer) handleAPISubmit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command    string   `json:"command"`
		Args       []string `json:"args"`
		Priority   int32    `json:"priority"`
		MaxRetries int32    `json:"max_retries"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	ctx, cancel := getContext(10 * time.Second)
	defer cancel()

	spec := &pb.TaskSpec{
		Command:    req.Command,
		Args:       req.Args,
		Priority:   req.Priority,
		MaxRetries: req.MaxRetries,
	}

	resp, err := ws.client.SubmitTask(ctx, &pb.SubmitTaskRequest{Spec: spec})

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"task_id": resp.TaskId,
	})
}

func getContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func main() {
	masterAddr := flag.String("master", "localhost:50051", "Master server address")
	port := flag.String("port", "8080", "Web server port")
	flag.Parse()

	ws, err := NewWebServer(*masterAddr)
	if err != nil {
		log.Fatalf("Failed to create web server: %v", err)
	}
	defer ws.Close()

	http.HandleFunc("/", ws.handleIndex)
	http.HandleFunc("/api/tasks", ws.handleAPITasks)
	http.HandleFunc("/api/submit", ws.handleAPISubmit)

	log.Printf("🌐 Web UI starting on http://localhost:%s", *port)
	log.Printf("📡 Connected to master at %s", *masterAddr)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
