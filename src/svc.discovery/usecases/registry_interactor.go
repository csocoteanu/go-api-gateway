package usecases

import (
	"common/discovery/domain"
	"sync"
)

type serviceRegistryInteractor struct {
	tcpLoadBalancersLock *sync.Mutex
	tcpLoadBalancers     map[string]domain.TCPLoadBalancer
	registry             domain.ServiceRegistry
	serviceRegistered    chan domain.RegistrantInfo
	serviceDeregistered  chan domain.RegistrantInfo
	quitChan             chan struct{}
}

func NewServiceRegistryInteractor(registry domain.ServiceRegistry) domain.RegisterHandler {
	si := serviceRegistryInteractor{
		tcpLoadBalancersLock: &sync.Mutex{},
		tcpLoadBalancers:     make(map[string]domain.TCPLoadBalancer),
		registry:             registry,
		serviceRegistered:    make(chan domain.RegistrantInfo),
		serviceDeregistered:  make(chan domain.RegistrantInfo),
		quitChan:             make(chan struct{}),
	}

	go si.Start()

	return &si
}

func (si serviceRegistryInteractor) OnServiceRegistered() chan domain.RegistrantInfo {
	return si.serviceRegistered
}

func (si serviceRegistryInteractor) OnServiceUnregistered() chan domain.RegistrantInfo {
	return si.serviceDeregistered
}

func (si serviceRegistryInteractor) Start() {
	for {
		select {
		case <-si.quitChan:
			return
		case rInfo := <-si.serviceRegistered:
			si.tcpLoadBalancersLock.Lock()
			lb, ok := si.tcpLoadBalancers[rInfo.ServiceName]
			if !ok {
				lb = NewTcpLoadBalancer(rInfo.ServiceBalancerAddress)
				si.tcpLoadBalancers[rInfo.ServiceName] = lb
			}
			lb.AddProxy(rInfo.ServiceLocalAddress)
			si.tcpLoadBalancersLock.Unlock()
			break
		case rInfo := <-si.serviceDeregistered:
			si.tcpLoadBalancersLock.Lock()
			lb, ok := si.tcpLoadBalancers[rInfo.ServiceName]
			if ok {
				lb.RemoveProxy(rInfo.ServiceLocalAddress)
				proxies := lb.GetProxies()
				if len(proxies) == 0 {
					delete(si.tcpLoadBalancers, rInfo.ServiceName)
				}
			}
			si.tcpLoadBalancersLock.Unlock()
			break
		}
	}
}

func (si serviceRegistryInteractor) Stop() {
	si.quitChan <- struct{}{}
}
