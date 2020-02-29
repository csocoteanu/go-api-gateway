package main

import (
	"common"
	pb "common/protos/gen"
	"common/sidecar"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.echo/usecases"
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

	address := common.ToGRPCAddress("", uint32(*appLocalPort))
	sock, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	controlAddress := common.ToGRPCAddress("localhost", uint32(*controlPort))
	balancerAddress := common.ToGRPCAddress("localhost", uint32(*appBalancerPort))
	localAddress := common.ToGRPCAddress("localhost", uint32(*appLocalPort))

	sidecar.NewRegistrantService(controlAddress, common.RegistryAddress, common.EchoServiceName, balancerAddress, localAddress)
	grpcServer := grpc.NewServer()
	pingPongService := usecases.NewEchoService()
	pb.RegisterEchoServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
