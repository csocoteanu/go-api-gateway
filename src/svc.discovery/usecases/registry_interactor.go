package usecases

import (
	"common"
	"sync"
)

type serviceRegistryInteractor struct {
	tcpLoadBalancersLock *sync.Mutex
	tcpLoadBalancers     map[string]common.TCPLoadBalancer
	registry             common.ServiceRegistry
	serviceRegistered    chan common.RegistrantInfo
	serviceDeregistered  chan common.RegistrantInfo
	quitChan             chan struct{}
}

func NewServiceRegistryInteractor(registry common.ServiceRegistry) common.RegisterHandler {
	si := serviceRegistryInteractor{
		tcpLoadBalancersLock: &sync.Mutex{},
		tcpLoadBalancers:     make(map[string]common.TCPLoadBalancer),
		registry:             registry,
		serviceRegistered:    make(chan common.RegistrantInfo),
		serviceDeregistered:  make(chan common.RegistrantInfo),
		quitChan:             make(chan struct{}),
	}

	go si.Start()

	return &si
}

func (si serviceRegistryInteractor) OnServiceRegistered() chan common.RegistrantInfo {
	return si.serviceRegistered
}

func (si serviceRegistryInteractor) OnServiceUnregistered() chan common.RegistrantInfo {
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
