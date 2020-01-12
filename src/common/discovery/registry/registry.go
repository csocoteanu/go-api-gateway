package registry

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
	"sync"
	"time"
)

const maxHeartBeatRetries = 1

type healthChecker struct {
	info   registrantInfo
	ticker *time.Ticker
	quit   chan struct{}
	done   chan registrantInfo
}

type registrantInfo struct {
	serviceName            string
	serviceBalancerAddress string
	serviceLocalAddress    string
	controlAddress         string
}

type ServiceInfo struct {
	BalancerAddress string   `json:"balancer"`
	LocalAddresses  []string `json:"locals"`
}

type ServiceRegistry struct {
	hostname           string
	port               uint32
	healthCheckers     map[string][]*healthChecker
	healthCheckersLock *sync.RWMutex
	chanReq            chan discovery.RegisterRequest
	done               chan registrantInfo
}

// ActiveServicesProvider provides a list of active service names
type ActiveServicesProvider interface {
	// GetActiveServices returns a list of active services
	GetActiveServices() map[string][]ServiceInfo
}

// NewServiceRegistry creates a new service registry instance
func NewServiceRegistry(hostname string, port uint32) discovery.RegistryServiceServer {
	s := ServiceRegistry{
		hostname:           hostname,
		port:               port,
		healthCheckers:     make(map[string][]*healthChecker),
		healthCheckersLock: &sync.RWMutex{},
		chanReq:            make(chan discovery.RegisterRequest),
		done:               make(chan registrantInfo),
	}

	s.startListen()
	s.startRemoveHealthChecker()

	return &s
}

func (s *ServiceRegistry) GetRegistered() chan discovery.RegisterRequest {
	return s.chanReq
}

func (s *ServiceRegistry) Register(ctx context.Context, req *discovery.RegisterRequest) (*discovery.RegisterResponse, error) {
	if len(req.ControlAddress) == 0 ||
		len(req.ServiceName) == 0 ||
		len(req.ServiceBalancerAddress) == 0 ||
		len(req.ServiceLocalAddress) == 0 {
		return nil, utils.ErrInvalidRequest
	}

	hCheckers, ok := s.healthCheckers[req.ServiceName]
	if ok {
		for _, hChecker := range hCheckers {
			if hChecker.info.controlAddress == req.ControlAddress {
				log.Printf("Already registered service=%s address=%s", req.ServiceName, req.ControlAddress)
				return nil, utils.ErrRegistrantExists
			}
		}
	}

	hChecker := newHealthChecker(req, s.done)
	s.healthCheckers[req.ServiceName] = append(s.healthCheckers[req.ServiceName], hChecker)
	s.chanReq <- *req

	log.Printf("Succesfully registered service=%s address=%s", req.ServiceName, req.ControlAddress)

	resp := discovery.RegisterResponse{
		Message: utils.ACK,
		Success: true,
	}
	return &resp, nil
}

func (s *ServiceRegistry) Unregister(ctx context.Context, req *discovery.UnregisterRequest) (*discovery.UnregisterResponse, error) {
	if len(req.ServiceName) == 0 {
		return nil, utils.ErrInvalidRequest
	}

	log.Printf("Trying to unregister service=%s control=%s", req.ServiceName, req.ControlAddress)

	s.healthCheckersLock.RLock()
	defer s.healthCheckersLock.RUnlock()

	hCheckers, ok := s.healthCheckers[req.ServiceName]
	if !ok {
		log.Printf("Service with name=%s does not exist! Skipping...", req.ServiceName)
		return nil, utils.ErrRegistrantMissing
	}

	for _, hChecker := range hCheckers {
		if hChecker.info.controlAddress == req.ControlAddress {
			hChecker.stopHealthCheck()
		}
	}

	resp := discovery.UnregisterResponse{
		Success: true,
		Message: utils.ACK,
	}
	return &resp, nil
}

func (s *ServiceRegistry) GetActiveServices() map[string]ServiceInfo {
	return s.getServices()
}

func (s *ServiceRegistry) startListen() {
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

func (s *ServiceRegistry) startRemoveHealthChecker() {
	go func() {
		for rInfo := range s.done {
			s.healthCheckersLock.Lock()

			hCheckers, ok := s.healthCheckers[rInfo.serviceName]
			if !ok {
				log.Printf("Skipping missing service name=%s", rInfo.serviceName)
				s.healthCheckersLock.Unlock()
				continue
			}

			remaining := []*healthChecker{}
			for _, hChecker := range hCheckers {
				if hChecker.info.controlAddress == rInfo.controlAddress {
					log.Printf("Succesfully unregistered service=%s control=%s", rInfo.serviceName, rInfo.controlAddress)
				} else {
					remaining = append(remaining, hChecker)
				}
			}

			if len(remaining) == 0 {
				delete(s.healthCheckers, rInfo.serviceName)
			} else {
				s.healthCheckers[rInfo.serviceName] = remaining
			}

			s.healthCheckersLock.Unlock()
		}
	}()
}

func (r *ServiceRegistry) getServices() map[string]ServiceInfo {
	r.healthCheckersLock.RLock()
	defer r.healthCheckersLock.RUnlock()

	result := map[string]ServiceInfo{}
	for k, v := range r.healthCheckers {
		serviceInfo, ok := result[k]
		if !ok {
			serviceInfo = ServiceInfo{}
		}

		for _, rInfo := range v {
			serviceInfo.LocalAddresses = append(serviceInfo.LocalAddresses, rInfo.info.serviceLocalAddress)
			serviceInfo.BalancerAddress = rInfo.info.serviceBalancerAddress
		}

		result[k] = serviceInfo
	}

	return result
}

func newRegistrantInfo(
	controlAddress string,
	serviceName, balancerAddress, localAddress string) registrantInfo {

	ri := registrantInfo{
		controlAddress:         controlAddress,
		serviceName:            serviceName,
		serviceBalancerAddress: balancerAddress,
		serviceLocalAddress:    localAddress,
	}

	return ri
}

func newHealthChecker(req *discovery.RegisterRequest, done chan registrantInfo) *healthChecker {
	info := newRegistrantInfo(req.ControlAddress, req.ServiceName, req.ServiceBalancerAddress, req.ServiceLocalAddress)
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
			log.Printf("Quiting healthcheck for %s (%s).....", r.info.serviceName, r.info.controlAddress)
			return
		case <-r.ticker.C:
			log.Printf("Sending heartbeat for %s (%s).......", r.info.serviceName, r.info.controlAddress)
			if err := r.sendHeartBeat(); err != nil {
				retries--
				log.Printf("Error sending heartbeat to service=%s (Retries remaining=%d! err=%s",
					r.info.serviceName, retries, err.Error())
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
	conn, err := grpc.Dial(r.info.controlAddress, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to service=%s at address=%s", r.info.serviceName, r.info.controlAddress)
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
		return utils.ErrHeartBeatFailed
	}

	if !resp.Success {
		return utils.AggregateErrors(resp.Errors...)
	}

	return nil
}
