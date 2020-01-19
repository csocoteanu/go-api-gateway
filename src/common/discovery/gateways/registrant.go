package gateways

import (
	"common/discovery/domain"
	discovery "common/discovery/domain/protos"
	"context"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
	"time"
)

type registrantService struct {
	controlAddress  string
	registryAddress string
	serviceName     string
	balancerAddress string
	localAddress    string
	lastUpdatedTime time.Time
	connectedLock   *sync.Mutex
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
		connectedLock:   &sync.Mutex{},
	}

	go s.connectToRegistry()

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

	s.setUpdatedTime()

	resp := discovery.HeartbeatResponse{
		Message: domain.ACK,
		Success: true,
	}

	return &resp, nil
}

func (s *registrantService) register() error {
	log.Printf("Registering to service: %s", s.registryAddress)

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
		return domain.ErrRegisterFailed
	}
	if !resp.Success {
		return domain.AggregateErrors(resp.Errors...)
	}

	s.setUpdatedTime()

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

func (s *registrantService) connectToRegistry() {
	defer func() { log.Printf("Succesfully registered service=%s", s.serviceName) }()

	if err := s.register(); err != nil {
		log.Printf("Failed registering! err=%s", err.Error())
	}

	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		register := false

		s.connectedLock.Lock()
		duration := time.Now().Sub(s.lastUpdatedTime)
		if duration.Minutes() > 1 {
			register = true
		}
		s.connectedLock.Unlock()

		if register {
			if err := s.register(); err != nil {
				log.Printf("Failed registering! err=%s", err.Error())
			}
		}
	}
}

func (s *registrantService) setUpdatedTime() {
	s.connectedLock.Lock()
	s.lastUpdatedTime = time.Now()
	s.connectedLock.Unlock()
}
