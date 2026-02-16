package main

import (
	"context"
	"log"
	"time"
	pb "github.com/dhaval314/epoch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

)

func main(){
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil{
		log.Fatalf("[-] Error connecting to server: %v", err)
	}
	
	defer conn.Close()

	client := pb.NewSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SubmitJob(ctx, &pb.Job{Id:"1", Command: "echo Hello from Docker", Schedule: "10", Image: "alpine"})
	if err != nil{
		log.Fatalf("[-] Error sending job to server %v", err)
	}

	log.Println(response.GetMessage())
	for{
		time.Sleep(1000000000)
		log.Println(client.GetJobStatus(context.Background(), &pb.JobStatusRequest{JobId: "1"}))

	}
}

