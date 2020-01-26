package usecases

import (
	"common/discovery/domain"
	"github.com/pkg/errors"
	"io"
	"log"
	"net"
	"sync"
)

// InstanceCount counts the number of instances
var InstanceCount = 0
var Backends = map[int][]string{
	0: {"localhost:10011", "localhost:10012", "localhost:10013"},
	1: {"localhost9091", "localhost:9092", "localhost:9093"},
}

type tcpLoadBalancer struct {
	lbAddress         string
	proxyAddresses    map[string]struct{}
	stopChan          chan struct{}
	currentProxyIndex int
	proxyLock         *sync.Mutex
	instanceID        int
}

func NewTcpLoadBalancer(lbAddress string) domain.TCPLoadBalancer {
	lb := tcpLoadBalancer{
		lbAddress:      lbAddress,
		proxyLock:      &sync.Mutex{},
		proxyAddresses: make(map[string]struct{}),
		stopChan:       make(chan struct{}),
		instanceID:     InstanceCount,
	}

	InstanceCount++

	return &lb
}

func (lb *tcpLoadBalancer) start() {
	listener, err := net.Listen("tcp", lb.lbAddress)
	if err != nil {
		panic(errors.Wrapf(err, "failed starting %s", lb.lbAddress))
	}

	log.Printf("Started TCP load balancer: %s", lb.lbAddress)

	for {
		select {
		case <-lb.stopChan:
			log.Printf("Closing TCP load balancer: %s", lb.lbAddress)
			err := listener.Close()
			if err != nil {
				panic(errors.Wrapf(err, "failed stopping %s", lb.lbAddress))
			}
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed handling connection! err=%s", err.Error())
				continue
			}

			go func(source net.Conn) {
				proxyAddress := lb.getNextProxy()
				destination, err := net.Dial("tcp", proxyAddress)
				if err != nil {
					log.Printf("Failed connecting to proxy=%s from=%s! err=%s", proxyAddress, lb.lbAddress, err.Error())
					return
				}

				go lb.copyIO(source, destination)
				go lb.copyIO(destination, source)
			}(conn)
		}
	}
}

func (lb *tcpLoadBalancer) stop() {
	lb.stopChan <- struct{}{}
}

func (lb *tcpLoadBalancer) AddProxy(proxy string) {
	log.Printf("Adding proxy: %s", proxy)

	lb.proxyLock.Lock()
	lb.proxyAddresses[proxy] = struct{}{}
	if len(lb.proxyAddresses) == 1 {
		go lb.start()
	}
	lb.proxyLock.Unlock()
}

func (lb *tcpLoadBalancer) RemoveProxy(proxy string) {
	log.Printf("Removing proxy: %s", proxy)

	lb.proxyLock.Lock()
	delete(lb.proxyAddresses, proxy)
	if len(lb.proxyAddresses) == 0 {
		lb.stop()
	}
	lb.proxyLock.Unlock()
}

func (lb *tcpLoadBalancer) GetProxies() []string {
	proxies := []string{}
	lb.proxyLock.Lock()
	for proxy := range lb.proxyAddresses {
		proxies = append(proxies, proxy)
	}
	lb.proxyLock.Unlock()

	return proxies
}

func (lb *tcpLoadBalancer) getNextProxy() string {
	lb.proxyLock.Lock()
	defer lb.proxyLock.Unlock()

	currentIndex := 0
	lbBackends := Backends[lb.instanceID]

	for _, proxyAddress := range lbBackends {
		if currentIndex == lb.currentProxyIndex {
			lb.currentProxyIndex = (lb.currentProxyIndex + 1) % len(lbBackends)

			return proxyAddress
		}

		currentIndex++
	}

	return ""
}

func (lb *tcpLoadBalancer) copyIO(src, dest net.Conn) {
	defer src.Close()
	defer dest.Close()
	io.Copy(src, dest)
}
