package main

import (
	discovery "common/discovery/registry"
	protos "common/svcprotos/gen"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"log"
	"net/http"
)

type registerCallback func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) (err error)

var services = map[string]registerCallback{
	"pingpong": protos.RegisterPingPongServiceHandlerFromEndpoint,
	"echo":     protos.RegisterEchoServiceHandlerFromEndpoint,
}

func startProxy(httpEP string, grpcEPs []string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	fmt.Printf("Starting API GW to %s\n", httpEP)

	opts := []grpc.DialOption{grpc.WithInsecure()}

	registry := discovery.NewServiceRegistry("localhost", 8500).(*discovery.ServiceRegistry)

	// Register gRPC server endpoints
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()

	go func() {
		for {
			select {
			case req := <-registry.GetRegistered():
				fmt.Printf("Registering GRPC service=%s address=%s\n", req.ServiceName, req.ServiceAddress)

				cb, ok := services[req.ServiceName]
				if ok {
					err := cb(ctx, mux, req.ServiceAddress, opts)
					if err != nil {
						fmt.Printf("Error encountered: %s", err.Error())
					}
				}
			}
		}
	}()

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(httpEP, mux)
}

func main() {
	err := startProxy(":8080", []string{":9090", ":10100"})
	if err != nil {
		log.Fatalf("Error starting server: %+v", err)
	}
}
