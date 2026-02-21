package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"

	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var status = &cobra.Command{
	Use:   "status -j <job id>",
	Short: "Get the status of the job",
	Long: `Get the status of the job`,
	Run : getStatus,
}

func init(){
	rootCmd.AddCommand(status)
	
	status.Flags().StringP("client-id", "j","0","Job id")
	
}

func getStatus(cmd *cobra.Command, args[]string) {
	id, _ := cmd.Flags().GetString("client-id")

	// NOTE: this code block for initializing the connection is repeated in submit.go and status.go
	// Generate the certificate from the pem blocks
	cert, err := tls.LoadX509KeyPair(cert, key) // Gets the cert and key from global flags from root.go
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
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok { // Gets the caCert from global flags
        log.Fatalln("[-] Could not append cert to pool")
    }
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs: caCertPool, // The Server used ClientCAs to verify incoming clients. The Client/Worker uses RootCAs to verify the destination server.
	}

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))  // Gets the target from global flags

	if err != nil{
		log.Fatalf("[-] Error connecting to server: %v", err)
	}
	
	defer conn.Close()

	client := pb.NewSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.GetJobStatus(ctx, &pb.JobStatusRequest{JobId: id})
	if err != nil{
		log.Printf("[-] Error getting job status: %v", err)
	}
	log.Println(resp)
}