package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"time"

	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	
)

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
		log.Fatalf("[-] Error connecting to server: %v", err)
	}
	
	defer conn.Close()

	client := pb.NewSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SubmitJob(ctx, &pb.Job{Id:"1", Command: "nope", Schedule: "15", Image: "alpine"})
	if err != nil{
		log.Fatalf("[-] Error sending job to server %v", err)
	}

	log.Println(response.GetMessage())
	// for{
	// 	time.Sleep(1000000000)
	// 	log.Println(client.GetJobStatus(context.Background(), &pb.JobStatusRequest{JobId: "1"}))

	// }
}

