package main

import (
	pb "apigw/protos/gen"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"log"
	"net/http"
)

func startProxy(httpEP string, grpcEPs []string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Printf("Starting API GW to %s\n", httpEP)

	opts := []grpc.DialOption{grpc.WithInsecure()}

	// Register gRPC server endpoints
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()

	for _, grpcEP := range grpcEPs {
		fmt.Printf("Registering GRPC server address=%s\n", grpcEP)

		err := pb.RegisterPingPongServiceHandlerFromEndpoint(ctx, mux, grpcEP, opts)
		if err != nil {
			return err
		}
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(httpEP, mux)
}

func main() {
	err := startProxy(":8080", []string{":9090", ":10100"})
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
