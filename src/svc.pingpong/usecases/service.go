package usecases

import (
	pb "common/protos/gen"
	"context"
	"fmt"
)

type pingPongService struct{}

func NewPingPongService() pb.PingPongServiceServer {
	return &pingPongService{}
}

func (s *pingPongService) Ping(ctx context.Context, req *pb.PingPongRequest) (*pb.PingPongResponse, error) {
	return createResponse(req, pb.PingPongType_PING)
}

func (s *pingPongService) Pong(ctx context.Context, req *pb.PingPongRequest) (*pb.PingPongResponse, error) {
	return createResponse(req, pb.PingPongType_PONG)
}

func createResponse(req *pb.PingPongRequest, respType pb.PingPongType) (*pb.PingPongResponse, error) {
	note := req.GetNote()
	reqType := req.GetType()

	fmt.Printf("Received request with note=%s req type=%s\n", note, typeToString(reqType))

	respTypeString, err := getResponseType(respType)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("%s: %s", typeToString(respTypeString), note)

	response := pb.PingPongResponse{
		Type:    respTypeString,
		Message: message,
	}

	return &response, nil
}

func typeToString(pType pb.PingPongType) string {
	switch pType {
	case pb.PingPongType_PING:
		return "ping"
	case pb.PingPongType_PONG:
		return "pong"
	default:
		return ""
	}
}

func getResponseType(pType pb.PingPongType) (pb.PingPongType, error) {
	switch pType {
	case pb.PingPongType_PING:
		return pb.PingPongType_PONG, nil
	case pb.PingPongType_PONG:
		return pb.PingPongType_PING, nil
	default:
		return pb.PingPongType_PONG, fmt.Errorf("unknwon PingPong type=%d", int(pType))
	}
}
