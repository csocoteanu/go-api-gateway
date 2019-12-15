package main

import (
	"common/discovery/registrant"
	protos "common/svcprotos/gen"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.pingpong/service"
)

func main() {
	grpcPort := 9090

	log.Printf("Starting gRPC server on port :%d", grpcPort)
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	registrantService := registrant.NewRegistrantService("localhost", 8055, "pingpong", "localhost", 8500)

	grpcServer := grpc.NewServer()
	pingPongService := service.NewPingPongService(registrantService)
	protos.RegisterPingPongServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
