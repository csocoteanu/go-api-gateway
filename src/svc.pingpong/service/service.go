package service

import (
	protos "common/svcprotos/gen"
	"context"
	"fmt"
)

type pingPongService struct{}

func NewPingPongService() protos.PingPongServiceServer {
	return &pingPongService{}
}

func (s *pingPongService) Ping(ctx context.Context, req *protos.PingPongRequest) (*protos.PingPongResponse, error) {
	return createResponse(req, protos.PingPongType_PING)
}

func (s *pingPongService) Pong(ctx context.Context, req *protos.PingPongRequest) (*protos.PingPongResponse, error) {
	return createResponse(req, protos.PingPongType_PONG)
}

func createResponse(req *protos.PingPongRequest, respType protos.PingPongType) (*protos.PingPongResponse, error) {
	note := req.GetNote()
	reqType := req.GetType()

	fmt.Printf("Received request with note=%s req type=%s\n", note, typeToString(reqType))

	respTypeString, err := getResponseType(respType)
	if err != nil {
		return nil, err
	}

	message := fmt.Sprintf("%s: %s", typeToString(respTypeString), note)

	response := protos.PingPongResponse{
		Type:    respTypeString,
		Message: message,
	}

	return &response, nil
}

func typeToString(pType protos.PingPongType) string {
	switch pType {
	case protos.PingPongType_PING:
		return "ping"
	case protos.PingPongType_PONG:
		return "pong"
	default:
		return ""
	}
}

func getResponseType(pType protos.PingPongType) (protos.PingPongType, error) {
	switch pType {
	case protos.PingPongType_PING:
		return protos.PingPongType_PONG, nil
	case protos.PingPongType_PONG:
		return protos.PingPongType_PING, nil
	default:
		return protos.PingPongType_PONG, fmt.Errorf("unknwon PingPong type=%d", int(pType))
	}
}
