package main

import (
	"log"
	"net"

	"github.com/abeardevil/together-engine/pb"
	"google.golang.org/grpc"
)

type togetherServer struct {
	pb.UnimplementedGameServiceServer
}

var GRPCServer togetherServer

func (s togetherServer) Connect(req *pb.ConnectRequest, stream pb.GameService_ConnectServer) error {
	log.Printf("Got ConnectRequest from %s.\n", req.Username)

	return nil
}

func startServer() {
	log.Println("Starting GRPC server...")

	lis, err := net.Listen("tcp", "localhost:9000")

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)

	pb.RegisterGameServiceServer(grpcServer, GRPCServer)

	grpcServer.Serve(lis)
}
