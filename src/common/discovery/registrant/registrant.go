package registrant

import (
	discovery "common/discovery/protos"
	"common/discovery/utils"
	"context"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

type registrantService struct {
	controlAddress  string
	registryAddress string
	serviceName     string
	balancerAddress string
	localAddress    string
}

// NewRegistrantService creates a new registrant service instance
func NewRegistrantService(
	controlAddress string,
	registryAddress string,
	serviceName, serviceBalancerAddress, serviceLocalAddress string) discovery.RegistrantServiceServer {

	s := registrantService{
		serviceName:     serviceName,
		balancerAddress: serviceBalancerAddress,
		localAddress:    serviceLocalAddress,
		controlAddress:  controlAddress,
		registryAddress: registryAddress,
	}

	go func() {
		defer func() { log.Printf("Succesfully registered service=%s", s.serviceName) }()

		if err := s.register(); err != nil {
			log.Printf("Failed registering! err=%s", err.Error())
		} else {
			return
		}

		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			if err := s.register(); err != nil {
				log.Printf("Failed registering! err=%s", err.Error())
			} else {
				ticker.Stop()
				return
			}
		}
	}()

	go func() {
		if err := s.listenForHeartBeats(); err != nil {
			log.Fatalf("Failed starting registrant on %s", s.controlAddress)
		}
	}()

	return &s
}

// HeartBeat is the gRPC implementation of the HeartBeat method
func (s *registrantService) HeartBeat(ctx context.Context, req *discovery.HeartbeatRequest) (*discovery.HeartbeatResponse, error) {
	log.Printf("Received heartbeat (%s)....", req.Message)

	resp := discovery.HeartbeatResponse{
		Message: utils.ACK,
		Success: true,
	}

	return &resp, nil
}

func (s *registrantService) register() error {
	conn, err := grpc.Dial(s.registryAddress, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to server at %s", s.registryAddress)
	}

	var resp *discovery.RegisterResponse
	var expRetrier = retrier.New(retrier.ExponentialBackoff(4, 500*time.Millisecond), nil)

	if err := expRetrier.Run(func() error {
		grpcClient := discovery.NewRegistryServiceClient(conn)
		req := discovery.RegisterRequest{
			ControlAddress:         s.controlAddress,
			ServiceName:            s.serviceName,
			ServiceBalancerAddress: s.balancerAddress,
			ServiceLocalAddress:    s.localAddress,
		}

		resp, err = grpcClient.Register(context.Background(), &req)
		if err != nil {
			log.Printf("Error registering to %s! err=%s", s.registryAddress, err.Error())
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if resp == nil {
		return utils.ErrRegisterFailed
	}

	if !resp.Success {
		return utils.AggregateErrors(resp.Errors...)
	}

	return nil
}

func (s *registrantService) listenForHeartBeats() error {
	log.Printf("Starting registrant on address=%s", s.controlAddress)
	sock, err := net.Listen("tcp", s.controlAddress)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	discovery.RegisterRegistrantServiceServer(grpcServer, s)

	err = grpcServer.Serve(sock)
	if err != nil {
		return err
	}

	return nil
}
