package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/distributed-scheduler/nimbus/internal/db"
	"github.com/distributed-scheduler/nimbus/internal/scheduler"
	pb "github.com/distributed-scheduler/nimbus/proto"
	"google.golang.org/grpc"
)

type masterServer struct {
	pb.UnimplementedMasterServiceServer
	scheduler *scheduler.Scheduler
}

func (s *masterServer) RegisterWorker(ctx context.Context, req *pb.RegisterWorkerRequest) (*pb.RegisterWorkerResponse, error) {
	workerID, err := s.scheduler.RegisterWorker(ctx, req.Addr, req.Capabilities)
	if err != nil {
		return nil, err
	}
	return &pb.RegisterWorkerResponse{WorkerId: workerID}, nil
}

func (s *masterServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	err := s.scheduler.Heartbeat(ctx, req.WorkerId, req.Capabilities)
	if err != nil {
		return nil, err
	}
	return &pb.HeartbeatResponse{Ack: true}, nil
}

func (s *masterServer) PollTask(ctx context.Context, req *pb.PollTaskRequest) (*pb.PollTaskResponse, error) {
	task, err := s.scheduler.PollTask(ctx, req.WorkerId, req.Capabilities)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return &pb.PollTaskResponse{HasTask: false}, nil
	}
	return &pb.PollTaskResponse{Task: task, HasTask: true}, nil
}

func (s *masterServer) ReportTaskResult(ctx context.Context, req *pb.ReportTaskResultRequest) (*pb.ReportTaskResultResponse, error) {
	err := s.scheduler.ReportTaskResult(ctx, req.TaskId, req.WorkerId, req.Status, req.Output, req.Error)
	if err != nil {
		return nil, err
	}
	return &pb.ReportTaskResultResponse{Ack: true}, nil
}

func (s *masterServer) SubmitTask(ctx context.Context, req *pb.SubmitTaskRequest) (*pb.SubmitTaskResponse, error) {
	taskID, err := s.scheduler.SubmitTask(ctx, req.Spec)
	if err != nil {
		return nil, err
	}
	return &pb.SubmitTaskResponse{TaskId: taskID}, nil
}

func (s *masterServer) GetTaskStatus(ctx context.Context, req *pb.GetTaskStatusRequest) (*pb.GetTaskStatusResponse, error) {
	task, err := s.scheduler.GetTaskStatus(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}
	return &pb.GetTaskStatusResponse{Task: task}, nil
}

func (s *masterServer) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	tasks, err := s.scheduler.ListTasks(ctx, req.FilterState, req.Limit)
	if err != nil {
		return nil, err
	}
	return &pb.ListTasksResponse{Tasks: tasks}, nil
}

func main() {
	port := flag.Int("port", 50051, "gRPC server port")
	dbPath := flag.String("db", "./master.db", "Database path")
	flag.Parse()

	// Initialize database
	database, err := db.NewDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize scheduler
	sched := scheduler.NewScheduler(database)

	// Start failure detector
	ctx := context.Background()
	go sched.StartFailureDetector(ctx)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMasterServiceServer(s, &masterServer{scheduler: sched})

	log.Printf("Master server listening on :%d", *port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

