package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	pb "github.com/distributed-scheduler/nimbus/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Worker struct {
	workerID string
	client   pb.MasterServiceClient
	conn     *grpc.ClientConn
	addr     string
}

func NewWorker(masterAddr string, workerAddr string) (*Worker, error) {
	conn, err := grpc.Dial(masterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}

	client := pb.NewMasterServiceClient(conn)

	// Register worker
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get system resources (simplified)
	caps := &pb.WorkerCapabilities{
		CpuTotal: 4,
		MemTotal: 8 * 1024 * 1024 * 1024, // 8GB
		CpuFree:  4,
		MemFree:  8 * 1024 * 1024 * 1024,
	}

	resp, err := client.RegisterWorker(ctx, &pb.RegisterWorkerRequest{
		Addr:         workerAddr,
		Capabilities: caps,
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to register worker: %w", err)
	}

	return &Worker{
		workerID: resp.WorkerId,
		client:   client,
		conn:     conn,
		addr:     workerAddr,
	}, nil
}

func (w *Worker) Start() {
	// Start heartbeat loop
	go w.heartbeatLoop()

	// Start task polling loop
	w.pollLoop()
}

func (w *Worker) heartbeatLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		caps := &pb.WorkerCapabilities{
			CpuTotal: 4,
			MemTotal: 8 * 1024 * 1024 * 1024,
			CpuFree:  4,
			MemFree:  8 * 1024 * 1024 * 1024,
		}

		_, err := w.client.Heartbeat(ctx, &pb.HeartbeatRequest{
			WorkerId:     w.workerID,
			Capabilities: caps,
		})
		if err != nil {
			log.Printf("Heartbeat failed: %v", err)
		}
		cancel()
	}
}

func (w *Worker) pollLoop() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		caps := &pb.WorkerCapabilities{
			CpuTotal: 4,
			MemTotal: 8 * 1024 * 1024 * 1024,
			CpuFree:  4,
			MemFree:  8 * 1024 * 1024 * 1024,
		}

		resp, err := w.client.PollTask(ctx, &pb.PollTaskRequest{
			WorkerId:     w.workerID,
			Capabilities: caps,
		})
		cancel()

		if err != nil {
			log.Printf("PollTask failed: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		if !resp.HasTask {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Execute task
		w.executeTask(resp.Task)
	}
}

func (w *Worker) executeTask(task *pb.Task) {
	log.Printf("Executing task %s: %s %v", task.TaskId, task.Spec.Command, task.Spec.Args)

	var output string
	var errMsg string
	var status pb.TaskState

	// Execute the command
	cmd := exec.Command(task.Spec.Command, task.Spec.Args...)
	cmd.Env = os.Environ()

	result, err := cmd.CombinedOutput()
	output = string(result)

	if err != nil {
		status = pb.TaskState_FAILED
		errMsg = err.Error()
		log.Printf("Task %s failed: %v", task.TaskId, err)
	} else {
		status = pb.TaskState_SUCCEEDED
		log.Printf("Task %s succeeded", task.TaskId)
	}

	// Report result
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = w.client.ReportTaskResult(ctx, &pb.ReportTaskResultRequest{
		TaskId:   task.TaskId,
		WorkerId: w.workerID,
		Status:   status,
		Output:   output,
		Error:    errMsg,
	})
	if err != nil {
		log.Printf("Failed to report task result: %v", err)
	}
}

func (w *Worker) Close() {
	w.conn.Close()
}

func main() {
	masterAddr := flag.String("master", "localhost:50051", "Master server address")
	workerAddr := flag.String("addr", "localhost:50052", "Worker address")
	flag.Parse()

	worker, err := NewWorker(*masterAddr, *workerAddr)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}
	defer worker.Close()

	log.Printf("Worker %s started, connected to master at %s", worker.workerID, *masterAddr)
	worker.Start()
}

