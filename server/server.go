package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
)


var jobQueue = make(chan *pb.Job, 100)

// type JobStore struct{
// 	mu sync.Mutex;
// 	jobs map[string]JobContext // HashMap to store all the jobs
// }

// type JobContext struct{
// 	status string;
// 	output string;
// 	job *pb.Job
// }

// // Initialize the JobStore struct
// var store = JobStore{
// 	jobs : make(map[string]JobContext),
// }

func runScheduler() {
    log.Println("[+] Scheduler started...")
    for {
        time.Sleep(1 * time.Second)
		
        func() {
            store.mu.Lock()
            defer store.mu.Unlock() 

            now := time.Now().Unix()

            for jobId, jobContext := range store.jobs {
                sch, err := strconv.Atoi(jobContext.Job.Schedule)
                if err != nil {
                    continue
                }

                if now % int64(sch) == 0 {
                    log.Printf("[*] Scheduling Job %s", jobId)
                    select {
                    case jobQueue <- jobContext.Job:
                        log.Println("[+] Job pushed to queue")
                    default:
                        log.Println("[-] Job queue full! Skipping.")
                    }
                }
            }
        }() 
    }
}


// Embed the struct from protobuf
type server struct{
	pb.UnimplementedSchedulerServer
}

// Client calls this function to submit a job to the server
func (s *server) SubmitJob(ctx context.Context, req *pb.Job) (*pb.JobResponse, error){
	store.mu.Lock() // No two goroutines can access the hashmap at the same time
	defer store.mu.Unlock()
	jobQueue <- req
	new_context := JobContext{
		Status: "QUEUED",
		Output: "",
		Job: req,
	}
	store.jobs[req.Id] = new_context

	// Save the job in the DB
	if err := SaveJob(req.Id, new_context, store.db); err != nil {
    	log.Printf("[-] Failed to save job to DB: %v", err)
	}
	log.Printf("[+] Saved Job %v : %v", req.Id, req.Command)
	return &pb.JobResponse{Success: true, Message: "[+] Job Accepted by the server"}, nil // server response
}

// Worker calls this function to connect to the server 
func (s *server) ConnectWorker(req *pb.WorkerHello, stream grpc.ServerStreamingServer[pb.Job]) (error){
	log.Printf("[+] Worker %s connected", req.WorkerId)
	for {
		select{
		case job := <- jobQueue: // If a new job enters the channel, it is sent to the worker
			log.Printf("[*] Dispatching Job %s to Worker %s", job.Id, req.WorkerId)
			err := stream.Send(job)
			if err != nil{
				log.Printf("[-] Error sending job to worker: %v", err)
				return err
			}
		case <-stream.Context().Done():
			log.Printf("[-] Worker %s disconnected.", req.WorkerId)
			return nil
		}
	}
}

// Worker calls this function to let the server know that the job has been completed
func (s* server) CompleteJob(ctx context.Context, req *pb.JobResult)(*pb.Empty, error){
	store.mu.Lock()
	defer store.mu.Unlock()
	
	// Retrieve job Id, status, and job context
	status := req.Success
	jobId := req.JobId
	jobContext := store.jobs[jobId]

	// Update the job status accordingly
	if status == true{
		jobContext.Status = "COMPLETED"
		jobContext.Output = req.Output
		store.jobs[jobId] = jobContext
		if err := SaveJob(req.JobId, jobContext, store.db); err != nil { 
    		log.Printf("[-] Failed to save job completion: %v", err)
		}
		log.Printf("[+] Job %v : %v", jobContext.Job.Id, jobContext.Status)
	} else{
		jobContext.Status = "FAILED"
		jobContext.Output = req.Output
		store.jobs[jobId] = jobContext
		log.Printf("[+] Job %v : %v", jobContext.Job.Id, jobContext.Status)
	}

	return &pb.Empty{}, nil
}

func (s* server) GetJobStatus(ctx context.Context, req *pb.JobStatusRequest)(*pb.JobStatusResponse, error){
	store.mu.Lock()
	defer store.mu.Unlock()
	jobContext, ok := store.jobs[req.JobId]
	if !ok {
    	return nil, fmt.Errorf("[-] Job not found")
	}

	return &pb.JobStatusResponse{JobId: req.JobId,
								 Status: jobContext.Status,
								 Output: string(jobContext.Output),}, nil				

}

func main(){
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil{
		log.Fatal(err)
	}
	log.Printf("[+] Server listening on port %v", port)
	go runScheduler()

	if store.db, err = CreateDB(); err != nil{
		log.Printf("[-] Error creating database: %v", err)
	}
	if err = LoadJobs(store.db); err!=nil{
		log.Printf("[-] Error loading jobs into hashmap: %v", err)
	}
	defer store.db.Close()
	
	grpcServer := grpc.NewServer()
	pb.RegisterSchedulerServer(grpcServer, &server{})
	if err:= grpcServer.Serve(lis); err != nil{
		log.Fatal(err)
	}
}

