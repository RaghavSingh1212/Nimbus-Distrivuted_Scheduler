package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func NewDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		return nil, err
	}

	return db, nil
}

func (d *DB) initSchema() error {
	// Tasks table
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			state INTEGER NOT NULL,
			command TEXT NOT NULL,
			args TEXT,
			priority INTEGER NOT NULL DEFAULT 0,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			assigned_worker_id TEXT,
			lease_expiry_time INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			output TEXT,
			error TEXT,
			cpu_needed INTEGER DEFAULT 0,
			mem_needed INTEGER DEFAULT 0,
			idempotency_key TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Workers table
	_, err = d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS workers (
			worker_id TEXT PRIMARY KEY,
			addr TEXT NOT NULL,
			last_heartbeat INTEGER NOT NULL,
			cpu_free INTEGER DEFAULT 0,
			mem_free INTEGER DEFAULT 0,
			state INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}

	// Completed tasks table (for exactly-once semantics)
	_, err = d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS completed_tasks (
			idempotency_key TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			completed_at INTEGER NOT NULL,
			result TEXT
		)
	`)
	if err != nil {
		return err
	}

	// Indexes for performance
	_, err = d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_state ON tasks(state)`)
	if err != nil {
		return err
	}

	_, err = d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority DESC)`)
	if err != nil {
		return err
	}

	_, err = d.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_workers_state ON workers(state)`)
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

// Task operations
type TaskRow struct {
	TaskID           string
	State            int
	Command          string
	Args             string
	Priority         int
	Attempts         int
	MaxRetries       int
	AssignedWorkerID sql.NullString
	LeaseExpiryTime  sql.NullInt64
	CreatedAt        int64
	UpdatedAt        int64
	Output           sql.NullString
	Error            sql.NullString
	CPUNeeded        int
	MemNeeded        int64
	IdempotencyKey   sql.NullString
}

func (d *DB) InsertTask(task *TaskRow) error {
	_, err := d.conn.Exec(`
		INSERT INTO tasks (
			task_id, state, command, args, priority, attempts, max_retries,
			assigned_worker_id, lease_expiry_time, created_at, updated_at,
			output, error, cpu_needed, mem_needed, idempotency_key
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.TaskID, task.State, task.Command, task.Args, task.Priority,
		task.Attempts, task.MaxRetries, task.AssignedWorkerID, task.LeaseExpiryTime,
		task.CreatedAt, task.UpdatedAt, task.Output, task.Error,
		task.CPUNeeded, task.MemNeeded, task.IdempotencyKey)
	return err
}

func (d *DB) GetTask(taskID string) (*TaskRow, error) {
	row := d.conn.QueryRow(`
		SELECT task_id, state, command, args, priority, attempts, max_retries,
			assigned_worker_id, lease_expiry_time, created_at, updated_at,
			output, error, cpu_needed, mem_needed, idempotency_key
		FROM tasks WHERE task_id = ?
	`, taskID)

	task := &TaskRow{}
	err := row.Scan(
		&task.TaskID, &task.State, &task.Command, &task.Args, &task.Priority,
		&task.Attempts, &task.MaxRetries, &task.AssignedWorkerID, &task.LeaseExpiryTime,
		&task.CreatedAt, &task.UpdatedAt, &task.Output, &task.Error,
		&task.CPUNeeded, &task.MemNeeded, &task.IdempotencyKey,
	)
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (d *DB) UpdateTaskState(taskID string, state int, updatedAt int64) error {
	_, err := d.conn.Exec(`
		UPDATE tasks SET state = ?, updated_at = ? WHERE task_id = ?
	`, state, updatedAt, taskID)
	return err
}

func (d *DB) AssignTask(taskID, workerID string, leaseExpiry int64, updatedAt int64) error {
	_, err := d.conn.Exec(`
		UPDATE tasks SET state = ?, assigned_worker_id = ?, lease_expiry_time = ?, updated_at = ?
		WHERE task_id = ?
	`, 1, workerID, leaseExpiry, updatedAt, taskID) // 1 = ASSIGNED
	return err
}

func (d *DB) CompleteTask(taskID string, state int, output, errorMsg string, updatedAt int64) error {
	_, err := d.conn.Exec(`
		UPDATE tasks SET state = ?, output = ?, error = ?, updated_at = ?
		WHERE task_id = ?
	`, state, output, errorMsg, updatedAt, taskID)
	return err
}

func (d *DB) IncrementTaskAttempts(taskID string, updatedAt int64) error {
	_, err := d.conn.Exec(`
		UPDATE tasks SET attempts = attempts + 1, updated_at = ?
		WHERE task_id = ?
	`, updatedAt, taskID)
	return err
}

func (d *DB) RequeueTask(taskID string, updatedAt int64) error {
	_, err := d.conn.Exec(`
		UPDATE tasks SET state = ?, assigned_worker_id = NULL, lease_expiry_time = NULL, updated_at = ?
		WHERE task_id = ?
	`, 0, updatedAt, taskID) // 0 = PENDING
	return err
}

func (d *DB) GetPendingTasks(limit int) ([]*TaskRow, error) {
	rows, err := d.conn.Query(`
		SELECT task_id, state, command, args, priority, attempts, max_retries,
			assigned_worker_id, lease_expiry_time, created_at, updated_at,
			output, error, cpu_needed, mem_needed, idempotency_key
		FROM tasks WHERE state = 0
		ORDER BY priority DESC, created_at ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRow
	for rows.Next() {
		task := &TaskRow{}
		err := rows.Scan(
			&task.TaskID, &task.State, &task.Command, &task.Args, &task.Priority,
			&task.Attempts, &task.MaxRetries, &task.AssignedWorkerID, &task.LeaseExpiryTime,
			&task.CreatedAt, &task.UpdatedAt, &task.Output, &task.Error,
			&task.CPUNeeded, &task.MemNeeded, &task.IdempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (d *DB) GetExpiredLeases(now int64) ([]*TaskRow, error) {
	rows, err := d.conn.Query(`
		SELECT task_id, state, command, args, priority, attempts, max_retries,
			assigned_worker_id, lease_expiry_time, created_at, updated_at,
			output, error, cpu_needed, mem_needed, idempotency_key
		FROM tasks WHERE state = 1 AND lease_expiry_time IS NOT NULL AND lease_expiry_time < ?
	`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRow
	for rows.Next() {
		task := &TaskRow{}
		err := rows.Scan(
			&task.TaskID, &task.State, &task.Command, &task.Args, &task.Priority,
			&task.Attempts, &task.MaxRetries, &task.AssignedWorkerID, &task.LeaseExpiryTime,
			&task.CreatedAt, &task.UpdatedAt, &task.Output, &task.Error,
			&task.CPUNeeded, &task.MemNeeded, &task.IdempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (d *DB) GetAllTasks(limit int) ([]*TaskRow, error) {
	rows, err := d.conn.Query(`
		SELECT task_id, state, command, args, priority, attempts, max_retries,
			assigned_worker_id, lease_expiry_time, created_at, updated_at,
			output, error, cpu_needed, mem_needed, idempotency_key
		FROM tasks
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*TaskRow
	for rows.Next() {
		task := &TaskRow{}
		err := rows.Scan(
			&task.TaskID, &task.State, &task.Command, &task.Args, &task.Priority,
			&task.Attempts, &task.MaxRetries, &task.AssignedWorkerID, &task.LeaseExpiryTime,
			&task.CreatedAt, &task.UpdatedAt, &task.Output, &task.Error,
			&task.CPUNeeded, &task.MemNeeded, &task.IdempotencyKey,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Worker operations
type WorkerRow struct {
	WorkerID      string
	Addr          string
	LastHeartbeat int64
	CPUFree       int
	MemFree       int64
	State         int
}

func (d *DB) InsertWorker(worker *WorkerRow) error {
	_, err := d.conn.Exec(`
		INSERT INTO workers (worker_id, addr, last_heartbeat, cpu_free, mem_free, state)
		VALUES (?, ?, ?, ?, ?, ?)
	`, worker.WorkerID, worker.Addr, worker.LastHeartbeat, worker.CPUFree, worker.MemFree, worker.State)
	return err
}

func (d *DB) UpdateWorkerHeartbeat(workerID string, heartbeat int64, cpuFree int, memFree int64) error {
	_, err := d.conn.Exec(`
		UPDATE workers SET last_heartbeat = ?, cpu_free = ?, mem_free = ?, state = ?
		WHERE worker_id = ?
	`, heartbeat, cpuFree, memFree, 0, workerID) // 0 = ALIVE
	return err
}

func (d *DB) UpdateWorkerState(workerID string, state int) error {
	_, err := d.conn.Exec(`UPDATE workers SET state = ? WHERE worker_id = ?`, state, workerID)
	return err
}

func (d *DB) GetWorker(workerID string) (*WorkerRow, error) {
	row := d.conn.QueryRow(`
		SELECT worker_id, addr, last_heartbeat, cpu_free, mem_free, state
		FROM workers WHERE worker_id = ?
	`, workerID)

	worker := &WorkerRow{}
	err := row.Scan(&worker.WorkerID, &worker.Addr, &worker.LastHeartbeat,
		&worker.CPUFree, &worker.MemFree, &worker.State)
	if err != nil {
		return nil, err
	}
	return worker, nil
}

func (d *DB) GetAllWorkers() ([]*WorkerRow, error) {
	rows, err := d.conn.Query(`
		SELECT worker_id, addr, last_heartbeat, cpu_free, mem_free, state
		FROM workers
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*WorkerRow
	for rows.Next() {
		worker := &WorkerRow{}
		err := rows.Scan(&worker.WorkerID, &worker.Addr, &worker.LastHeartbeat,
			&worker.CPUFree, &worker.MemFree, &worker.State)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, nil
}

func (d *DB) GetStaleWorkers(threshold int64) ([]*WorkerRow, error) {
	rows, err := d.conn.Query(`
		SELECT worker_id, addr, last_heartbeat, cpu_free, mem_free, state
		FROM workers WHERE last_heartbeat < ?
	`, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []*WorkerRow
	for rows.Next() {
		worker := &WorkerRow{}
		err := rows.Scan(&worker.WorkerID, &worker.Addr, &worker.LastHeartbeat,
			&worker.CPUFree, &worker.MemFree, &worker.State)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}
	return workers, nil
}

// Completed tasks (for exactly-once)
func (d *DB) MarkTaskCompleted(idempotencyKey, taskID, result string) error {
	_, err := d.conn.Exec(`
		INSERT OR IGNORE INTO completed_tasks (idempotency_key, task_id, completed_at, result)
		VALUES (?, ?, ?, ?)
	`, idempotencyKey, taskID, time.Now().UnixNano(), result)
	return err
}

func (d *DB) IsTaskCompleted(idempotencyKey string) (bool, string, error) {
	var taskID, result string
	err := d.conn.QueryRow(`
		SELECT task_id, result FROM completed_tasks WHERE idempotency_key = ?
	`, idempotencyKey).Scan(&taskID, &result)
	if err == sql.ErrNoRows {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}
	return true, result, nil
}

