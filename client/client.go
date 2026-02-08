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
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewSchedulerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	response, err := client.SubmitJob(ctx, &pb.Job{Id:"1", Command: "ls", Schedule: "5"})
	if err != nil{
		log.Fatal(err)
	}
	log.Println(response.GetMessage())
}

