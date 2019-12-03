package client

import (
	pb "apigw/protos/gen"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type PingPongClient interface {
	SendPing(note string) (*pb.PingPongResponse, error)
	SendPong(note string) (*pb.PingPongResponse, error)
}

type pingpongClient struct {
	serverAddress string
	port          int
	conn          *grpc.ClientConn
	grpcClient    pb.PingPongServiceClient
}

func NewPingPongClient(serverAddress string, port int) (PingPongClient, error) {
	c := pingpongClient{
		serverAddress: serverAddress,
		port:          port,
	}

	err := c.init()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *pingpongClient) init() error {
	address := fmt.Sprintf("%s:%d", c.serverAddress, c.port)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to server at %s", address)
	}

	c.conn = conn
	c.grpcClient = pb.NewPingPongServiceClient(conn)

	return nil
}

func (c *pingpongClient) SendPing(note string) (*pb.PingPongResponse, error) {
	ctx := context.Background()
	pType := pb.PingPongType_PING

	req := pb.PingPongRequest{
		Type: pType,
		Note: note,
	}

	return c.grpcClient.Ping(ctx, &req)
}

func (c *pingpongClient) SendPong(note string) (*pb.PingPongResponse, error) {
	ctx := context.Background()
	pType := pb.PingPongType_PONG

	req := pb.PingPongRequest{
		Type: pType,
		Note: note,
	}

	return c.grpcClient.Pong(ctx, &req)
}
