package usecases

import (
	pb "common/protos/gen"
	"context"
	"fmt"
)

type echoService struct{}

func NewEchoService() pb.EchoServiceServer {
	s := echoService{}

	return &s
}

func (s *echoService) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	fmt.Printf("Received request with message=%s\n", req.Message)

	resp := pb.EchoResponse{
		Message: req.Message,
	}

	return &resp, nil
}
