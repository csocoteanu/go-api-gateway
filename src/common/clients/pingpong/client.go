package pingpong

import (
	pb "common/protos/gen"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Client interface {
	SendPing(note string) (*pb.PingPongResponse, error)
	SendPong(note string) (*pb.PingPongResponse, error)
}

type client struct {
	serverAddress string
	port          int
	conn          *grpc.ClientConn
	grpcClient    pb.PingPongServiceClient
}

func NewClient(serverAddress string, port int) (Client, error) {
	c := client{
		serverAddress: serverAddress,
		port:          port,
	}

	err := c.init()
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c *client) init() error {
	address := fmt.Sprintf("%s:%d", c.serverAddress, c.port)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to server at %s", address)
	}

	c.conn = conn
	c.grpcClient = pb.NewPingPongServiceClient(conn)

	return nil
}

func (c *client) SendPing(note string) (*pb.PingPongResponse, error) {
	ctx := context.Background()
	pType := pb.PingPongType_PING

	req := pb.PingPongRequest{
		Type: pType,
		Note: note,
	}

	return c.grpcClient.Ping(ctx, &req)
}

func (c *client) SendPong(note string) (*pb.PingPongResponse, error) {
	ctx := context.Background()
	pType := pb.PingPongType_PONG

	req := pb.PingPongRequest{
		Type: pType,
		Note: note,
	}

	return c.grpcClient.Pong(ctx, &req)
}
