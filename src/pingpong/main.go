package main

import (
	pb "apigw/protos/gen"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"pingpong/service"
)

func main() {
	grpcPort := 9090

	log.Printf("Starting gRPC server on port :%d", grpcPort)
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pingPongService := service.NewPingPongService()
	pb.RegisterPingPongServiceServer(grpcServer, pingPongService)

	// TODO: determine whether to use TLS or not

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
