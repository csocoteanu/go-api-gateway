package main

import (
	"common"
	pb "common/protos/gen"
	"common/sidecar"
	"flag"
	"google.golang.org/grpc"
	"log"
	"net"
	"svc.pingpong/usecases"
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

	address := common.ToGRPCAddress("", uint32(*appLocalPort))
	sock, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	controlAddress := common.ToGRPCAddress("localhost", uint32(*controlPort))
	balancerAddress := common.ToGRPCAddress("localhost", uint32(*appBalancerPort))
	localAddress := common.ToGRPCAddress("localhost", uint32(*appLocalPort))

	sidecar.NewRegistrantService(controlAddress, common.RegistryAddress, common.PingPongServiceName, balancerAddress, localAddress)

	grpcServer := grpc.NewServer()
	pingPongService := usecases.NewPingPongService()
	pb.RegisterPingPongServiceServer(grpcServer, pingPongService)

	err = grpcServer.Serve(sock)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
