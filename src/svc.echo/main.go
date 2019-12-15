package main

import (
	protos "common/svcprotos/gen"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.echo/service"
)

func main() {
	grpcPort := 9090

	log.Printf("Starting gRPC server on port :%d", grpcPort)
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	echoService := service.NewEchoService()

	protos.RegisterEchoServiceServer(grpcServer, echoService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
