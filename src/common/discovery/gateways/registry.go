package gateways

import (
	"common/discovery/domain"
	discovery "common/discovery/domain/protos"
	"context"
	"fmt"
	"github.com/eapache/go-resiliency/retrier"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
	"time"
)

const maxHeartBeatRetries = 1

type healthChecker struct {
	info   domain.RegistrantInfo
	ticker *time.Ticker
	quit   chan struct{}
	done   chan domain.RegistrantInfo
}

type serviceRegistry struct {
	hostname           string
	port               uint32
	healthCheckers     map[string][]*healthChecker
	healthCheckersLock *sync.RWMutex
	healthCheckerExit  chan domain.RegistrantInfo
	handlers           []domain.RegisterHandler
}

// NewServiceRegistryServer creates a new service registry instance
func NewServiceRegistryServer(hostname string, port uint32) domain.ServiceRegistry {
	s := serviceRegistry{
		hostname:           hostname,
		port:               port,
		healthCheckers:     make(map[string][]*healthChecker),
		healthCheckersLock: &sync.RWMutex{},
		healthCheckerExit:  make(chan domain.RegistrantInfo),
	}

	s.startListen()
	s.startRemoveHealthChecker()

	return &s
}

func (s *serviceRegistry) Register(ctx context.Context, req *discovery.RegisterRequest) (*discovery.RegisterResponse, error) {
	if len(req.ControlAddress) == 0 ||
		len(req.ServiceName) == 0 ||
		len(req.ServiceBalancerAddress) == 0 ||
		len(req.ServiceLocalAddress) == 0 {
		return nil, domain.ErrInvalidRequest
	}

	rInfo := domain.NewRegistrantInfo(req.ControlAddress, req.ServiceName, req.ServiceBalancerAddress, req.ServiceLocalAddress)
	err := s.Load(rInfo)
	if err != nil {
		return nil, err
	}

	resp := discovery.RegisterResponse{
		Message: domain.ACK,
		Success: true,
	}
	return &resp, nil
}

func (s *serviceRegistry) Unregister(ctx context.Context, req *discovery.UnregisterRequest) (*discovery.UnregisterResponse, error) {
	if len(req.ServiceName) == 0 {
		return nil, domain.ErrInvalidRequest
	}

	log.Printf("Trying to unregister service=%s control=%s", req.ServiceName, req.ControlAddress)

	s.healthCheckersLock.RLock()
	defer s.healthCheckersLock.RUnlock()

	hCheckers, ok := s.healthCheckers[req.ServiceName]
	if !ok {
		log.Printf("Service with name=%s does not exist! Skipping...", req.ServiceName)
		return nil, domain.ErrRegistrantMissing
	}

	for _, hChecker := range hCheckers {
		if hChecker.info.ControlAddress == req.ControlAddress {
			hChecker.stopHealthCheck()
		}
	}

	resp := discovery.UnregisterResponse{
		Success: true,
		Message: domain.ACK,
	}
	return &resp, nil
}

func (s *serviceRegistry) GetServices(ctx context.Context, req *discovery.GetServicesRequest) (*discovery.ServicesResponse, error) {
	s.healthCheckersLock.RLock()
	defer s.healthCheckersLock.RUnlock()

	result := &discovery.ServicesResponse{}

	for _, hCheckers := range s.healthCheckers {
		serviceInfo := &discovery.ServiceInfo{}

		for _, hChecker := range hCheckers {
			serviceInfo.ServiceName = hChecker.info.ServiceName
			serviceInfo.ServiceBalancerAddress = hChecker.info.ServiceBalancerAddress
			serviceInfo.ServiceLocalAddress = append(serviceInfo.ServiceLocalAddress, hChecker.info.ServiceLocalAddress)
		}

		result.ServiceInfos = append(result.ServiceInfos, serviceInfo)
	}

	return result, nil
}

func (s *serviceRegistry) RegisterHandler(h domain.RegisterHandler) {
	s.handlers = append(s.handlers, h)
}

func (s *serviceRegistry) Load(rInfos ...domain.RegistrantInfo) error {
	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	for _, rInfo := range rInfos {
		hCheckers, ok := s.healthCheckers[rInfo.ServiceName]
		if ok {
			for _, hChecker := range hCheckers {
				if hChecker.info.ControlAddress == rInfo.ControlAddress {
					log.Printf("Already registered service=%s address=%s", rInfo.ServiceName, rInfo.ControlAddress)
					return domain.ErrRegistrantExists
				}
			}
		}

		hChecker := newHealthChecker(rInfo, s.healthCheckerExit)
		s.healthCheckers[rInfo.ServiceName] = append(s.healthCheckers[rInfo.ServiceName], hChecker)

		for _, handler := range s.handlers {
			handler.OnServiceRegistered() <- rInfo
		}

		log.Printf("Succesfully registered service=%s address=%s", rInfo.ServiceName, rInfo.ControlAddress)
	}

	return nil
}

func (s *serviceRegistry) startListen() {
	go func() {
		log.Printf("Starting registry on port=%d", s.port)
		sock, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.hostname, s.port))
		if err != nil {
			log.Fatalf("Failed starting service registry on %s:%d! err=%+v", s.hostname, s.port, err)
		}

		grpcServer := grpc.NewServer()
		discovery.RegisterRegistryServiceServer(grpcServer, s)

		err = grpcServer.Serve(sock)
		if err != nil {
			log.Fatalf("Failed starting gRPC registry on %s:%d! err=%+v", s.hostname, s.port, err)
		}
	}()
}

func (s *serviceRegistry) startRemoveHealthChecker() {
	go func() {
		for rInfo := range s.healthCheckerExit {
			s.removeHealthChecker(rInfo)

			for _, handler := range s.handlers {
				handler.OnServiceUnregistered() <- rInfo
			}
		}
	}()
}

func (s *serviceRegistry) removeHealthChecker(rInfo domain.RegistrantInfo) {
	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	hCheckers, ok := s.healthCheckers[rInfo.ServiceName]
	if !ok {
		log.Printf("Skipping missing service name=%s", rInfo.ServiceName)
		return
	}

	remaining := []*healthChecker{}
	for _, hChecker := range hCheckers {
		if hChecker.info.ControlAddress == rInfo.ControlAddress {
			log.Printf("Succesfully unregistered service=%s control=%s", rInfo.ServiceName, rInfo.ControlAddress)
		} else {
			remaining = append(remaining, hChecker)
		}
	}

	if len(remaining) == 0 {
		delete(s.healthCheckers, rInfo.ServiceName)
	} else {
		s.healthCheckers[rInfo.ServiceName] = remaining
	}
}

func newHealthChecker(info domain.RegistrantInfo, done chan domain.RegistrantInfo) *healthChecker {
	r := healthChecker{
		info: info,
		quit: make(chan struct{}),
		done: done,
	}

	go r.startHealthCheck()

	return &r
}

func (r *healthChecker) startHealthCheck() {
	r.ticker = time.NewTicker(10 * time.Second)
	retries := maxHeartBeatRetries

	defer func() {
		r.ticker.Stop()
		r.done <- r.info
	}()

	for retries > 0 {
		select {
		case <-r.quit:
			log.Printf("Quiting healthcheck for %s (%s).....", r.info.ServiceName, r.info.ControlAddress)
			return
		case <-r.ticker.C:
			log.Printf("Sending heartbeat for %s (%s).......", r.info.ServiceName, r.info.ControlAddress)
			if err := r.sendHeartBeat(); err != nil {
				retries--
				log.Printf("Error sending heartbeat to service=%s (Retries remaining=%d! err=%s",
					r.info.ServiceName, retries, err.Error())
				return
			} else {
				retries = maxHeartBeatRetries
			}
		}
	}
}

func (r *healthChecker) stopHealthCheck() {
	r.quit <- struct{}{}
}

func (r *healthChecker) sendHeartBeat() error {
	conn, err := grpc.Dial(r.info.ControlAddress, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to service=%s at address=%s", r.info.ServiceName, r.info.ControlAddress)
	}

	var resp *discovery.HeartbeatResponse
	var expRetrier = retrier.New(retrier.ExponentialBackoff(4, 500*time.Millisecond), nil)

	if err := expRetrier.Run(func() error {
		grpcClient := discovery.NewRegistrantServiceClient(conn)
		req := discovery.HeartbeatRequest{}

		resp, err = grpcClient.HeartBeat(context.Background(), &req)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if resp == nil {
		return domain.ErrHeartBeatFailed
	}

	if !resp.Success {
		return domain.AggregateErrors(resp.Errors...)
	}

	return nil
}
