package scheduler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/distributed-scheduler/nimbus/internal/db"
	"github.com/distributed-scheduler/nimbus/internal/leases"
	"github.com/distributed-scheduler/nimbus/internal/queue"
	pb "github.com/distributed-scheduler/nimbus/proto"
)

type Scheduler struct {
	db           *db.DB
	taskQueue    *queue.PriorityQueue
	leaseManager *leases.LeaseManager
	workers      map[string]*WorkerInfo
	mu           sync.RWMutex
}

type WorkerInfo struct {
	WorkerID      string
	Addr          string
	LastHeartbeat time.Time
	CPUFree       int
	MemFree       int64
	State         pb.WorkerState
}

func NewScheduler(database *db.DB) *Scheduler {
	s := &Scheduler{
		db:           database,
		taskQueue:    queue.NewPriorityQueue(),
		leaseManager: leases.NewLeaseManager(),
		workers:      make(map[string]*WorkerInfo),
	}
	
	// Load pending tasks from database into queue
	// Do it synchronously but after initialization to avoid deadlock
	s.loadPendingTasks()
	
	return s
}

func (s *Scheduler) loadPendingTasks() {
	pendingTasks, err := s.db.GetPendingTasks(1000) // Load up to 1000 pending tasks
	if err != nil {
		return
	}
	
	for _, task := range pendingTasks {
		s.taskQueue.PushTask(task.TaskID, task.Priority)
	}
}

func (s *Scheduler) SubmitTask(ctx context.Context, spec *pb.TaskSpec) (string, error) {
	taskID := generateTaskID()
	now := time.Now().UnixNano()

	// Check idempotency if key is provided
	if spec.IdempotencyKey != "" {
		completed, result, err := s.db.IsTaskCompleted(spec.IdempotencyKey)
		if err != nil {
			return "", fmt.Errorf("failed to check idempotency: %w", err)
		}
		if completed {
			// Return existing task ID
			return result, nil
		}
	}

	// Serialize args
	argsJSON, _ := json.Marshal(spec.Args)

	taskRow := &db.TaskRow{
		TaskID:         taskID,
		State:          int(pb.TaskState_PENDING),
		Command:        spec.Command,
		Args:           string(argsJSON),
		Priority:       int(spec.Priority),
		Attempts:       0,
		MaxRetries:     int(spec.MaxRetries),
		CreatedAt:      now,
		UpdatedAt:      now,
		CPUNeeded:      int(spec.CpuNeeded),
		MemNeeded:      spec.MemNeeded,
		IdempotencyKey: sql.NullString{String: spec.IdempotencyKey, Valid: spec.IdempotencyKey != ""},
	}

	if err := s.db.InsertTask(taskRow); err != nil {
		return "", fmt.Errorf("failed to insert task: %w", err)
	}

	// Add to priority queue
	s.taskQueue.PushTask(taskID, int(spec.Priority))

	return taskID, nil
}

func (s *Scheduler) RegisterWorker(ctx context.Context, addr string, caps *pb.WorkerCapabilities) (string, error) {
	workerID := generateWorkerID()
	now := time.Now().UnixNano()

	workerRow := &db.WorkerRow{
		WorkerID:      workerID,
		Addr:          addr,
		LastHeartbeat: now,
		CPUFree:       int(caps.CpuFree),
		MemFree:       caps.MemFree,
		State:         int(pb.WorkerState_ALIVE),
	}

	if err := s.db.InsertWorker(workerRow); err != nil {
		return "", fmt.Errorf("failed to insert worker: %w", err)
	}

	s.mu.Lock()
	s.workers[workerID] = &WorkerInfo{
		WorkerID:      workerID,
		Addr:          addr,
		LastHeartbeat: time.Now(),
		CPUFree:       int(caps.CpuFree),
		MemFree:       caps.MemFree,
		State:         pb.WorkerState_ALIVE,
	}
	s.mu.Unlock()

	return workerID, nil
}

func (s *Scheduler) Heartbeat(ctx context.Context, workerID string, caps *pb.WorkerCapabilities) error {
	now := time.Now().UnixNano()

	if err := s.db.UpdateWorkerHeartbeat(workerID, now, int(caps.CpuFree), caps.MemFree); err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	s.mu.Lock()
	if worker, exists := s.workers[workerID]; exists {
		worker.LastHeartbeat = time.Now()
		worker.CPUFree = int(caps.CpuFree)
		worker.MemFree = caps.MemFree
		worker.State = pb.WorkerState_ALIVE
	}
	s.mu.Unlock()

	return nil
}

func (s *Scheduler) PollTask(ctx context.Context, workerID string, caps *pb.WorkerCapabilities) (*pb.Task, error) {
	// Check context first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Update worker capabilities
	s.mu.Lock()
	if worker, exists := s.workers[workerID]; exists {
		worker.CPUFree = int(caps.CpuFree)
		worker.MemFree = caps.MemFree
	}
	s.mu.Unlock()

	// If queue is empty, try loading pending tasks from DB
	// Check length by trying to pop (non-destructive check would require exposing internal method)
	// For now, just try loading - it's safe to call multiple times
	s.loadPendingTasks()

	// Limit iterations to prevent infinite loop
	maxIterations := 100
	iterations := 0

	// Try to get a task from queue
	for iterations < maxIterations {
		// Check context periodically
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		taskID, ok := s.taskQueue.PopTask()
		if !ok {
			return nil, nil // No tasks available
		}

		iterations++

		// Get task from DB
		taskRow, err := s.db.GetTask(taskID)
		if err != nil {
			continue // Task might have been deleted, try next
		}

		// Check if task is still PENDING
		if taskRow.State != int(pb.TaskState_PENDING) {
			continue // Task already assigned, try next
		}

		// Check resource requirements
		if !s.canAssignTask(taskRow, caps) {
			// Put back in queue
			s.taskQueue.PushTask(taskID, taskRow.Priority)
			continue
		}

		// Assign task with lease
		leaseExpiry := time.Now().Add(leases.DefaultLeaseDuration).UnixNano()
		now := time.Now().UnixNano()

		if err := s.db.AssignTask(taskID, workerID, leaseExpiry, now); err != nil {
			s.taskQueue.PushTask(taskID, taskRow.Priority) // Put back
			continue
		}

		s.leaseManager.Acquire(taskID, leases.DefaultLeaseDuration)

		// Convert to protobuf
		return s.taskRowToProto(taskRow, workerID, leaseExpiry), nil
	}

	// If we've tried many times and failed, return nil (no task available)
	return nil, nil
}

func (s *Scheduler) ReportTaskResult(ctx context.Context, taskID, workerID string, status pb.TaskState, output, errorMsg string) error {
	taskRow, err := s.db.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Verify worker owns this task
	if !taskRow.AssignedWorkerID.Valid || taskRow.AssignedWorkerID.String != workerID {
		return fmt.Errorf("worker %s does not own task %s", workerID, taskID)
	}

	now := time.Now().UnixNano()
	s.leaseManager.Release(taskID)

	switch status {
	case pb.TaskState_SUCCEEDED:
		if err := s.db.CompleteTask(taskID, int(status), output, "", now); err != nil {
			return err
		}

		// Mark as completed for idempotency
		if taskRow.IdempotencyKey.Valid {
			s.db.MarkTaskCompleted(taskRow.IdempotencyKey.String, taskID, output)
		}

	case pb.TaskState_FAILED:
		// Check retries
		if taskRow.Attempts < taskRow.MaxRetries {
			// Retry with exponential backoff
			backoff := time.Duration(math.Pow(2, float64(taskRow.Attempts))) * time.Second
			time.Sleep(backoff)

			if err := s.db.IncrementTaskAttempts(taskID, now); err != nil {
				return err
			}
			if err := s.db.RequeueTask(taskID, now); err != nil {
				return err
			}

			// Put back in queue
			s.taskQueue.PushTask(taskID, taskRow.Priority)
		} else {
			// Max retries exceeded
			if err := s.db.CompleteTask(taskID, int(status), "", errorMsg, now); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Scheduler) GetTaskStatus(ctx context.Context, taskID string) (*pb.Task, error) {
	taskRow, err := s.db.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	return s.taskRowToProto(taskRow, "", 0), nil
}

func (s *Scheduler) ListTasks(ctx context.Context, filterState pb.TaskState, limit int32) ([]*pb.Task, error) {
	var taskRows []*db.TaskRow
	var err error

	if filterState == pb.TaskState_PENDING {
		taskRows, err = s.db.GetPendingTasks(int(limit))
	} else {
		taskRows, err = s.db.GetAllTasks(int(limit))
	}

	if err != nil {
		return nil, err
	}

	tasks := make([]*pb.Task, 0, len(taskRows))
	for _, row := range taskRows {
		tasks = append(tasks, s.taskRowToProto(row, "", 0))
	}

	return tasks, nil
}

// Background processes
func (s *Scheduler) StartFailureDetector(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkWorkerHealth()
			s.recoverExpiredLeases()
		}
	}
}

func (s *Scheduler) checkWorkerHealth() {
	now := time.Now()
	suspectedThreshold := now.Add(-6 * time.Second)
	deadThreshold := now.Add(-12 * time.Second)

	s.mu.RLock()
	workers := make([]*WorkerInfo, 0, len(s.workers))
	for _, w := range s.workers {
		workers = append(workers, w)
	}
	s.mu.RUnlock()

	for _, worker := range workers {
		lastHeartbeat := worker.LastHeartbeat
		if lastHeartbeat.Before(deadThreshold) {
			s.db.UpdateWorkerState(worker.WorkerID, int(pb.WorkerState_DEAD))
			s.mu.Lock()
			worker.State = pb.WorkerState_DEAD
			s.mu.Unlock()
		} else if lastHeartbeat.Before(suspectedThreshold) {
			s.db.UpdateWorkerState(worker.WorkerID, int(pb.WorkerState_SUSPECTED))
			s.mu.Lock()
			worker.State = pb.WorkerState_SUSPECTED
			s.mu.Unlock()
		}
	}
}

func (s *Scheduler) recoverExpiredLeases() {
	now := time.Now().UnixNano()
	expiredTasks, err := s.db.GetExpiredLeases(now)
	if err != nil {
		return
	}

	for _, taskRow := range expiredTasks {
		// Requeue the task
		if err := s.db.RequeueTask(taskRow.TaskID, now); err != nil {
			continue
		}

		s.leaseManager.Release(taskRow.TaskID)
		s.taskQueue.PushTask(taskRow.TaskID, taskRow.Priority)
	}
}

// Helper methods
func (s *Scheduler) canAssignTask(taskRow *db.TaskRow, caps *pb.WorkerCapabilities) bool {
	if taskRow.CPUNeeded > 0 && int(caps.CpuFree) < taskRow.CPUNeeded {
		return false
	}
	if taskRow.MemNeeded > 0 && caps.MemFree < taskRow.MemNeeded {
		return false
	}
	return true
}

func (s *Scheduler) taskRowToProto(row *db.TaskRow, assignedWorkerID string, leaseExpiry int64) *pb.Task {
	var args []string
	json.Unmarshal([]byte(row.Args), &args)

	task := &pb.Task{
		TaskId:   row.TaskID,
		State:    pb.TaskState(row.State),
		Attempts: int32(row.Attempts),
		Spec: &pb.TaskSpec{
			Command:        row.Command,
			Args:           args,
			Priority:       int32(row.Priority),
			MaxRetries:     int32(row.MaxRetries),
			CpuNeeded:      int32(row.CPUNeeded),
			MemNeeded:      row.MemNeeded,
			IdempotencyKey: getStringValue(row.IdempotencyKey),
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if assignedWorkerID != "" {
		task.AssignedWorkerId = assignedWorkerID
	} else if row.AssignedWorkerID.Valid {
		task.AssignedWorkerId = row.AssignedWorkerID.String
	}

	if leaseExpiry > 0 {
		task.LeaseExpiryTime = leaseExpiry
	} else if row.LeaseExpiryTime.Valid {
		task.LeaseExpiryTime = row.LeaseExpiryTime.Int64
	}

	if row.Output.Valid {
		task.Output = row.Output.String
	}
	if row.Error.Valid {
		task.Error = row.Error.String
	}

	return task
}

func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func generateWorkerID() string {
	return fmt.Sprintf("worker-%d", time.Now().UnixNano())
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

