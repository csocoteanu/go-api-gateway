package main

import (
	protos "common/svcprotos/gen"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
)

func main() {
	serverAddress := "localhost"
	port := 9090

	address := fmt.Sprintf("%s:%d", serverAddress, port)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed connecting to server at %s! ERR=%s", address, err.Error())
	}

	client := protos.NewEchoServiceClient(conn)
	ctx := context.Background()
	req := protos.EchoRequest{Message: "Hello"}

	resp, err := client.Echo(ctx, &req)
	if err != nil {
		log.Fatalf("Failed receiving message from %s! ERR=%s", address, err.Error())
	}

	log.Printf("Received message: %s", resp.Message)
}
