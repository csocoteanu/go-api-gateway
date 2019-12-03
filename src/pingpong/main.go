package main

import (
	pb "apigw/protos/gen"
	"context"
	"fmt"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
	"path"
	"pingpong/service"
	"strings"
)

func run(grpcEP string, httpEP string) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	dir := "./pb/"

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := http.NewServeMux()
	/*opts := []grpc.DialOption{grpc.WithInsecure()}

	err := pb.RegisterPingPongServiceHandlerFromEndpoint(ctx, mux, grpcEP, opts)
	if err != nil {
		return err
	}*/

	mux.HandleFunc("/swagger/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			glog.Errorf("Not Found: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(dir, p)
		glog.Infof("Serving %s [%s]", r.URL.Path, p)

		http.ServeFile(w, r, p)
	})

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(httpEP, mux)
}

func main() {
	grpcPort := 9090
	httpPort := 8080

	go func() {
		log.Printf("Starting gRPC server on port :%d", grpcPort)
		sock, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		grpcServer := grpc.NewServer()
		pingPongService := service.NewPingPongService()
		pb.RegisterPingPongServiceServer(grpcServer, pingPongService)

		// TODO: determine whether to use TLS or not

		err = grpcServer.Serve(sock)
		if err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	log.Printf("Starting HTTP server on port :%d", httpPort)
	err := run(fmt.Sprintf(":%d", grpcPort), fmt.Sprintf(":%d", httpPort))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
