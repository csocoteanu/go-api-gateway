package echo

import (
	pb "common/protos/gen"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type Client interface {
	SendEcho(message string) (*pb.EchoResponse, error)
}

type client struct {
	serverAddress string
	port          int
	conn          *grpc.ClientConn
	grpcClient    pb.EchoServiceClient
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
	c.grpcClient = pb.NewEchoServiceClient(conn)

	return nil
}

func (c *client) SendEcho(message string) (*pb.EchoResponse, error) {
	ctx := context.Background()
	req := pb.EchoRequest{
		Message: message,
	}

	return c.grpcClient.Echo(ctx, &req)
}
