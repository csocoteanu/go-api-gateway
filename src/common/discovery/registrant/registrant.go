package registrant

import (
	discovery "common/discovery/protos"
	"common/discovery/utils"
	"context"
	"fmt"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

type registrantService struct {
	hostname         string
	port             uint32
	serviceName      string
	registryHostname string
	registryPort     uint32
}

// NewRegistrantService creates a new registrant service instance
func NewRegistrantService(
	hostname string, port uint32,
	serviceName string,
	registryHostname string, registryPort uint32) discovery.RegistrantServiceServer {

	s := registrantService{
		hostname:         hostname,
		port:             port,
		serviceName:      serviceName,
		registryHostname: registryHostname,
		registryPort:     registryPort,
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
			log.Fatalf("Failed starting registrant on %s:%d", s.hostname, s.port)
		}
	}()

	return &s
}

// HeartBeat is the gRPC implementation of the HeartBeat method
func (s *registrantService) HeartBeat(ctx context.Context, req *discovery.HeartbeatRequest) (*discovery.HeartbeatResponse, error) {
	resp := discovery.HeartbeatResponse{
		Message: utils.ACK,
		Success: true,
	}

	return &resp, nil
}

func (s *registrantService) register() error {
	address := fmt.Sprintf("%s:%d", s.registryHostname, s.registryPort)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to server at %s", address)
	}

	var resp *discovery.RegisterResponse
	var expRetrier = retrier.New(retrier.ExponentialBackoff(4, 500*time.Millisecond), nil)

	if err := expRetrier.Run(func() error {
		grpcClient := discovery.NewRegistryServiceClient(conn)
		req := discovery.RegisterRequest{
			Hostname:    s.hostname,
			Port:        s.port,
			ServiceName: s.serviceName,
		}

		resp, err = grpcClient.Register(context.Background(), &req)
		if err != nil {
			log.Printf("Error registering to %s! err=%s", s.registryHostname, err.Error())
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
	log.Printf("Starting registrant on port=%d", s.port)
	sock, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.hostname, s.port))
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
