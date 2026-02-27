package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var submit = &cobra.Command{
	Use:   "submit -i <image_name> -c <command> -s <time in seconds>",
	Short: "Submit a job to the server",
	Long: `Submit a job which includes the docker image and a command`,
	Run : submitJob,
}

func init(){
	rootCmd.AddCommand(submit)
	
	submit.Flags().StringP("command", "c","test","Command")
	submit.Flags().StringP("image", "i","alpine","Docker image")
	submit.Flags().StringP("schedule", "s","100","Time interval to execute the image in (in seconds)")

	submit.Flags().String("registry-user", "", "registry username")
	submit.Flags().String("registry-pass", "", "registry password")
	submit.Flags().String("registry-url", "", "docker.io")

}

func submitJob(cmd *cobra.Command, args []string){
	command, _ := cmd.Flags().GetString("command")
	image, _ := cmd.Flags().GetString("image")
	schedule, _ := cmd.Flags().GetString("schedule")
	
	registry_user, _ := cmd.Flags().GetString("registry-user")
	registry_pass, _ := cmd.Flags().GetString("registry-pass")
	registry_url, _ := cmd.Flags().GetString("registry-url")
	
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

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))) // Gets the target from global flags

	if err != nil{
		log.Fatalf("[-] Error connecting to server: %v", err)
	}
	
	defer conn.Close()

	client := pb.NewSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SubmitJob(ctx, &pb.Job{Id:strconv.Itoa(rand.Int()), 
													Command: command, 
													Schedule: schedule, 
													Image: image,
													RegistryUsername: registry_user,
													RegistryPassword: registry_pass,
													RegistryServer: registry_url,})
	if err != nil{
		log.Fatalf("[-] Error sending job to server %v", err)
	}
	log.Println(response.GetMessage(), response.GetId())
}