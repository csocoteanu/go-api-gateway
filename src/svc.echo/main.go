package main

import (
	"common/discovery/registrant"
	"common/discovery/utils"
	protos "common/svcprotos/gen"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.echo/service"
)

var controlPort, appBalancerPort, appLocalPort *int

func parseArgs() {
	controlPort = flag.Int("control-port", 8060, "Control port")
	appBalancerPort = flag.Int("app-balancer-port", 10010, "Application balancer port")
	appLocalPort = flag.Int("app-port", 10010, "Application port")

	flag.Parse()
}

func main() {
	parseArgs()

	log.Printf("Starting ECHO server on port :%d", *appLocalPort)

	address := utils.NewGRPCAddress("", uint32(*appLocalPort))
	sock, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	controlAddress := utils.NewGRPCAddress("localhost", uint32(*controlPort))
	balancerAddress := utils.NewGRPCAddress("localhost", uint32(*appBalancerPort))
	localAddress := utils.NewGRPCAddress("localhost", uint32(*appLocalPort))

	registrant.NewRegistrantService(controlAddress, utils.RegistryAddress, utils.EchoServiceName, balancerAddress, localAddress)
	grpcServer := grpc.NewServer()
	pingPongService := service.NewEchoService()
	protos.RegisterEchoServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
