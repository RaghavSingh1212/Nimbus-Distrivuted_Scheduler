.PHONY: proto build master worker client clean run-master run-worker test

proto:
	@echo "Generating protobuf code..."
	@mkdir -p proto
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/scheduler.proto

build: proto
	@echo "Building master..."
	@go build -o bin/master ./master
	@echo "Building worker..."
	@go build -o bin/worker ./worker
	@echo "Building client..."
	@go build -o bin/client ./client

master: proto
	@go build -o bin/master ./master

worker: proto
	@go build -o bin/worker ./worker

client: proto
	@go build -o bin/client ./client

clean:
	@rm -rf bin/ proto/*.pb.go proto/*_grpc.pb.go *.db

run-master: master
	@./bin/master -port 50051 -db ./master.db

run-worker: worker
	@./bin/worker -master localhost:50051

test:
	@go test ./...

deps:
	@go mod download
	@go mod tidy

