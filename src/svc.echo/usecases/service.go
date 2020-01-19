package usecases

import (
	protos "common/svcprotos/gen"
	"context"
	"fmt"
)

type echoService struct{}

func NewEchoService() protos.EchoServiceServer {
	s := echoService{}

	return &s
}

func (s *echoService) Echo(ctx context.Context, req *protos.EchoRequest) (*protos.EchoResponse, error) {
	fmt.Printf("Received request with message=%s\n", req.Message)

	resp := protos.EchoResponse{
		Message: req.Message,
	}

	return &resp, nil
}
