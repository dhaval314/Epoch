package main

import (
	"net"
	"log"
	"context"
	"google.golang.org/grpc"
	pb "github.com/dhaval314/epoch/proto"
)

type server struct{
	pb.UnimplementedSchedulerServer
}

func (s *server) SubmitJob(ctx context.Context, req *pb.Job) (*pb.JobResponse, error){
	log.Println(req)
	return &pb.JobResponse{Success: true, Message: "Job Accepted by the server"}, nil
}

func main(){
	lis, err := net.Listen("tcp", ":50051")
	if err != nil{
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSchedulerServer(grpcServer, &server{})
	if err:= grpcServer.Serve(lis); err != nil{
		log.Fatal(err)
	}
}

