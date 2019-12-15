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

const (
	serviceActive   = 0
	serviceInactive = 1
)

type healthChecker struct {
	hostname    string
	port        uint32
	serviceName string
	ticker      *time.Ticker
	quit        chan struct{}
}

type statusTracker struct {
	statuses map[string]registrantInfo
	lock     *sync.RWMutex
	addChan  chan registrantInfo
	remChan  chan registrantInfo
}

type registrantInfo struct {
	serviceName string
	status      uint8
}

type serviceRegistry struct {
	hostname           string
	port               uint32
	healthCheckers     map[string]*healthChecker
	healthCheckersLock *sync.Mutex
	tracker            *statusTracker
}

// ActiveServicesProvider provides a list of active service names
type ActiveServicesProvider interface {
	// GetActiveServices returns a list of active services
	GetActiveServices() []string
}

// NewServiceRegistry creates a new service registry instance
func NewServiceRegistry(hostname string, port uint32) discovery.RegistryServiceServer {
	s := serviceRegistry{
		hostname:           hostname,
		port:               port,
		healthCheckers:     make(map[string]*healthChecker),
		healthCheckersLock: &sync.Mutex{},
		tracker:            newStatusTracker(),
	}

	go func() {
		if err := s.listen(); err != nil {
			log.Fatalf("Failed starting service registry on %s:%d", s.hostname, s.port)
		}
	}()

	return &s
}

func (s *serviceRegistry) Register(ctx context.Context, req *discovery.RegisterRequest) (*discovery.RegisterResponse, error) {
	if req.Port == 0 || len(req.Hostname) == 0 || len(req.ServiceName) == 0 {
		return nil, utils.ErrInvalidRequest
	}

	_, ok := s.healthCheckers[req.ServiceName]
	if ok {
		log.Printf("Service with name=%s already registered! Skipping...", req.ServiceName)
		return nil, utils.ErrRegistrantExists
	}

	log.Printf("Succesfully registered service=%s", req.ServiceName)
	s.healthCheckers[req.ServiceName] = newHealthChecker(req)

	resp := discovery.RegisterResponse{
		Message: utils.ACK,
		Success: true,
	}
	return &resp, nil
}

func (s *serviceRegistry) Unregister(ctx context.Context, req *discovery.UnregisterRequest) (*discovery.UnregisterResponse, error) {
	if len(req.ServiceName) == 0 {
		return nil, utils.ErrInvalidRequest
	}

	s.healthCheckersLock.Lock()
	defer s.healthCheckersLock.Unlock()

	hChecker, ok := s.healthCheckers[req.ServiceName]
	if !ok {
		log.Printf("Service with name=%s does not exist! Skipping...", req.ServiceName)
		return nil, utils.ErrRegistrantMissing
	}

	log.Printf("Succesfully unregistered service=%s", req.ServiceName)
	delete(s.healthCheckers, req.ServiceName)
	hChecker.stopHealthCheck()

	resp := discovery.UnregisterResponse{
		Success: true,
		Message: utils.ACK,
	}
	return &resp, nil
}

func (s *serviceRegistry) GetActiveServices() []string {
	return s.tracker.getServiceNameByStatus(serviceActive)
}

func (s *serviceRegistry) listen() error {
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

func newStatusTracker() *statusTracker {
	t := statusTracker{
		statuses: make(map[string]registrantInfo),
		lock:     &sync.RWMutex{},
		addChan:  make(chan registrantInfo),
		remChan:  make(chan registrantInfo),
	}

	return &t
}

func (t *statusTracker) start() {
	for {
		select {
		case status := <-t.addChan:
			t.lock.Lock()
			t.statuses[status.serviceName] = status
			t.lock.Unlock()
		case status := <-t.remChan:
			t.lock.Lock()
			delete(t.statuses, status.serviceName)
			t.lock.Unlock()
		}
	}
}

func (t *statusTracker) getServiceNameByStatus(statusType uint8) []string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	result := []string{}
	for _, svc := range t.statuses {
		if svc.status == statusType {
			result = append(result, svc.serviceName)
		}
	}

	return result
}

func newHealthChecker(req *discovery.RegisterRequest) *healthChecker {
	r := healthChecker{
		hostname:    req.Hostname,
		serviceName: req.ServiceName,
		port:        req.Port,
		ticker:      time.NewTicker(1 * time.Minute),
		quit:        make(chan struct{}),
	}

	go r.startHealthCheck()

	return &r
}

func (r *healthChecker) startHealthCheck() {
	for {
		select {
		case <-r.quit:
			r.ticker.Stop()
			return
		case <-r.ticker.C:
			if err := r.sendHeartBeat(); err != nil {
				log.Printf("Error sending heartbeat to service=%s! err=%s", r.serviceName, err.Error())
				// TODO: set serviceInactive
			}
		}
	}
}

func (r *healthChecker) stopHealthCheck() {
	r.quit <- struct{}{}
}

func (r *healthChecker) sendHeartBeat() error {
	address := fmt.Sprintf("%s:%d", r.hostname, r.port)

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return errors.Wrapf(err, "failed connecting to service=%s at address=%s", r.serviceName, address)
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
