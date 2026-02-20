package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"log"
	"os"

	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
)

func executeCommand(ctx context.Context, req *pb.Job)(string, error){

	// NOTE: client.NewClientWithOpts is Deprecated, but the new version (client.New()) doesnt work because of dependency issues
	// Create client 
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err!= nil{
		log.Printf("[-] Error creating client: %v\n", err)
		return "", err
	}
	defer apiClient.Close()

	// Pull the image from dockerhub
	reader, err := apiClient.ImagePull(ctx, req.Image, image.PullOptions{})
	if err!= nil{
		log.Printf("[-] Error pulling container: %v\n", err)
		return "", err
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
		return "", err
	}
	log.Printf("[+] Created container with Id: %v\n", resp.ID)

	// Start the container
	err = apiClient.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil{
		log.Printf("[-] Error starting container: %v\n", err)
		return "", err
	}
	log.Printf("[+] Started container with Id: %v\n", resp.ID)

	statusCh, errCh := apiClient.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	// Wait for the container to finish
	select{
	case err := <-errCh:
		if err !=nil{
			log.Printf("[-] Error waiting: %v", err)
            return "", err
		}
	case <-statusCh:
		// Job is done
	}
	log.Printf("[+] Executed container with Id: %v\n", resp.ID)

	// Get the output from the container
	out, err := apiClient.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
    if err != nil {
        log.Printf("[-] Error getting logs: %v", err)
        return "", err
    }
    defer out.Close()

	// Return the container output
	bodyBytes, err := io.ReadAll(out)
	if err != nil{
		log.Println("[-] Error reading container output")
		return "", nil
	}
	bodyString := string(bodyBytes)
	return bodyString, nil
}

func main(){

	// Generate the certificate from the pem blocks
	cert, err := tls.LoadX509KeyPair("certs/worker-cert.pem", "certs/worker-key.pem")
	if err != nil{
		log.Fatalf("[-] Error reading certificates %v", err)
	}

	// Root cert
	caCert, err := os.ReadFile("certs/ca-cert.pem")
	if err != nil{
		log.Printf("[-] Error loading server certificate %v", err)
	}

	// Create a cert pool and add the root ca to it
	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
        log.Fatalln("[-] Could not append cert to pool")
    }
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs: caCertPool, // The Server used ClientCAs to verify incoming clients. The Client/Worker uses RootCAs to verify the destination server.
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil{
		log.Printf("[-] Error connecting to server: %v\n", err)
	}
	defer conn.Close()
	log.Println("[+] Successfully Connected to the server")

	client := pb.NewSchedulerClient(conn)
	ctx:= context.Background()

	stream, err := client.ConnectWorker(ctx, &pb.WorkerHello{WorkerId: "2", MemoryMb: 2})
	if err != nil{
		log.Printf("[-] Error connecting to server: %v\n", err)
	}
	for{
		job, err := stream.Recv()
		if err != nil{
			log.Printf("[-] Error recieving job: %v\n", err)
			break
		}
		output, err := executeCommand(context.Background(), job)
		if err != nil{
			_, err := client.CompleteJob(context.Background(), &pb.JobResult{
																	JobId: job.Id, 
																	Success: false,
																	Output: output,})
			if err != nil{
				log.Printf("[-] Error sending job result to server")
			} else{
				log.Printf("[+] Sent job result to server")
			}
		} else{
			_, err := client.CompleteJob(context.Background(), &pb.JobResult{
																	JobId: job.Id, 
																	Success: true,
																	Output: output,})
			if err != nil{
				log.Printf("[-] Error sending job result to server")
			} else{
				log.Printf("[+] Sent job result to server")
			}
		}
	}
}

