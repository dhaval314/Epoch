package main

import (
	"context"
	"io"
	"os"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	//"os/exec"
	pb "github.com/dhaval314/epoch/proto"
	//"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

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

func executeCommand(ctx context.Context, req *pb.Job){

	// NOTE: client.NewClientWithOpts is Deprecated, but the new version (client.New()) doesnt work because of dependency issues
	// Create client 
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err!= nil{
		log.Printf("[-] Error creating client: %v\n", err)
		return
	}
	defer apiClient.Close()

	// Pull the image from dockerhub
	reader, err := apiClient.ImagePull(ctx, req.Image, image.PullOptions{})
	if err!= nil{
		log.Printf("[-] Error pulling container: %v\n", err)
		return
	}
	defer reader.Close()
	
	io.Copy(os.Stdout, reader)

	// Create a container
	resp, err := apiClient.ContainerCreate(ctx, &container.Config{
		Cmd:   []string{"sh","-c", req.Command},
		Image: req.Image,
	}, nil, nil, nil, "")
	if err != nil{
		log.Printf("[-] Error creating container: %v\n", err)
		return
	}
	log.Printf("[+] Created container with Id: %v\n", resp.ID)

	// Start the container
	err = apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil{
		log.Printf("[-] Error starting container: %v\n", err)
		return
	}
	log.Printf("[+] Started container with Id: %v\n", resp.ID)

	statusCh, errCh := apiClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	// Wait for the container to finish
	select{
	case err := <-errCh:
		if err !=nil{
			log.Printf("[-] Error waiting: %v", err)
            return
		}
	case <-statusCh:
		// Job is done
	}

	// Get the output from the container
	out, err := apiClient.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
    if err != nil {
        log.Printf("[-] Error getting logs: %v", err)
        return
    }
    defer out.Close()

	
	fmt.Println("--- CONTAINER OUTPUT ---")
    io.Copy(os.Stdout, out)
    fmt.Println("------------------------")
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
					log.Printf("[+] Executing %v : %v", jobId, job.Command)
					ctx := context.Background()
					go executeCommand(ctx, job)
				}
			} else{
				log.Printf("[-] Invalid Schedule %v for Job %v. Error: %v", job.Schedule, jobId, err)
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
	log.Printf("[+] Saved Job %v : %v", req.Id, req.Command)
	return &pb.JobResponse{Success: true, Message: "[+] Job Accepted by the server"}, nil // server response
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

