package main

import (
	"context"
	//"fmt"
	"log"
	"net"
	"sync"
	"time"
	"strconv"
	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
)

type JobStore struct{
	mu sync.Mutex;
	jobs map[string]*pb.Job // HashMap to store all the jobs
}

// Initialize the JobStore struct
var store = JobStore{
	jobs : make(map[string]*pb.Job),
}

func runScheduler() {
	curr_time := time.Now().Unix()
	for {
		time.Sleep(1* time.Second)
		store.mu.Lock()

		// Iterate over jobs
		for jobId, job := range store.jobs{
			if sch, err := strconv.Atoi(job.Schedule); err == nil {
				
				// Check if its time to execute the command
				if (time.Now().Unix() - curr_time) % int64(sch) == 0 {
					log.Printf("Executing %v : %v", jobId, job.Command)
				}
			} else{
				log.Printf("Invalid Schedule %v for Job %v. Error: %v", job.Schedule, jobId, err)
			}
		}
		store.mu.Unlock()
	}
}


// Embed the struct from protobuf
type server struct{
	pb.UnimplementedSchedulerServer
}

func (s *server) SubmitJob(ctx context.Context, req *pb.Job) (*pb.JobResponse, error){
	store.mu.Lock() // No two goroutines can access the hashmap at the same time
	defer store.mu.Unlock()

	store.jobs[req.Id] = req
	log.Printf("Saved Job %v : %v", req.Id, req.Command)
	return &pb.JobResponse{Success: true, Message: "Job Accepted by the server"}, nil // server response
}

func main(){
	lis, err := net.Listen("tcp", ":50051")
	if err != nil{
		log.Fatal(err)
	}
	
	go runScheduler()

	grpcServer := grpc.NewServer()
	pb.RegisterSchedulerServer(grpcServer, &server{})
	if err:= grpcServer.Serve(lis); err != nil{
		log.Fatal(err)
	}
}

