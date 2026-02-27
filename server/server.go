package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
				if sch == -1 {
					log.Printf("[*] Scheduling one-off Job %s", jobId)
					select {
					case jobQueue <- jobContext.Job:
						log.Println("[+] Job pushed to queue")
						// Use -2 as sentinel: "already dispatched, do not re-schedule"
						jobContext.Job.Schedule = "-2"
						store.jobs[jobId] = jobContext
					default:
						log.Println("[-] Job queue full! Skipping.")
					}
				} else if sch > 0 && now % int64(sch) == 0 {
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
	return &pb.JobResponse{Success: true, Message: "[+] Job Accepted by the server", Id: req.Id}, nil // server response
}

// Worker calls this function to connect to the server 
func (s *server) ConnectWorker(req *pb.WorkerHello, stream grpc.ServerStreamingServer[pb.Job]) (error){
	log.Printf("[+] Worker %s connected", req.WorkerId)
	for {
		select {
		case job := <-jobQueue: // If a new job enters the channel, it is sent to the worker
			log.Printf("[*] Dispatching Job %s to Worker %s", job.Id, req.WorkerId)
			err := stream.Send(job)
			if err != nil {
				log.Printf("[-] Error sending job to worker %s, re-queuing: %v", req.WorkerId, err)
				// Put the job back so another worker can pick it up.
				select {
				case jobQueue <- job:
				default:
					log.Printf("[-] Re-queue failed: channel full, job %s lost", job.Id)
				}
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
		jobContext.Output += req.Output + "\n"
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
	
	// Load the server certifcates and keys
	serverCert, err := os.ReadFile("certs/server-cert.pem")
	if err != nil{
		log.Fatalf("[-] Error loading server certificate %v", err)
	}
	serverKey, err := os.ReadFile("certs/server-key.pem")
	if err != nil{
		log.Fatalf("[-] Error loading server key %v", err)
	}

	// Root cert
	caCert, err := os.ReadFile("certs/ca-cert.pem")
	if err != nil{
		log.Printf("[-] Error loading server certificate %v", err)
	}

	// Generate a certificate using key and cert block
	cert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		log.Fatal(err)
	}

	// Create a cert pool and add the root ca to it
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
        log.Fatalln("[-] Could not append cert to pool")
    }

	// Create a custom TLS configuration
	tlsConfig := &tls.Config{
		ClientCAs: caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}

	// Wrap the tls.Config
	creds := credentials.NewTLS(tlsConfig)

	go runScheduler()

	if store.db, err = CreateDB(); err != nil{
		log.Printf("[-] Error creating database: %v", err)
	}
	if err = LoadJobs(store.db); err!=nil{
		log.Printf("[-] Error loading jobs into hashmap: %v", err)
	}
	defer store.db.Close()
	
	grpcServer := grpc.NewServer(grpc.Creds(creds)) // Create a new grpc server using the credentials
	pb.RegisterSchedulerServer(grpcServer, &server{})
	if err:= grpcServer.Serve(lis); err != nil{
		log.Fatal(err)
	}
}

