package discovery

import (
	"common"
	pb "common/protos/gen"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Client interface {
	GetServices() ([]*common.RegistrantInfo, error)
}

type client struct {
	serverAddress string
	port          int
	conn          *grpc.ClientConn
	grpcClient    pb.RegistryServiceClient
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
	c.grpcClient = pb.NewRegistryServiceClient(conn)

	return nil
}

func (c *client) GetServices() ([]*common.RegistrantInfo, error) {
	ctx := context.Background()
	req := &pb.GetServicesRequest{}

	resp, err := c.grpcClient.GetServices(ctx, req)
	if err != nil {
		return nil, err
	}

	registrants := []*common.RegistrantInfo{}
	for _, si := range resp.ServiceInfos {
		for _, ip := range si.ServiceLocalAddress {
			rInfo := common.RegistrantInfo{
				ServiceLocalAddress:    ip,
				ServiceName:            si.ServiceName,
				ServiceBalancerAddress: si.ServiceBalancerAddress,
			}

			registrants = append(registrants, &rInfo)
		}
	}

	return registrants, nil
}
