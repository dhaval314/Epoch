package main

import (
	"context"

	"log"
	"net"
	"strconv"
	"sync"
	"time"

	//"os/exec"
	pb "github.com/dhaval314/epoch/proto"
	//"github.com/docker/docker/api/types"

	"google.golang.org/grpc"
)


var jobQueue = make(chan *pb.Job, 100)

type JobStore struct{
	mu sync.Mutex;
	jobs map[string]*pb.Job // HashMap to store all the jobs
}

// Initialize the JobStore struct
var store = JobStore{
	jobs : make(map[string]*pb.Job),
}

func runScheduler() {
    log.Println("[+] Scheduler started...")
    for {
        time.Sleep(1 * time.Second)
		
        func() {
            store.mu.Lock()
            defer store.mu.Unlock() 

            now := time.Now().Unix()

            for jobId, job := range store.jobs {
                sch, err := strconv.Atoi(job.Schedule)
                if err != nil {
                    continue
                }

                if now % int64(sch) == 0 {
                    log.Printf("[*] Scheduling Job %s", jobId)
                    select {
                    case jobQueue <- job:
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

func (s *server) SubmitJob(ctx context.Context, req *pb.Job) (*pb.JobResponse, error){
	store.mu.Lock() // No two goroutines can access the hashmap at the same time
	defer store.mu.Unlock()
	jobQueue <- req
	store.jobs[req.Id] = req
	log.Printf("[+] Saved Job %v : %v", req.Id, req.Command)
	return &pb.JobResponse{Success: true, Message: "[+] Job Accepted by the server"}, nil // server response
}

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

func main(){
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil{
		log.Fatal(err)
	}
	log.Printf("[+] Server listening on port %v", port)
	go runScheduler()

	grpcServer := grpc.NewServer()
	pb.RegisterSchedulerServer(grpcServer, &server{})
	if err:= grpcServer.Serve(lis); err != nil{
		log.Fatal(err)
	}
}

