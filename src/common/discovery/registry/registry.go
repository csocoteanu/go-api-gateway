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
	info     registrantInfo
	ticker   *time.Ticker
	quit     chan struct{}
	tracker  *serviceTracker
	registry *ServiceRegistry
}

type serviceTracker struct {
	services map[string][]registrantInfo
	lock     *sync.RWMutex
	addChan  chan registrantInfo
	remChan  chan registrantInfo
}

type registrantInfo struct {
	serviceName            string
	serviceBalancerAddress string
	serviceLocalAddress    string
	controlAddress         string
}

type ServiceInfo struct {
	BalancerAddress string   `json:"balancer_address"`
	LocalAddresses  []string `json:"local_addresses"`
}

type ServiceRegistry struct {
	hostname           string
	port               uint32
	healthCheckers     map[string][]*healthChecker
	healthCheckersLock *sync.Mutex
	tracker            *serviceTracker
	chanReq            chan discovery.RegisterRequest
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
		healthCheckersLock: &sync.Mutex{},
		tracker:            newServiceTracker(),
		chanReq:            make(chan discovery.RegisterRequest),
	}

	go func() {
		if err := s.listen(); err != nil {
			log.Fatalf("Failed starting service registry on %s:%d", s.hostname, s.port)
		}
	}()

	go s.tracker.start()

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

	s.healthCheckers[req.ServiceName] = append(s.healthCheckers[req.ServiceName], newHealthChecker(req, s.tracker, s))
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

	err := s.removeHealthChecker(req.ServiceName, req.ControlAddress)
	if err != nil {
		return nil, err
	}

	resp := discovery.UnregisterResponse{
		Success: true,
		Message: utils.ACK,
	}
	return &resp, nil
}

func (s *ServiceRegistry) GetActiveServices() map[string]ServiceInfo {
	return s.tracker.getServices()
}

func (s *ServiceRegistry) listen() error {
	log.Printf("Starting registry on port=%d", s.port)
	sock, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.hostname, s.port))
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	discovery.RegisterRegistryServiceServer(grpcServer, s)

	err = grpcServer.Serve(sock)
	if err != nil {
		return err
	}

	return nil
}

func (s *ServiceRegistry) removeHealthChecker(serviceName, controlAddress string) error {
	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	hCheckers, ok := s.healthCheckers[serviceName]
	if !ok {
		log.Printf("Service with name=%s does not exist! Skipping...", serviceName)
		return utils.ErrRegistrantMissing
	}

	remaining := []*healthChecker{}
	for _, hChecker := range hCheckers {
		if hChecker.info.controlAddress == controlAddress {
			hChecker.stopHealthCheck()
			log.Printf("Succesfully unregistered service=%s hostname=%s", serviceName, controlAddress)
		} else {
			remaining = append(remaining, hChecker)
		}
	}

	if len(remaining) == 0 {
		delete(s.healthCheckers, serviceName)
	} else {
		s.healthCheckers[serviceName] = remaining
	}

	return nil
}

func newServiceTracker() *serviceTracker {
	t := serviceTracker{
		services: make(map[string][]registrantInfo),
		lock:     &sync.RWMutex{},
		addChan:  make(chan registrantInfo),
		remChan:  make(chan registrantInfo),
	}

	return &t
}

func (t *serviceTracker) start() {
	for {
		select {
		case rInfo := <-t.addChan:
			t.lock.Lock()
			t.services[rInfo.serviceName] = append(t.services[rInfo.serviceName], rInfo)
			t.lock.Unlock()
		case rInfo := <-t.remChan:
			t.lock.Lock()
			infos, ok := t.services[rInfo.serviceName]
			if ok {
				remaining := []registrantInfo{}
				for _, info := range infos {
					if info.controlAddress != rInfo.controlAddress {
						remaining = append(remaining, info)
					}
				}

				if len(remaining) == 0 {
					delete(t.services, rInfo.serviceName)
				} else {
					t.services[rInfo.serviceName] = remaining
				}
			}
			t.lock.Unlock()
		}
	}
}

func (t *serviceTracker) getServices() map[string]ServiceInfo {
	t.lock.RLock()
	defer t.lock.RUnlock()

	result := map[string]ServiceInfo{}
	for k, v := range t.services {
		serviceInfo, ok := result[k]
		if !ok {
			serviceInfo = ServiceInfo{}
		}

		for _, rInfo := range v {
			serviceInfo.LocalAddresses = append(serviceInfo.LocalAddresses, rInfo.serviceLocalAddress)
			serviceInfo.BalancerAddress = rInfo.serviceBalancerAddress
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

func newHealthChecker(req *discovery.RegisterRequest, tracker *serviceTracker, registry *ServiceRegistry) *healthChecker {
	info := newRegistrantInfo(req.ControlAddress, req.ServiceName, req.ServiceBalancerAddress, req.ServiceLocalAddress)
	r := healthChecker{
		info:     info,
		quit:     make(chan struct{}),
		tracker:  tracker,
		registry: registry,
	}

	go r.startHealthCheck()

	return &r
}

func (r *healthChecker) startHealthCheck() {
	r.ticker = time.NewTicker(20 * time.Second)
	retries := maxHeartBeatRetries

	r.tracker.addChan <- r.info
	defer func() {
		r.ticker.Stop()
		r.tracker.remChan <- r.info
		err := r.registry.removeHealthChecker(r.info.serviceName, r.info.controlAddress)
		if err != nil {
			log.Printf("Error removing registrant service=%s (%s)", r.info.serviceName, r.info.controlAddress)
		}
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
