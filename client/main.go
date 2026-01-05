package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/distributed-scheduler/nimbus/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client pb.MasterServiceClient
	conn   *grpc.ClientConn
}

func NewClient(masterAddr string) (*Client, error) {
	conn, err := grpc.Dial(masterAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master: %w", err)
	}

	return &Client{
		client: pb.NewMasterServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}

func (c *Client) SubmitTask(command string, args []string, priority int32, maxRetries int32) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	spec := &pb.TaskSpec{
		Command:    command,
		Args:       args,
		Priority:   priority,
		MaxRetries: maxRetries,
	}

	resp, err := c.client.SubmitTask(ctx, &pb.SubmitTaskRequest{Spec: spec})
	if err != nil {
		return "", err
	}

	return resp.TaskId, nil
}

func (c *Client) GetTaskStatus(taskID string) (*pb.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.GetTaskStatus(ctx, &pb.GetTaskStatusRequest{TaskId: taskID})
	if err != nil {
		return nil, err
	}

	return resp.Task, nil
}

func (c *Client) ListTasks(filterState pb.TaskState, limit int32) ([]*pb.Task, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := c.client.ListTasks(ctx, &pb.ListTasksRequest{
		FilterState: filterState,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	return resp.Tasks, nil
}

func main() {
	masterAddr := flag.String("master", "localhost:50051", "Master server address")
	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  submit <command> [args...] [--priority=N] [--retries=N]")
		fmt.Println("  status <task_id>")
		fmt.Println("  list [--state=STATE] [--limit=N]")
		os.Exit(1)
	}

	client, err := NewClient(*masterAddr)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	command := os.Args[1]

	switch command {
	case "submit":
		if len(os.Args) < 3 {
			log.Fatal("Usage: submit <command> [args...]")
		}

		cmd := os.Args[2]
		var args []string
		priority := int32(0)
		maxRetries := int32(3)

		for i := 3; i < len(os.Args); i++ {
			arg := os.Args[i]
			if strings.HasPrefix(arg, "--priority=") {
				p, _ := strconv.Atoi(strings.TrimPrefix(arg, "--priority="))
				priority = int32(p)
			} else if strings.HasPrefix(arg, "--retries=") {
				r, _ := strconv.Atoi(strings.TrimPrefix(arg, "--retries="))
				maxRetries = int32(r)
			} else {
				args = append(args, arg)
			}
		}

		taskID, err := client.SubmitTask(cmd, args, priority, maxRetries)
		if err != nil {
			log.Fatalf("Failed to submit task: %v", err)
		}
		fmt.Printf("Task submitted: %s\n", taskID)

	case "status":
		if len(os.Args) < 3 {
			log.Fatal("Usage: status <task_id>")
		}
		taskID := os.Args[2]

		task, err := client.GetTaskStatus(taskID)
		if err != nil {
			log.Fatalf("Failed to get task status: %v", err)
		}

		fmt.Printf("Task ID: %s\n", task.TaskId)
		fmt.Printf("State: %s\n", task.State.String())
		fmt.Printf("Command: %s %v\n", task.Spec.Command, task.Spec.Args)
		fmt.Printf("Priority: %d\n", task.Spec.Priority)
		fmt.Printf("Attempts: %d/%d\n", task.Attempts, task.Spec.MaxRetries)
		if task.AssignedWorkerId != "" {
			fmt.Printf("Assigned to: %s\n", task.AssignedWorkerId)
		}
		if task.Output != "" {
			fmt.Printf("Output:\n%s\n", task.Output)
		}
		if task.Error != "" {
			fmt.Printf("Error: %s\n", task.Error)
		}

	case "list":
		filterState := pb.TaskState_PENDING
		limit := int32(10)

		for i := 2; i < len(os.Args); i++ {
			arg := os.Args[i]
			if strings.HasPrefix(arg, "--state=") {
				stateStr := strings.TrimPrefix(arg, "--state=")
				switch strings.ToUpper(stateStr) {
				case "PENDING":
					filterState = pb.TaskState_PENDING
				case "ASSIGNED":
					filterState = pb.TaskState_ASSIGNED
				case "RUNNING":
					filterState = pb.TaskState_RUNNING
				case "SUCCEEDED":
					filterState = pb.TaskState_SUCCEEDED
				case "FAILED":
					filterState = pb.TaskState_FAILED
				}
			} else if strings.HasPrefix(arg, "--limit=") {
				l, _ := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
				limit = int32(l)
			}
		}

		tasks, err := client.ListTasks(filterState, limit)
		if err != nil {
			log.Fatalf("Failed to list tasks: %v", err)
		}

		fmt.Printf("Found %d tasks:\n", len(tasks))
		for _, task := range tasks {
			fmt.Printf("  %s [%s] %s %v (attempts: %d/%d)\n",
				task.TaskId, task.State.String(), task.Spec.Command, task.Spec.Args,
				task.Attempts, task.Spec.MaxRetries)
		}

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

