package main

import (
	"common/discovery/registrant"
	protos "common/svcprotos/gen"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.pingpong/service"
)

var controlPort, appPort *int

func parseArgs() {
	controlPort = flag.Int("control-port", 8051, "Control port")
	appPort = flag.Int("app-port", 9091, "Application port")

	flag.Parse()
}

func main() {
	parseArgs()

	address := fmt.Sprintf(":%d", *appPort)

	log.Printf("Starting PING-PONG server on port :%d", *appPort)
	sock, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	registrant.NewRegistrantService("localhost", uint32(*controlPort), "pingpong", address, "localhost", 8500)

	grpcServer := grpc.NewServer()
	pingPongService := service.NewPingPongService()
	protos.RegisterPingPongServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
