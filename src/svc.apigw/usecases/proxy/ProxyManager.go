package proxy

import (
	"common/discovery/domain"
	protos "common/svcprotos/gen"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"log"
)

type registerCallback func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) (err error)

type ProxyManager struct {
	serverMux         *runtime.ServeMux
	added             chan domain.RegistrantInfo
	removed           chan domain.RegistrantInfo
	grpcOpts          []grpc.DialOption
	registerCallbacks map[string]registerCallback
}

func NewProxyManager() *ProxyManager {
	callbacks := map[string]registerCallback{
		domain.PingPongServiceName: protos.RegisterPingPongServiceHandlerFromEndpoint,
		domain.EchoServiceName:     protos.RegisterEchoServiceHandlerFromEndpoint,
	}
	opts := []grpc.DialOption{grpc.WithInsecure()}

	m := ProxyManager{
		serverMux:         runtime.NewServeMux(),
		grpcOpts:          opts,
		registerCallbacks: callbacks,
		added:             make(chan domain.RegistrantInfo),
		removed:           make(chan domain.RegistrantInfo),
	}

	m.listenForRegistrants()

	return &m
}

func (m *ProxyManager) OnServiceRegistered() chan domain.RegistrantInfo {
	return m.added
}

func (m *ProxyManager) OnServiceUnregistered() chan domain.RegistrantInfo {
	return m.removed
}

func (m *ProxyManager) GetServerMux() *runtime.ServeMux {
	return m.serverMux
}

func (m *ProxyManager) listenForRegistrants() {
	go func() {
		for {
			select {
			case registrant := <-m.added:
				fmt.Printf("Registering GRPC service=%s address=%s\n", registrant.ServiceName, registrant.ServiceBalancerAddress)

				callback, ok := m.registerCallbacks[registrant.ServiceName]
				if !ok {
					log.Printf("No service register handler for %s", registrant.ServiceName)
				}

				err := callback(context.Background(), m.serverMux, registrant.ServiceBalancerAddress, m.grpcOpts)
				if err != nil {
					fmt.Printf("Error encountered: %s", err.Error())
				}
			case <-m.removed:
			}
		}
	}()
}
