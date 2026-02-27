package cmd

import (
	"os"
	"io"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"encoding/json"
	"encoding/base64"
	"github.com/spf13/cobra"

	pb "github.com/dhaval314/epoch/proto"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/pkg/stdcopy"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var caCert string
var cert string
var key string
var target string

var WorkerId string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "worker",
	Short: "Connect to the server",
	Long: `Connect to the server`,
	Run : connectWorker,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVarP(&target, "target", "t", "127.0.0.1:50051", "Server IP")
	rootCmd.PersistentFlags().StringVarP(&caCert, "ca-cert", "r", "certs/ca-cert.pem", "Root cert.pem file path")
	rootCmd.PersistentFlags().StringVarP(&cert, "cert", "e", "certs/worker-cert.pem", "Worker cert.pem file path")
	rootCmd.PersistentFlags().StringVarP(&key, "key", "k", "certs/worker-key.pem", "Worker key.pem file path")
	
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.Flags().StringVarP(&WorkerId, "worker-id", "i", "0", "Specify the worker id")
}

func executeCommand(ctx context.Context, req *pb.Job)(string, error){

	// NOTE: client.NewClientWithOpts is Deprecated, but the new version (client.New()) doesnt work because of dependency issues
	// Create client 
	apiClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err!= nil{
		log.Printf("[-] Error creating client: %v\n", err)
		return "", err
	}
	defer apiClient.Close()
	
	encoded_auth := ""
	log.Printf("[*] Received Auth User: '%s'", req.RegistryUsername)
	// Check whether the client wants to pull a private image
	if req.RegistryUsername != ""{
		auth_config := registry.AuthConfig{Username: req.RegistryUsername, 
											Password: req.RegistryPassword, 
											ServerAddress: req.RegistryServer}
		
		auth_config_json, err := json.Marshal(auth_config)
		if err != nil{
			log.Printf("[-] Error marshal %v", err)
		}

		encoded_auth = base64.URLEncoding.EncodeToString(auth_config_json)
	}

	// Pull the image from dockerhub
	reader, err := apiClient.ImagePull(ctx, req.Image, image.PullOptions{RegistryAuth: encoded_auth})
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
	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, out)
	if err != nil {
		log.Println("[-] Error demultiplexing container output")
		return "", err
	}
	bodyString := stdout.String() + stderr.String()
	return bodyString, nil
}

func connectWorker(cmd *cobra.Command, args[] string){

	// Generate the certificate from the pem blocks
	cert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil{
		log.Fatalf("[-] Error reading certificates %v", err)
	}

	// Root cert
	caCert, err := os.ReadFile(caCert)
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

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil{
		log.Fatalf("[-] Error connecting to server: %v\n", err)
	}
	defer conn.Close()
	log.Println("[+] Successfully Connected to the server")

	client := pb.NewSchedulerClient(conn)
	ctx:= context.Background()

	stream, err := client.ConnectWorker(ctx, &pb.WorkerHello{WorkerId: WorkerId, MemoryMb: 2}) // Get the cmd.workerid from parsed the flag
	if err != nil{
		log.Fatalf("[-] Error connecting to server: %v\n", err)
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