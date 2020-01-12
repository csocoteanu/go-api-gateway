package main

import (
	"common/discovery/registrant"
	"common/discovery/utils"
	protos "common/svcprotos/gen"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.pingpong/service"
)

var controlPort, appBalancerPort, appLocalPort *int

func parseArgs() {
	controlPort = flag.Int("control-port", 8051, "Control port")
	appBalancerPort = flag.Int("app-balancer-port", 9090, "Application balancer port")
	appLocalPort = flag.Int("app-port", 9090, "Application local port")

	flag.Parse()
}

func main() {
	parseArgs()

	log.Printf("Starting PING-PONG server on port :%d", *appLocalPort)

	address := utils.NewGRPCAddress("", uint32(*appLocalPort))
	sock, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	controlAddress := utils.NewGRPCAddress("localhost", uint32(*controlPort))
	balancerAddress := utils.NewGRPCAddress("localhost", uint32(*appBalancerPort))
	localAddress := utils.NewGRPCAddress("localhost", uint32(*appLocalPort))

	registrant.NewRegistrantService(controlAddress, utils.RegistryAddress, utils.PingPongServiceName, balancerAddress, localAddress)

	grpcServer := grpc.NewServer()
	pingPongService := service.NewPingPongService()
	protos.RegisterPingPongServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
